package runtime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
)

// PairingPeerSession 是 Route connector 完成 signaling、ICE、DTLS 和 DataChannel 后交给通用 pairing service 的最小边界。
// 它只暴露当前 peer 的 transport 与实际远端证书 fingerprint；Endpoint、Route、generation 和 ticket 仍由 Go runtime 持有。
type PairingPeerSession interface {
	// Transport 返回本次 peer 的可靠有序 DataChannel transport。
	Transport() transport.Transport
	// RemoteCertificateFingerprint 返回当前实际 DTLS peer certificate fingerprint，不能使用 signaling 提供的未验证字符串替代。
	RemoteCertificateFingerprint() (string, error)
	// Close 幂等释放本次 pairing peer、DataChannel 和 Route-specific 资源。
	Close() error
}

// PairingService 是 Direct、SSH 和 Cloud 共用的一次性 PairingTicket 兑换服务。
// service 不建立网络、不选择 Route、不持久化 grant；成功或失败都精确调用一次 peer Close，调用方不得继续复用 pairing DataChannel。
type PairingService struct {
	Now func() time.Time
}

// Redeem 校验 attempt generation、Endpoint pin 和实际 DTLS binding 后执行公共 remote-auth pairing handshake。
// 返回 grant 后 peer 已关闭；持久化 ClientAccessIdentity/grant 的原子事务仍由调用方 credential owner 完成。
func (service PairingService) Redeem(ctx context.Context, request AttemptRequest, peer PairingPeerSession, pairing remoteauth.ClientPairingRequest) (result remoteauth.PairingExchangeResult, err error) {
	if peer == nil || (reflect.ValueOf(peer).Kind() == reflect.Pointer && reflect.ValueOf(peer).IsNil()) {
		return remoteauth.PairingExchangeResult{}, runtimeError(ErrorUnavailable, "pairing peer session is required", nil)
	}
	defer func() {
		if closeErr := peer.Close(); closeErr != nil {
			if err == nil {
				result = remoteauth.PairingExchangeResult{}
				err = runtimeError(ErrorUnavailable, "close pairing peer session", closeErr)
			} else {
				err = errors.Join(err, closeErr)
			}
		}
	}()
	if err := ValidatePairingAttempt(request, pairing); err != nil {
		return remoteauth.PairingExchangeResult{}, err
	}
	connection := peer.Transport()
	if connection == nil {
		return remoteauth.PairingExchangeResult{}, runtimeError(ErrorUnavailable, "pairing peer has no DataChannel transport", nil)
	}
	fingerprint, fingerprintErr := peer.RemoteCertificateFingerprint()
	if fingerprintErr != nil {
		return remoteauth.PairingExchangeResult{}, runtimeError(ErrorIdentity, "read pairing peer DTLS certificate", fingerprintErr)
	}
	binding, bindingErr := remoteauth.DTLSChannelBinding(strings.TrimSpace(fingerprint))
	if bindingErr != nil {
		return remoteauth.PairingExchangeResult{}, runtimeError(ErrorIdentity, "build pairing peer DTLS channel binding", bindingErr)
	}
	pairing.ChannelBinding = binding
	result, err = (remoteauth.ClientPairingHandshake{Now: service.Now}).Redeem(ctx, connection, pairing)
	if err != nil {
		return remoteauth.PairingExchangeResult{}, fmt.Errorf("redeem pairing ticket over peer session: %w", err)
	}
	return result, nil
}

// ValidatePairingAttempt 在任何 signaling 或网络副作用前校验 claim 兑换所需的 attempt、signer 和 Endpoint pin。
// Direct、SSH 和 Cloud connector 必须调用本函数，不能复制或放宽 pairing admission 规则。
func ValidatePairingAttempt(request AttemptRequest, pairing remoteauth.ClientPairingRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	if pairing.Signer == nil || len(pairing.PairingClaimOffer) == 0 {
		return runtimeError(ErrorInvalidRequest, "pairing identity transaction is incomplete", nil)
	}
	identity := request.DaemonIdentity()
	if pairing.ExpectedDeviceID != identity.DeviceID || pairing.ExpectedDeviceFingerprint != identity.DeviceFingerprint {
		return runtimeError(ErrorIdentity, "pairing endpoint pin does not match route attempt", nil)
	}
	return nil
}
