package cloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
	"github.com/anytty/anytty/shared/transport/datachannel"
)

type cloudPeerAttempt struct {
	preference     cloudv1.RelayPreference
	icePolicy      port.ICETransportPolicy
	relayTransport endpoint.RelayTransport
}

const cloudPeerCleanupTimeout = 5 * time.Second

type cloudSignalLifecycle interface {
	Done() <-chan struct{}
	Err() error
}

type cloudPeerReadyWaiter interface {
	WaitReady(context.Context) error
}

var errCloudSignalingEndedDuringPeerSetup = errors.New("Cloud signaling ended during peer setup")

type openedCloudPeer struct {
	peer              port.WebRTCPeer
	signaling         *cloudclient.SignalSession
	connection        transport.Transport
	path              endpoint.Path
	icePolicy         port.ICETransportPolicy
	relayTCPAvailable bool
	closeOnce         sync.Once
	closeErr          error
}

func openResolvedCloudPeer(
	ctx context.Context,
	request clientruntime.AttemptRequest,
	peers PeerFactory,
	cloud *cloudclient.Client,
	resolved *cloudclient.RouteResolution,
	identity remoteauth.ClientAccessIdentity,
	signer cloudclient.Signer,
	product cloudv1.ClientProduct,
	report func(clientruntime.EndpointPhase),
) (*openedCloudPeer, error) {
	attempt, err := planCloudPeerAttempt(request.Route().RelayMode)
	if err != nil {
		return nil, err
	}
	return openResolvedCloudPeerAttempt(ctx, request, peers, cloud, resolved, identity, signer, product, report, attempt)
}

func planCloudPeerAttempt(mode endpoint.RelayMode) (cloudPeerAttempt, error) {
	switch mode {
	case "", endpoint.RelayAuto, endpoint.RelaySmart:
		return cloudPeerAttempt{preference: cloudv1.RelayPreference_RELAY_PREFERENCE_AUTO, icePolicy: port.ICETransportAll}, nil
	case endpoint.RelayDirect:
		return cloudPeerAttempt{preference: cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY, icePolicy: port.ICETransportAll}, nil
	case endpoint.RelayOnly:
		return cloudPeerAttempt{preference: cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY, icePolicy: port.ICETransportRelayOnly}, nil
	default:
		return cloudPeerAttempt{}, fmt.Errorf("unsupported Cloud relay mode %q", mode)
	}
}

func planCloudPeerTransportFallback(route endpoint.AccessRoute, err error) (cloudPeerAttempt, bool) {
	var handshakeErr *remoteauth.HandshakeError
	if !errors.As(err, &handshakeErr) ||
		handshakeErr.Code != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL ||
		!errors.Is(handshakeErr.Cause, io.EOF) {
		return cloudPeerAttempt{}, false
	}
	switch handshakeErr.Detail {
	case "receive remote auth frame", "send remote auth frame":
		return planCloudRelayTCPFallback(route)
	default:
		return cloudPeerAttempt{}, false
	}
}

func planCloudPeerPostAuthFallback(
	ctx context.Context,
	route endpoint.AccessRoute,
	path endpoint.Path,
	relayTCPAvailable bool,
	err error,
) (cloudPeerAttempt, bool) {
	if ctx == nil || ctx.Err() != nil || path != endpoint.PathDirect || !relayTCPAvailable {
		return cloudPeerAttempt{}, false
	}
	if !errors.Is(err, io.EOF) &&
		!errors.Is(err, io.ErrUnexpectedEOF) &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded) {
		return cloudPeerAttempt{}, false
	}
	return planCloudRelayTCPFallback(route)
}

func planCloudRelayTCPFallback(route endpoint.AccessRoute) (cloudPeerAttempt, bool) {
	switch route.RelayMode {
	case "", endpoint.RelayAuto, endpoint.RelaySmart:
	default:
		return cloudPeerAttempt{}, false
	}
	if route.RelayTransport == endpoint.RelayTransportUDP {
		return cloudPeerAttempt{}, false
	}
	return cloudPeerAttempt{
		preference:     cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY,
		icePolicy:      port.ICETransportRelayOnly,
		relayTransport: endpoint.RelayTransportTCP,
	}, true
}

func openResolvedCloudPeerAttempt(
	ctx context.Context,
	request clientruntime.AttemptRequest,
	peers PeerFactory,
	cloud *cloudclient.Client,
	resolved *cloudclient.RouteResolution,
	identity remoteauth.ClientAccessIdentity,
	signer cloudclient.Signer,
	product cloudv1.ClientProduct,
	report func(clientruntime.EndpointPhase),
	attempt cloudPeerAttempt,
) (*openedCloudPeer, error) {
	startedAt := time.Now()
	lastAt := startedAt
	reportTiming := func(stage string) {
		now := time.Now()
		log.Printf("anytty cloud connect generation=%d stage=%s stage_ms=%d total_ms=%d", request.Stamp().Generation, stage, now.Sub(lastAt).Milliseconds(), now.Sub(startedAt).Milliseconds())
		lastAt = now
	}
	route := request.Route()
	var peer port.WebRTCPeer
	var relayTCPAvailable bool
	closePeer := func() {
		if peer != nil {
			_ = peer.Close()
		}
	}
	if report != nil {
		report(clientruntime.EndpointPhaseConnecting)
	}
	signalSession, err := cloud.Exchange(ctx, resolved, identity, signer, product, uint64(request.Stamp().Generation), attempt.preference, func(ctx context.Context, ready *cloudv1.ClientReady) (string, error) {
		peerConfig := port.WebRTCConfig{Policy: attempt.icePolicy}
		if relay := ready.GetRelay(); relay != nil {
			tcpURLs, tcpErr := filterManagedICEURLs(relay.GetUrls(), endpoint.RelayTransportTCP)
			relayTCPAvailable = tcpErr == nil && hasManagedTURNServer([]port.ICEServer{{URLs: tcpURLs}})
			relayTransport := route.RelayTransport
			if attempt.relayTransport != "" {
				relayTransport = attempt.relayTransport
			}
			urls, filterErr := filterManagedICEURLs(relay.GetUrls(), relayTransport)
			if filterErr != nil {
				return "", filterErr
			}
			if len(urls) > 0 {
				peerConfig.Servers = append(peerConfig.Servers, port.ICEServer{URLs: urls, Username: relay.GetUsername(), Credential: relay.GetCredential()})
			}
		}
		if attempt.icePolicy == port.ICETransportRelayOnly && !hasManagedTURNServer(peerConfig.Servers) {
			return "", errors.New("Cloud Relay-only attempt did not receive TURN material")
		}
		openedPeer, openErr := peers.OpenCloudPeer(ctx, peerConfig)
		if openErr != nil {
			return "", fmt.Errorf("create Cloud WebRTC peer: %w", openErr)
		}
		peer = openedPeer
		reportTiming("webrtc_peer_open")
		if peer.Channel() == nil {
			closePeer()
			return "", errors.New("Cloud WebRTC peer has no protocol DataChannel")
		}
		offer, offerErr := peer.CreateOffer(ctx)
		return offer, offerErr
	})
	if err != nil {
		closePeer()
		return nil, err
	}
	reportTiming("signaling_answer_ready")
	opened := &openedCloudPeer{peer: peer, signaling: signalSession, icePolicy: attempt.icePolicy, relayTCPAvailable: relayTCPAvailable}
	peerContext, stopPeerContext := cloudPeerSetupContext(ctx, signalSession)
	defer stopPeerContext()
	if signalErr, ended := cloudSignalTerminalError(signalSession); ended {
		_ = opened.Close()
		return nil, signalErr
	}
	answer := signalSession.Answer()
	candidates := make([]port.ICECandidate, 0, len(answer.GetCandidates()))
	for _, candidate := range answer.GetCandidates() {
		if candidate != nil {
			candidates = append(candidates, port.ICECandidate{Candidate: candidate.GetCandidate(), SDPMid: candidate.GetSdpMid(), SDPMLineIndex: candidate.GetSdpMlineIndex(), UsernameFragment: candidate.GetUsernameFragment()})
		}
	}
	if err := peer.ApplyAnswer(peerContext, answer.GetAnswerSdp(), candidates); err != nil {
		if signalErr, ended := cloudSignalTerminalError(signalSession); ended {
			_ = opened.Close()
			return nil, signalErr
		}
		return nil, errors.Join(fmt.Errorf("apply Cloud WebRTC answer: %w", err), abandonOpenedCloudPeer(ctx, opened))
	}
	reportTiming("webrtc_answer_applied")
	if err := waitCloudPeerReady(peerContext, peer, signalSession); err != nil {
		if cloudclient.IsAdminDisconnect(err) {
			_ = opened.Close()
			return nil, err
		}
		return nil, errors.Join(fmt.Errorf("wait Cloud WebRTC DataChannel: %w", err), abandonOpenedCloudPeer(ctx, opened))
	}
	reportTiming("datachannel_ready")
	opened.path, err = observeCloudPeerPath(peer, attempt.icePolicy)
	if err != nil {
		return nil, errors.Join(err, abandonOpenedCloudPeer(ctx, opened))
	}
	log.Printf("anytty cloud connect generation=%d stage=path_selected path=%s", request.Stamp().Generation, opened.path)
	opened.connection = datachannel.New(peer.Channel())
	return opened, nil
}

func cloudPeerSetupContext(parent context.Context, signaling cloudSignalLifecycle) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	if signaling == nil {
		return ctx, cancel
	}
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		select {
		case <-signaling.Done():
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() {
		cancel()
		<-watchDone
	}
}

func waitCloudPeerReady(ctx context.Context, peer cloudPeerReadyWaiter, signaling cloudSignalLifecycle) error {
	if signalErr, ended := cloudSignalTerminalError(signaling); ended {
		return signalErr
	}
	err := peer.WaitReady(ctx)
	if signalErr, ended := cloudSignalTerminalError(signaling); ended {
		return signalErr
	}
	return err
}

func cloudSignalTerminalError(signaling cloudSignalLifecycle) (error, bool) {
	if signaling == nil {
		return nil, false
	}
	select {
	case <-signaling.Done():
		if err := signaling.Err(); err != nil {
			return err, true
		}
		return errCloudSignalingEndedDuringPeerSetup, true
	default:
		return nil, false
	}
}

func (opened *openedCloudPeer) Transport() transport.Transport {
	if opened == nil {
		return nil
	}
	return opened.connection
}

func (opened *openedCloudPeer) RemoteCertificateFingerprint() (string, error) {
	if opened == nil || opened.peer == nil {
		return "", errors.New("Cloud peer is unavailable")
	}
	return opened.peer.RemoteCertificateFingerprint()
}

func (opened *openedCloudPeer) ObservedPath() endpoint.Path {
	if opened == nil {
		return ""
	}
	if current, err := observeCloudPeerPath(opened.peer, opened.icePolicy); err == nil {
		return current
	}
	return opened.path
}

func observeCloudPeerPath(peer interface{ ObservedPath() endpoint.Path }, policy port.ICETransportPolicy) (endpoint.Path, error) {
	if peer == nil {
		return "", errors.New("Cloud peer is unavailable")
	}
	path := peer.ObservedPath()
	if path != endpoint.PathDirect && path != endpoint.PathSingleRelay || policy == port.ICETransportRelayOnly && path != endpoint.PathSingleRelay {
		return "", fmt.Errorf("Cloud connector established a path that violates Relay policy: %q", path)
	}
	return path, nil
}

func (opened *openedCloudPeer) ConfirmAuthenticatedPath(ctx context.Context) (endpoint.Path, error) {
	if opened == nil || opened.signaling == nil {
		return "", errors.New("Cloud signaling session is unavailable")
	}
	if ctx == nil {
		return "", errors.New("Cloud path confirmation context is required")
	}
	currentPath, err := observeCloudPeerPath(opened.peer, opened.icePolicy)
	if err != nil {
		return "", err
	}
	opened.path = currentPath
	selectedPath := cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT
	if opened.path == endpoint.PathSingleRelay {
		selectedPath = cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_RELAY
	} else if opened.path != endpoint.PathDirect {
		return "", fmt.Errorf("Cloud connector has invalid selected path %q", opened.path)
	}
	if signalErr, ended := cloudSignalTerminalError(opened.signaling); ended {
		return "", signalErr
	}
	if err := opened.signaling.ConfirmPath(ctx, selectedPath); err != nil {
		if signalErr, ended := cloudSignalTerminalError(opened.signaling); ended {
			return "", signalErr
		}
		return "", fmt.Errorf("confirm authenticated Cloud path: %w", err)
	}
	return opened.path, nil
}

func (opened *openedCloudPeer) Release() (port.WebRTCPeer, *cloudclient.SignalSession) {
	if opened == nil {
		return nil, nil
	}
	peer, signaling := opened.peer, opened.signaling
	opened.peer, opened.signaling, opened.connection = nil, nil, nil
	return peer, signaling
}

func (opened *openedCloudPeer) Close() error {
	return opened.close(nil, false)
}

func (opened *openedCloudPeer) AbandonAndWait(ctx context.Context) error {
	if ctx == nil {
		return errors.New("Cloud peer abandonment context is required")
	}
	return opened.close(ctx, false)
}

func (opened *openedCloudPeer) ReleaseAndWait(ctx context.Context) error {
	if ctx == nil {
		return errors.New("Cloud peer release context is required")
	}
	return opened.close(ctx, true)
}

func (opened *openedCloudPeer) close(waitContext context.Context, releaseConfirmed bool) error {
	if opened == nil {
		return nil
	}
	opened.closeOnce.Do(func() {
		if opened.connection != nil {
			opened.closeErr = opened.connection.Close()
		}
		if opened.peer != nil {
			opened.closeErr = errors.Join(opened.closeErr, opened.peer.Close())
		}
		if opened.signaling != nil {
			if waitContext != nil {
				if releaseConfirmed {
					opened.closeErr = errors.Join(opened.closeErr, opened.signaling.ReleaseAndWait(waitContext))
				} else {
					opened.closeErr = errors.Join(opened.closeErr, opened.signaling.AbandonAndWait(waitContext))
				}
			} else {
				opened.closeErr = errors.Join(opened.closeErr, opened.signaling.Close())
			}
		}
	})
	return opened.closeErr
}

func abandonOpenedCloudPeer(parent context.Context, opened *openedCloudPeer) error {
	if opened == nil {
		return nil
	}
	if parent == nil || parent.Err() != nil {
		return opened.Close()
	}
	if _, ended := cloudSignalTerminalError(opened.signaling); ended {
		return opened.Close()
	}
	cleanupContext, cancelCleanup := context.WithTimeout(context.WithoutCancel(parent), cloudPeerCleanupTimeout)
	defer cancelCleanup()
	return opened.AbandonAndWait(cleanupContext)
}

var _ clientruntime.PairingPeerSession = (*openedCloudPeer)(nil)
