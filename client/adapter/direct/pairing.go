package direct

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
)

// PairingConnector 通过 daemon embedded signaling 与 ICE-TCP 建立一次性 PairingExchange peer。
// 它不执行 capability auth 或 Hello；PairingService 成功或失败后会精确关闭当前 DataChannel/peer。
type PairingConnector struct {
	Peers           PeerFactory
	Signaling       SignalingClient
	RouteKind       endpoint.RouteKind
	Locators        []string
	TransformAnswer func(*remoteauthpb.DirectSignalingAnswerV2) (*remoteauthpb.DirectSignalingAnswerV2, error)
	Random          io.Reader
	Now             func() time.Time
	Phase           func(clientruntime.EndpointPhase)
}

// Redeem 使用当前 Direct attempt 的 Endpoint pin 与实际 DTLS certificate 兑换 PairingTicket。
// embedded signaling 只携带 claim digest、客户端公钥投影和绝对期限；claim body 与 signer 仍只进入 DataChannel pairing 状态机。
func (connector *PairingConnector) Redeem(ctx context.Context, request clientruntime.AttemptRequest, pairing remoteauth.ClientPairingRequest) (remoteauth.PairingExchangeResult, error) {
	if err := clientruntime.ValidatePairingAttempt(request, pairing); err != nil {
		return remoteauth.PairingExchangeResult{}, err
	}
	if connector == nil {
		return remoteauth.PairingExchangeResult{}, fmt.Errorf("direct pairing connector is required")
	}
	expectedKind := connector.RouteKind
	if expectedKind == "" {
		expectedKind = endpoint.RouteDirectWebRTCTCP
	}
	if request.Route().Kind != expectedKind {
		return remoteauth.PairingExchangeResult{}, fmt.Errorf("route %q kind %q does not match pairing connector kind %q", request.Route().ID, request.Route().Kind, expectedKind)
	}
	offer, err := remoteauth.ParsePairingClaimOfferForExchange(pairing.PairingClaimOffer)
	if err != nil {
		return remoteauth.PairingExchangeResult{}, err
	}
	digest := sha256.Sum256(offer.GetClaim())
	opened, err := openDirectPeer(ctx, request, directPeerOptions{
		Peers: connector.Peers, Signaling: connector.Signaling, Locators: connector.Locators, TransformAnswer: connector.TransformAnswer,
		Random: connector.Random, Now: connector.Now, Phase: connector.Phase,
		PairingDigest: digest[:], PairingPublicKey: pairing.Identity.PublicKey,
		PairingExpiresAt: time.Unix(0, offer.GetExpiresAtUnixNano()).UTC(),
	})
	if err != nil {
		return remoteauth.PairingExchangeResult{}, err
	}
	return (clientruntime.PairingService{Now: connector.Now}).Redeem(ctx, request, opened, pairing)
}

var _ clientruntime.PairingPeerSession = (*openedDirectPeer)(nil)
