package cloud

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/remoteauth"
)

// PairingConnector 通过 Cloud bootstrap ticket 建立 pairing-only DataChannel。
// claim 的验证和消费仍由公共 PairingService 与 owning daemon 完成。
type PairingConnector struct {
	Peers   PeerFactory
	Cloud   *cloudclient.Client
	Product cloudv1.ClientProduct
	Now     func() time.Time
	Phase   func(clientruntime.EndpointPhase)
}

func (connector *PairingConnector) Redeem(ctx context.Context, request clientruntime.AttemptRequest, pairing remoteauth.ClientPairingRequest) (remoteauth.PairingExchangeResult, error) {
	if connector == nil || connector.Peers == nil || connector.Cloud == nil || connector.Product == cloudv1.ClientProduct_CLIENT_PRODUCT_UNSPECIFIED {
		return remoteauth.PairingExchangeResult{}, errors.New("Cloud pairing connector dependencies are incomplete")
	}
	if err := clientruntime.ValidatePairingAttempt(request, pairing); err != nil {
		return remoteauth.PairingExchangeResult{}, err
	}
	if request.Route().Kind != endpoint.RouteManagedWebRTC {
		return remoteauth.PairingExchangeResult{}, fmt.Errorf("route %q is not managed WebRTC", request.Route().ID)
	}
	if len(pairing.PairingClaimOffer) == 0 {
		return remoteauth.PairingExchangeResult{}, errors.New("Cloud pairing requires a short-lived claim offer")
	}
	if connector.Phase != nil {
		connector.Phase(clientruntime.EndpointPhaseSignaling)
	}
	resolved, err := connector.Cloud.PairingRoute(pairing.PairingClaimOffer)
	if err != nil {
		return remoteauth.PairingExchangeResult{}, reportCloudFailure(request.Stamp().Generation, cloudFailurePairingRoute, cloudConnectionError(err))
	}
	opened, err := openResolvedCloudPeer(ctx, request, connector.Peers, connector.Cloud, resolved, pairing.Identity, pairing.Signer, connector.Product, connector.Phase)
	if err != nil {
		return remoteauth.PairingExchangeResult{}, reportCloudFailure(request.Stamp().Generation, cloudFailurePairingExchange, cloudConnectionError(err))
	}
	if connector.Phase != nil {
		connector.Phase(clientruntime.EndpointPhaseAuthorizing)
	}
	result, err := (clientruntime.PairingService{Now: connector.Now}).Redeem(ctx, request, opened, pairing)
	if err != nil {
		return remoteauth.PairingExchangeResult{}, reportCloudFailure(request.Stamp().Generation, cloudFailurePairingHandshake, err)
	}
	return result, nil
}
