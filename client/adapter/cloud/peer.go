package cloud

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
	"github.com/anytty/anytty/shared/transport/datachannel"
)

type cloudPeerAttempt struct {
	preference     cloudv1.RelayPreference
	icePolicy      port.ICETransportPolicy
	relayTransport endpoint.RelayTransport
}

type cloudSignalLifecycle interface {
	Done() <-chan struct{}
	Err() error
}

type cloudPeerReadyWaiter interface {
	WaitReady(context.Context) error
}

var errCloudSignalingEndedDuringPeerSetup = errors.New("Cloud signaling ended during peer setup")

type openedCloudPeer struct {
	peer       port.WebRTCPeer
	signaling  *cloudclient.SignalSession
	connection transport.Transport
	path       endpoint.Path
	closeOnce  sync.Once
	closeErr   error
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
	if ctx == nil {
		return nil, errors.New("Cloud peer attempt context is required")
	}
	attempts, err := planCloudPeerAttempts(request.Route().RelayMode, request.Route().RelayTransport)
	if err != nil {
		return nil, err
	}
	if len(attempts) == 1 {
		return openResolvedCloudPeerAttempt(ctx, request, peers, cloud, resolved, identity, signer, product, report, attempts[0])
	}
	return raceCloudPeerAttempts(ctx, request, peers, cloud, resolved, identity, signer, product, report, attempts)
}

func relayTransportOptions(value endpoint.RelayTransport) ([]endpoint.RelayTransport, error) {
	switch value {
	case "", endpoint.RelayTransportAuto:
		return []endpoint.RelayTransport{endpoint.RelayTransportTCP, endpoint.RelayTransportUDP}, nil
	case endpoint.RelayTransportUDP, endpoint.RelayTransportTCP:
		return []endpoint.RelayTransport{value}, nil
	default:
		return nil, fmt.Errorf("unsupported Cloud relay transport %q", value)
	}
}

func planCloudPeerAttempts(mode endpoint.RelayMode, relayTransport endpoint.RelayTransport) ([]cloudPeerAttempt, error) {
	transportOptions, err := relayTransportOptions(relayTransport)
	if err != nil {
		return nil, err
	}
	switch mode {
	case "", endpoint.RelayAuto, endpoint.RelaySmart:
		attempts := []cloudPeerAttempt{{
			preference: cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY,
			icePolicy:  port.ICETransportAll,
		}}
		for _, value := range transportOptions {
			attempts = append(attempts, cloudPeerAttempt{
				preference:     cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY,
				icePolicy:      port.ICETransportRelayOnly,
				relayTransport: value,
			})
		}
		return attempts, nil
	case endpoint.RelayDirect:
		return []cloudPeerAttempt{{
			preference: cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY,
			icePolicy:  port.ICETransportAll,
		}}, nil
	case endpoint.RelayOnly:
		attempts := make([]cloudPeerAttempt, 0, len(transportOptions))
		for _, value := range transportOptions {
			attempts = append(attempts, cloudPeerAttempt{
				preference:     cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY,
				icePolicy:      port.ICETransportRelayOnly,
				relayTransport: value,
			})
		}
		return attempts, nil
	default:
		return nil, fmt.Errorf("unsupported Cloud relay mode %q", mode)
	}
}

func relayTransportProto(value endpoint.RelayTransport) cloudv1.RelayTransport {
	switch value {
	case endpoint.RelayTransportUDP:
		return cloudv1.RelayTransport_RELAY_TRANSPORT_UDP
	case endpoint.RelayTransportTCP:
		return cloudv1.RelayTransport_RELAY_TRANSPORT_TCP
	default:
		return cloudv1.RelayTransport_RELAY_TRANSPORT_UNSPECIFIED
	}
}

func (attempt cloudPeerAttempt) label() string {
	if attempt.preference == cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY {
		return "Direct"
	}
	return fmt.Sprintf("Relay-%s", strings.ToUpper(string(attempt.relayTransport)))
}

type cloudPeerAttemptResult struct {
	index  int
	opened *openedCloudPeer
	err    error
}

func raceCloudPeerAttempts(
	ctx context.Context,
	request clientruntime.AttemptRequest,
	peers PeerFactory,
	cloud *cloudclient.Client,
	resolved *cloudclient.RouteResolution,
	identity remoteauth.ClientAccessIdentity,
	signer cloudclient.Signer,
	product cloudv1.ClientProduct,
	report func(clientruntime.EndpointPhase),
	attempts []cloudPeerAttempt,
) (*openedCloudPeer, error) {
	if len(attempts) == 0 {
		return nil, errors.New("Cloud peer attempts are required")
	}
	if ctx == nil {
		return nil, errors.New("Cloud peer attempt context is required")
	}
	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan cloudPeerAttemptResult, len(attempts))
	for index, attempt := range attempts {
		go func(index int, attempt cloudPeerAttempt) {
			opened, err := openResolvedCloudPeerAttempt(raceCtx, request, peers, cloud, resolved, identity, signer, product, report, attempt)
			results <- cloudPeerAttemptResult{index: index, opened: opened, err: err}
		}(index, attempt)
	}

	orderedErrors := make([]error, len(attempts))
	var winner *openedCloudPeer
	for remaining := len(attempts); remaining > 0; remaining-- {
		result := <-results
		if result.err == nil && result.opened != nil && winner == nil {
			winner = result.opened
			cancel()
			continue
		}
		if result.opened != nil {
			_ = result.opened.Close()
		}
		if result.err == nil {
			result.err = errors.New("Cloud peer attempt returned no peer")
		}
		orderedErrors[result.index] = fmt.Errorf("Cloud %s attempt: %w", attempts[result.index].label(), result.err)
	}
	if winner != nil {
		return winner, nil
	}
	if err := context.Cause(ctx); err != nil {
		return nil, err
	}
	return nil, errors.Join(orderedErrors...)
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
	var peer port.WebRTCPeer
	closePeer := func() {
		if peer != nil {
			_ = peer.Close()
		}
	}
	if report != nil {
		report(clientruntime.EndpointPhaseConnecting)
	}
	clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseSignaling, clientruntime.EndpointStageSignaling)
	signalSession, err := cloud.Exchange(ctx, resolved, identity, signer, product, uint64(request.Stamp().Generation), attempt.preference, relayTransportProto(attempt.relayTransport), func(ctx context.Context, ready *cloudv1.ClientReady) (string, error) {
		peerConfig := port.WebRTCConfig{Policy: attempt.icePolicy}
		if relay := ready.GetRelay(); relay != nil && attempt.preference != cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY {
			urls, filterErr := filterManagedICEURLs(relay.GetUrls(), attempt.relayTransport)
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
		clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseConnecting, clientruntime.EndpointStagePeerOpening)
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
		clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseConnecting, clientruntime.EndpointStageICEGathering)
		offer, offerErr := peer.CreateOffer(ctx)
		return offer, offerErr
	})
	if err != nil {
		closePeer()
		return nil, err
	}
	reportTiming("signaling_answer_ready")
	opened := &openedCloudPeer{peer: peer, signaling: signalSession}
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
	if attempt.icePolicy == port.ICETransportRelayOnly {
		clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseConnecting, clientruntime.EndpointStageCloudRelayAttempt)
	} else {
		clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseConnecting, clientruntime.EndpointStageCloudP2PAttempt)
	}
	if err := peer.ApplyAnswer(peerContext, answer.GetAnswerSdp(), candidates); err != nil {
		if signalErr, ended := cloudSignalTerminalError(signalSession); ended {
			_ = opened.Close()
			return nil, signalErr
		}
		_ = opened.Close()
		return nil, fmt.Errorf("apply Cloud WebRTC answer: %w", err)
	}
	reportTiming("webrtc_answer_applied")
	if err := waitCloudPeerReady(peerContext, peer, signalSession); err != nil {
		_ = opened.Close()
		if cloudclient.IsAdminDisconnect(err) {
			return nil, err
		}
		return nil, fmt.Errorf("wait Cloud WebRTC DataChannel: %w", err)
	}
	reportTiming("datachannel_ready")
	opened.path = peer.ObservedPath()
	if opened.path != endpoint.PathDirect && opened.path != endpoint.PathSingleRelay || attempt.icePolicy == port.ICETransportRelayOnly && opened.path != endpoint.PathSingleRelay {
		_ = opened.Close()
		return nil, fmt.Errorf("Cloud connector established a path that violates Relay policy: %q", opened.path)
	}
	selectedPath := cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT
	if opened.path == endpoint.PathSingleRelay {
		selectedPath = cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_RELAY
		if attempt.icePolicy == port.ICETransportRelayOnly {
			clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseConnecting, clientruntime.EndpointStageCloudRelaySelected)
		} else {
			clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseConnecting, clientruntime.EndpointStageCloudRelayFallback)
		}
	} else {
		clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseConnecting, clientruntime.EndpointStageCloudDirectSelected)
	}
	if signalErr, ended := cloudSignalTerminalError(signalSession); ended {
		_ = opened.Close()
		return nil, signalErr
	}
	if err := signalSession.ConfirmPath(peerContext, selectedPath); err != nil {
		if signalErr, ended := cloudSignalTerminalError(signalSession); ended {
			_ = opened.Close()
			return nil, signalErr
		}
		_ = opened.Close()
		return nil, fmt.Errorf("confirm selected Cloud path: %w", err)
	}
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
	return opened.path
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
	if opened == nil {
		return nil
	}
	opened.closeOnce.Do(func() {
		opened.closeErr = errors.Join(opened.closeErr, releaseCloudSession(opened.signaling))
		if opened.connection != nil {
			opened.closeErr = errors.Join(opened.closeErr, opened.connection.Close())
		}
		if opened.peer != nil {
			opened.closeErr = errors.Join(opened.closeErr, opened.peer.Close())
		}
		if opened.signaling != nil {
			opened.closeErr = errors.Join(opened.closeErr, opened.signaling.Close())
		}
	})
	return opened.closeErr
}

var _ clientruntime.PairingPeerSession = (*openedCloudPeer)(nil)
