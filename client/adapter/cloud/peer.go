package cloud

import (
	"context"
	"errors"
	"fmt"
	"log"
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

const autoDirectReadyTimeout = 3 * time.Second

type cloudPeerAttempt struct {
	preference   cloudv1.RelayPreference
	icePolicy    port.ICETransportPolicy
	readyTimeout time.Duration
}

type cloudPeerConnectivityError struct{ err error }

func (failure *cloudPeerConnectivityError) Error() string { return failure.err.Error() }
func (failure *cloudPeerConnectivityError) Unwrap() error { return failure.err }

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
	attempts, err := cloudPeerAttempts(request.Route().RelayMode)
	if err != nil {
		return nil, err
	}
	return runCloudPeerAttempts(ctx, attempts, func(attempt cloudPeerAttempt) (*openedCloudPeer, error) {
		return openResolvedCloudPeerAttempt(ctx, request, peers, cloud, resolved, identity, signer, product, report, attempt)
	}, func() {
		log.Printf("anytty cloud connect generation=%d stage=direct_fallback", request.Stamp().Generation)
	})
}

func runCloudPeerAttempts(ctx context.Context, attempts []cloudPeerAttempt, open func(cloudPeerAttempt) (*openedCloudPeer, error), onFallback func()) (*openedCloudPeer, error) {
	if len(attempts) == 0 || open == nil {
		return nil, errors.New("Cloud peer attempt plan is empty")
	}
	var directErr error
	for index, attempt := range attempts {
		opened, attemptErr := open(attempt)
		if attemptErr == nil {
			return opened, nil
		}
		if index == 0 && len(attempts) > 1 && ctx.Err() == nil && isCloudPeerConnectivityError(attemptErr) {
			directErr = attemptErr
			if onFallback != nil {
				onFallback()
			}
			continue
		}
		if directErr != nil {
			return nil, fmt.Errorf("Cloud Direct attempt failed (%v); Relay fallback failed: %w", directErr, attemptErr)
		}
		return nil, attemptErr
	}
	return nil, errors.New("Cloud peer attempt plan did not produce a result")
}

func cloudPeerAttempts(mode endpoint.RelayMode) ([]cloudPeerAttempt, error) {
	switch mode {
	case "", endpoint.RelayAuto, endpoint.RelaySmart:
		return []cloudPeerAttempt{
			{preference: cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY, icePolicy: port.ICETransportAll, readyTimeout: autoDirectReadyTimeout},
			{preference: cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY, icePolicy: port.ICETransportRelayOnly},
		}, nil
	case endpoint.RelayDirect:
		return []cloudPeerAttempt{{preference: cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY, icePolicy: port.ICETransportAll}}, nil
	case endpoint.RelayOnly:
		return []cloudPeerAttempt{{preference: cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY, icePolicy: port.ICETransportRelayOnly}}, nil
	default:
		return nil, fmt.Errorf("unsupported Cloud relay mode %q", mode)
	}
}

func isCloudPeerConnectivityError(err error) bool {
	var failure *cloudPeerConnectivityError
	return errors.As(err, &failure)
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
			urls, filterErr := filterManagedICEURLs(relay.GetUrls(), route.RelayTransport)
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
		if offerErr != nil && attempt.preference == cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY {
			return "", &cloudPeerConnectivityError{err: offerErr}
		}
		return offer, offerErr
	})
	if err != nil {
		closePeer()
		return nil, err
	}
	reportTiming("signaling_answer_ready")
	opened := &openedCloudPeer{peer: peer, signaling: signalSession}
	answer := signalSession.Answer()
	candidates := make([]port.ICECandidate, 0, len(answer.GetCandidates()))
	for _, candidate := range answer.GetCandidates() {
		if candidate != nil {
			candidates = append(candidates, port.ICECandidate{Candidate: candidate.GetCandidate(), SDPMid: candidate.GetSdpMid(), SDPMLineIndex: candidate.GetSdpMlineIndex(), UsernameFragment: candidate.GetUsernameFragment()})
		}
	}
	if err := peer.ApplyAnswer(ctx, answer.GetAnswerSdp(), candidates); err != nil {
		_ = opened.Close()
		return nil, &cloudPeerConnectivityError{err: fmt.Errorf("apply Cloud WebRTC answer: %w", err)}
	}
	reportTiming("webrtc_answer_applied")
	readyContext := ctx
	cancelReady := func() {}
	if attempt.readyTimeout > 0 {
		readyContext, cancelReady = context.WithTimeout(ctx, attempt.readyTimeout)
	}
	err = peer.WaitReady(readyContext)
	cancelReady()
	if err != nil {
		_ = opened.Close()
		return nil, &cloudPeerConnectivityError{err: fmt.Errorf("wait Cloud WebRTC DataChannel: %w", err)}
	}
	reportTiming("datachannel_ready")
	opened.path = peer.ObservedPath()
	if opened.path != endpoint.PathDirect && opened.path != endpoint.PathSingleRelay || attempt.icePolicy == port.ICETransportRelayOnly && opened.path != endpoint.PathSingleRelay {
		_ = opened.Close()
		return nil, &cloudPeerConnectivityError{err: fmt.Errorf("Cloud connector established a path that violates Relay policy: %q", opened.path)}
	}
	selectedPath := cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT
	if opened.path == endpoint.PathSingleRelay {
		selectedPath = cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_RELAY
	}
	if err := signalSession.ConfirmPath(selectedPath); err != nil {
		_ = opened.Close()
		return nil, fmt.Errorf("confirm selected Cloud path: %w", err)
	}
	opened.connection = datachannel.New(peer.Channel())
	return opened, nil
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
		if opened.connection != nil {
			opened.closeErr = opened.connection.Close()
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
