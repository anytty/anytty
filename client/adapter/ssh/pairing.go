package ssh

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anytty/anytty/client/adapter/direct"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/shared/remoteauth"
)

// PairingConnector 通过 SSH direct-tcpip 到达 daemon 的 loopback signaling/ICE，并复用公共 PairingService。
// SSH 只负责可达性和 host key pin；DeviceIdentity、claim 与 grant 仍在端到端 DTLS DataChannel 内校验和签发。
type PairingConnector struct {
	Peers          direct.PeerFactory
	Credentials    port.SSHCredentialSource
	ConnectTimeout time.Duration
	NetworkDialer  direct.ContextDialer
	Now            func() time.Time
}

// Redeem 只处理 request 指定的 SSH Route，完成后无论成功失败都会关闭 SSH client 与 ICE forwarder。
func (connector *PairingConnector) Redeem(ctx context.Context, request clientruntime.AttemptRequest, pairing remoteauth.ClientPairingRequest) (result remoteauth.PairingExchangeResult, err error) {
	if connector == nil || connector.Peers == nil || connector.Credentials == nil {
		return result, fmt.Errorf("SSH pairing connector dependencies are incomplete")
	}
	if err := clientruntime.ValidatePairingAttempt(request, pairing); err != nil {
		return result, err
	}
	route := request.Route()
	if route.Kind != endpoint.RouteSSHWebRTCTCP || len(route.HostKeyFingerprints) == 0 {
		return result, fmt.Errorf("SSH pairing Route requires pinned host keys")
	}
	credential, err := connector.Credentials.ResolveSSHCredential(ctx, route.SSHCredentialRef, route.CredentialDescriptor)
	if err != nil {
		return result, fmt.Errorf("resolve SSH pairing credential: %w", err)
	}
	timeout := connector.ConnectTimeout
	if timeout <= 0 {
		timeout = defaultConnectTimeout
	}
	sshClient, dialErr := dialSSHClient(ctx, route, credential.AuthMethods, timeout, connector.NetworkDialer)
	releaseErr := credential.Release()
	if dialErr != nil || releaseErr != nil {
		if sshClient != nil {
			_ = sshClient.Close()
		}
		return result, errors.Join(dialErr, releaseErr)
	}
	forwarder, err := newICEForwarder(sshClient, route.RemoteICETCPAddress)
	if err != nil {
		_ = sshClient.Close()
		return result, err
	}
	defer func() {
		err = errors.Join(err, forwarder.Close(), sshClient.Close())
	}()
	return (&direct.PairingConnector{
		Peers: connector.Peers, Signaling: direct.TCPSignalingClient{Dialer: sshClientDialer{client: sshClient}},
		RouteKind: endpoint.RouteSSHWebRTCTCP, Locators: []string{route.RemoteSignalingAddress}, TransformAnswer: forwarder.projectAnswer,
		Now: connector.Now,
	}).Redeem(ctx, request, pairing)
}
