package app

import (
	"context"
	"fmt"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

const endpointDefaultsTokenPrefix = "endpoint.defaults:"

// EndpointDefaultsRequestMsg 请求从 owning daemon 拉取创建终端默认值。
// 默认 command/cwd 的 truth 在 endpoint daemon，TUI reducer 只保存结果投影。
type EndpointDefaultsRequestMsg struct {
	EndpointID state.EndpointID
	Force      bool
}

func (EndpointDefaultsRequestMsg) isMsg() {}

// SkipRender 表示请求消息本身不改变可见 state；结果回投后再刷新 prompt。
func (EndpointDefaultsRequestMsg) SkipRender() bool {
	return true
}

// EndpointDefaultsResultMsg 是 path.defaults 查询回投。
// Request.EndpointID 是 stale guard；Result.EndpointID 由 client runtime adapter 回填。
type EndpointDefaultsResultMsg struct {
	Request EndpointDefaultsRequestMsg
	Result  port.PathDefaultsResult
	Err     error
}

func (EndpointDefaultsResultMsg) isMsg() {}

// NewEndpointDefaultsReducer 维护 endpoint create defaults 投影。
// 它不创建 terminal，也不读本地环境，只把 PathService 返回值写入 EndpointStore。
func NewEndpointDefaultsReducer(deps LiveDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case EndpointDefaultsRequestMsg:
			return reduceEndpointDefaultsRequest(root, msg, deps)
		case EndpointDefaultsResultMsg:
			return reduceEndpointDefaultsResult(root, msg)
		default:
			return root, nil
		}
	}
}

func reduceEndpointDefaultsRequest(root state.Root, msg EndpointDefaultsRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	endpointID := state.NormalizeEndpointID(msg.EndpointID)
	if !msg.Force {
		if endpoint, ok := root.Endpoints.DisplayEndpoint(endpointID); ok && endpoint.DefaultsLoaded && endpoint.DefaultsError == "" {
			return root, nil
		}
	}
	request := EndpointDefaultsRequestMsg{EndpointID: endpointID, Force: msg.Force}
	return root, []Effect{FuncEffect{
		Token:            CancelToken(endpointDefaultsTokenPrefix + string(endpointID)),
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			if deps.Path == nil {
				return EndpointDefaultsResultMsg{Request: request, Err: fmt.Errorf("path service missing")}
			}
			result, err := deps.Path.Defaults(ctx, port.PathDefaultsRequest{EndpointID: endpointID})
			return EndpointDefaultsResultMsg{Request: request, Result: result, Err: err}
		},
	}}
}

func reduceEndpointDefaultsResult(root state.Root, msg EndpointDefaultsResultMsg) (state.Root, []Effect) {
	endpointID := state.NormalizeEndpointID(msg.Request.EndpointID)
	if msg.Result.EndpointID != "" {
		endpointID = state.NormalizeEndpointID(msg.Result.EndpointID)
	}
	root.Endpoints = root.Endpoints.ApplyDefaults(endpointID, msg.Result.DefaultCommand, msg.Result.DefaultCWD, errorString(msg.Err))
	if currentCreatePromptEndpoint(root) == endpointID {
		root.Shell = syncCreatePromptDefaultsForEndpoint(root, root.Shell, endpointID)
	}
	return root.Advance(), nil
}

func endpointDefaultsRequestEffect(endpointID state.EndpointID, force bool) []Effect {
	endpointID = state.NormalizeEndpointID(endpointID)
	return []Effect{FuncEffect{Run: func(context.Context) Msg {
		return EndpointDefaultsRequestMsg{EndpointID: endpointID, Force: force}
	}}}
}

func currentCreatePromptEndpoint(root state.Root) state.EndpointID {
	shell := root.Shell.EnsureDefaults()
	if shell.Overlay.Kind != state.OverlayPrompt || !shell.Overlay.Open || shell.Overlay.Prompt.Purpose != "terminal.create" {
		return ""
	}
	endpointID, err := terminalCreateEndpointIDFromPrompt(root, shell.Overlay.Prompt)
	if err != nil {
		return ""
	}
	return state.NormalizeEndpointID(endpointID)
}
