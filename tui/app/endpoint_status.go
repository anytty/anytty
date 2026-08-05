package app

import (
	"context"
	"strings"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

const endpointStatusWatchToken = CancelToken("endpoint.status.watch")

// EndpointWatchRequestMsg 请求启动 endpoint lifecycle 订阅。
// 订阅源来自 client runtime adapter；消息本身不改 reducer state，只建立主动侦测链路。
type EndpointWatchRequestMsg struct{}

func (EndpointWatchRequestMsg) isMsg() {}

func (EndpointWatchRequestMsg) SkipRender() bool {
	return true
}

// EndpointRuntimeStatusMsg 是 endpoint lifecycle 事件进入 reducer 的唯一入口。
// 它承载 endpoint-scoped 状态，不代表任何 terminal process 已退出或 history 已变化。
type EndpointRuntimeStatusMsg struct {
	Event port.EndpointRuntimeEvent
	Err   error
}

func (EndpointRuntimeStatusMsg) isMsg() {}

// NewEndpointStatusReducer 处理 client runtime adapter 主动发布的 transport/protocol 状态。
// reducer 只更新 endpoint/pane/live 投影；断线错误不得作为全局 toast 写入 ShellStore。
func NewEndpointStatusReducer(deps LiveDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case EndpointWatchRequestMsg:
			return reduceEndpointWatchRequest(root, deps)
		case EndpointRuntimeStatusMsg:
			return reduceEndpointRuntimeStatus(root, msg)
		default:
			return root, nil
		}
	}
}

func reduceEndpointWatchRequest(root state.Root, deps LiveDeps) (state.Root, []Effect) {
	source := deps.EndpointEvents
	if source == nil {
		if candidate, ok := deps.Terminal.(port.EndpointEventSource); ok {
			source = candidate
		}
	}
	if source == nil {
		return root, nil
	}
	return root, []Effect{StreamEffect{Token: endpointStatusWatchToken, Run: func(ctx context.Context, post func(Msg)) {
		events, err := source.WatchEndpointEvents(ctx)
		if err != nil {
			logEffectError(deps.Logger, "endpoint.status.watch", err)
			if isContextLifecycleError(err) {
				return
			}
			post(EndpointRuntimeStatusMsg{Err: err})
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				post(EndpointRuntimeStatusMsg{Event: event})
			}
		}
	}}}
}

func reduceEndpointRuntimeStatus(root state.Root, msg EndpointRuntimeStatusMsg) (state.Root, []Effect) {
	if msg.Err != nil {
		return root, nil
	}
	event := msg.Event
	event.EndpointID = state.NormalizeEndpointID(event.EndpointID)
	if event.EndpointID == "" {
		return root, nil
	}
	if current, ok := root.Endpoints.Endpoint(event.EndpointID); ok {
		if current.ConnectionGeneration > 0 && event.Generation == 0 {
			// Snapshot resolution 尚未分配 generation；已有 Ready session 时，这类尝试不能覆盖当前连接事实。
			return root, nil
		}
		if event.Generation > 0 && event.Generation < current.ConnectionGeneration {
			// SessionOwner generation 是 stale event 的唯一 fence；旧 Offline/Ready 不得污染新 winner。
			return root, nil
		}
	}
	message := endpointRuntimeEventMessage(event)
	errorKind := state.NormalizeEndpointErrorKind(event.ErrorKind)
	if errorKind == state.EndpointErrorUnknown && event.Err != nil {
		errorKind = state.ClassifyEndpointErrorText(event.Err.Error())
	}
	if errorKind == state.EndpointErrorUnknown && message != "" {
		errorKind = state.ClassifyEndpointErrorText(message)
	}
	status := event.Status
	if status == state.EndpointStatusUnknown {
		if message == "" {
			status = state.EndpointStatusConnected
		} else {
			status = state.EndpointStatusOffline
		}
	}
	count := endpointTerminalCount(root, event.EndpointID)
	root.Endpoints = root.Endpoints.MarkRuntimeStatus(event.EndpointID, status, errorKind, count, message)
	root.Endpoints = root.Endpoints.MarkRuntimeConnection(event.EndpointID, event.RouteID, event.Generation, status, event.ObservedPath, event.RouteSelectionReason)
	if event.Phase != "" {
		root.Endpoints = root.Endpoints.MarkConnectionPhase(event.EndpointID, event.Phase)
	} else if status == state.EndpointStatusConnected {
		root.Endpoints = root.Endpoints.MarkConnectionPhase(event.EndpointID, state.EndpointConnectionConnected)
	} else if status == state.EndpointStatusOffline {
		root.Endpoints = root.Endpoints.MarkConnectionPhase(event.EndpointID, state.EndpointConnectionFailed)
	}
	if status == state.EndpointStatusOffline {
		root = markEndpointOfflineInTerminalViews(root, event.EndpointID, endpointRuntimeDisplayMessage(errorKind, message))
	}
	return root.Advance(), nil
}

func endpointRuntimeEventMessage(event port.EndpointRuntimeEvent) string {
	message := strings.TrimSpace(event.Message)
	if message != "" {
		return message
	}
	if event.Err != nil {
		return strings.TrimSpace(event.Err.Error())
	}
	return ""
}

func endpointRuntimeDisplayMessage(kind state.EndpointErrorKind, message string) string {
	kind = state.NormalizeEndpointErrorKind(kind)
	message = strings.TrimSpace(message)
	if kind == state.EndpointErrorUnknown {
		if message == "" {
			return "endpoint offline"
		}
		return message
	}
	if message == "" {
		return string(kind)
	}
	return string(kind) + ": " + message
}

func markEndpointOfflineInTerminalViews(root state.Root, endpointID state.EndpointID, message string) state.Root {
	refs := terminalRefsForEndpointBindings(root, endpointID)
	root.TerminalViews = root.TerminalViews.MarkEndpointRuntimeError(endpointID, message)
	for _, ref := range refs {
		active := liveErrorRefOwnsActiveProjection(root, ref)
		root.Session = root.Session.ClearInputChannelRef(ref)
		root = applyLiveErrorRefWithActive(root, ref, message, active)
	}
	return root
}

func terminalRefsForEndpointBindings(root state.Root, endpointID state.EndpointID) []state.TerminalRef {
	endpointID = state.NormalizeEndpointID(endpointID)
	seen := map[string]struct{}{}
	refs := []state.TerminalRef{}
	for _, binding := range root.TerminalViews.Bindings() {
		ref := binding.TerminalRef()
		if ref.EndpointID != endpointID || ref.TerminalID == "" {
			continue
		}
		key := ref.Key()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		refs = append(refs, ref)
	}
	return refs
}

func endpointTerminalCount(root state.Root, endpointID state.EndpointID) int {
	endpointID = state.NormalizeEndpointID(endpointID)
	count := 0
	for _, item := range root.TerminalPool.Items {
		if item.TerminalRef().EndpointID == endpointID && item.TerminalID != "" {
			count++
		}
	}
	return count
}
