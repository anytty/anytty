package clientruntimeadapter

import (
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

// ProjectEndpointEvent 把共享 runtime lifecycle event 映射为 TUI-owned endpoint 投影。
// 映射只转换稳定枚举和展示字段，不读取 registry、不创建连接；stamp 原样投影用于 reducer generation fence。
func ProjectEndpointEvent(event clientruntime.EndpointEvent) port.EndpointRuntimeEvent {
	phase := projectConnectionPhase(event.Phase)
	status := state.EndpointStatusConnecting
	switch event.Phase {
	case clientruntime.EndpointPhaseReady:
		status = state.EndpointStatusConnected
	case clientruntime.EndpointPhaseOffline:
		status = state.EndpointStatusOffline
	}
	return port.EndpointRuntimeEvent{
		EndpointID:           state.EndpointID(event.EndpointID),
		RouteID:              string(event.Stamp.RouteID),
		Generation:           uint64(event.Stamp.Generation),
		Status:               status,
		ErrorKind:            projectErrorKind(event.ErrorCode),
		Phase:                phase,
		ObservedPath:         event.ObservedPath,
		RouteSelectionReason: event.RouteSelectionReason,
		Message:              projectErrorMessage(event.ErrorCode, event.Message),
	}
}

func projectConnectionPhase(phase clientruntime.EndpointPhase) state.EndpointConnectionPhase {
	switch phase {
	case clientruntime.EndpointPhaseIdle:
		return state.EndpointConnectionIdle
	case clientruntime.EndpointPhasePlanning, clientruntime.EndpointPhaseResolving:
		return state.EndpointConnectionResolving
	case clientruntime.EndpointPhaseSignaling:
		return state.EndpointConnectionSignaling
	case clientruntime.EndpointPhaseConnecting:
		return state.EndpointConnectionConnecting
	case clientruntime.EndpointPhaseAuthorizing:
		return state.EndpointConnectionAuthorizing
	case clientruntime.EndpointPhaseReady:
		return state.EndpointConnectionConnected
	case clientruntime.EndpointPhaseOffline:
		return state.EndpointConnectionFailed
	default:
		return ""
	}
}

func projectErrorKind(code clientruntime.ErrorCode) state.EndpointErrorKind {
	switch code {
	case "":
		return state.EndpointErrorUnknown
	case clientruntime.ErrorInvalidRequest, clientruntime.ErrorUnsupportedRoute:
		return state.EndpointErrorConfig
	case clientruntime.ErrorIdentity, clientruntime.ErrorAuthorization:
		return state.EndpointErrorAuth
	case clientruntime.ErrorStaleSession:
		return state.EndpointErrorProtocol
	case clientruntime.ErrorCanceled, clientruntime.ErrorUnavailable:
		return state.EndpointErrorUnavailable
	case clientruntime.ErrorRelayNotInPlan:
		return state.EndpointErrorRelayPlan
	case clientruntime.ErrorRelayQuotaExhausted:
		return state.EndpointErrorRelayQuota
	case clientruntime.ErrorRelayConcurrencyExhausted:
		return state.EndpointErrorRelayConcurrency
	case clientruntime.ErrorSubscriptionInactive:
		return state.EndpointErrorSubscription
	case clientruntime.ErrorRelayRegionUnavailable:
		return state.EndpointErrorRelayRegion
	case clientruntime.ErrorEntitlement, clientruntime.ErrorResourceExhausted:
		return state.EndpointErrorEntitlement
	default:
		return state.EndpointErrorUnknown
	}
}

func projectErrorMessage(code clientruntime.ErrorCode, fallback string) string {
	switch code {
	case clientruntime.ErrorRelayNotInPlan:
		return "P2P could not be established and this device's Cloud plan does not include Relay. Use Direct or SSH, or ask the device owner to change the plan."
	case clientruntime.ErrorRelayQuotaExhausted:
		return "This period's Relay traffic is used up. P2P, Direct, and SSH remain available; check the Cloud console or wait for the next period."
	case clientruntime.ErrorRelayConcurrencyExhausted:
		return "The Relay connection limit is reached. Close another Relay connection, or use P2P, Direct, or SSH."
	case clientruntime.ErrorSubscriptionInactive:
		return "This device's AnyTTY Cloud subscription has not started, is suspended, or has expired. Ask the device owner to manage it in the Cloud console; Direct and SSH remain available."
	case clientruntime.ErrorRelayRegionUnavailable:
		return "Relay is unavailable in the current region. Select another region, or use P2P, Direct, or SSH."
	default:
		return fallback
	}
}
