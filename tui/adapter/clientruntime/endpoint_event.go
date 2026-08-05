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
		Message:              event.Message,
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
	case clientruntime.ErrorEntitlement, clientruntime.ErrorResourceExhausted,
		clientruntime.ErrorRelayNotInPlan, clientruntime.ErrorRelayQuotaExhausted, clientruntime.ErrorRelayConcurrencyExhausted,
		clientruntime.ErrorSubscriptionInactive, clientruntime.ErrorRelayRegionUnavailable:
		return state.EndpointErrorEntitlement
	default:
		return state.EndpointErrorUnknown
	}
}
