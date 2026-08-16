package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

const (
	promptPathCompletionToken        CancelToken = "prompt.path.completion"
	defaultPromptPathCompletionLimit int         = 100
)

// PromptPathCompletionRequestMsg 请求对当前 prompt workdir 字段做 endpoint-scoped 目录补全。
// 它只携带触发时的 endpoint、字段和值快照；真正的文件系统 truth 必须由 PathService
// 通过 owning daemon 查询，不能在 reducer 内读取远端路径。
type PromptPathCompletionRequestMsg struct {
	EndpointID  state.EndpointID
	ActiveField int
	Value       string
	Cursor      int
	Prefix      string
	Limit       int
	Focus       bool
}

func (PromptPathCompletionRequestMsg) isMsg() {}

// SkipRender 避免请求消息本身产生空 frame；结果消息回来后才更新 prompt 候选。
func (PromptPathCompletionRequestMsg) SkipRender() bool {
	return true
}

// PromptPathCompletionResultMsg 是 endpoint daemon 目录候选回到 TUI reducer 的结果。
// Request 是 stale guard 的真值快照，Result.EndpointID 由 client runtime adapter 回填。
type PromptPathCompletionResultMsg struct {
	Request PromptPathCompletionRequestMsg
	Result  port.PathListDirectoriesResult
	Err     error
}

func (PromptPathCompletionResultMsg) isMsg() {}

// NewPromptPathCompletionReducer 处理远端 workdir prompt 补全的异步 service path。
// UI input reducer 只发请求消息；该 reducer 通过 PathService 查询 endpoint daemon，
// 再把候选写回 reducer-owned prompt state。
func NewPromptPathCompletionReducer(deps LiveDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case PromptPathCompletionRequestMsg:
			return reducePromptPathCompletionRequest(root, msg, deps)
		case PromptPathCompletionResultMsg:
			return reducePromptPathCompletionResult(root, msg)
		default:
			return root, nil
		}
	}
}

func reducePromptPathCompletionRequest(root state.Root, msg PromptPathCompletionRequestMsg, deps LiveDeps) (state.Root, []Effect) {
	if !promptPathCompletionMatches(root, msg) {
		return root, nil
	}
	if msg.Limit <= 0 {
		msg.Limit = defaultPromptPathCompletionLimit
	}
	return root, []Effect{FuncEffect{
		Token:            promptPathCompletionToken,
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			if deps.Path == nil {
				return PromptPathCompletionResultMsg{Request: msg, Err: fmt.Errorf("path service missing")}
			}
			result, err := deps.Path.ListDirectories(ctx, port.PathListDirectoriesRequest{
				EndpointID: msg.EndpointID,
				Prefix:     msg.Prefix,
				Limit:      msg.Limit,
			})
			return PromptPathCompletionResultMsg{Request: msg, Result: result, Err: err}
		},
	}}
}

func reducePromptPathCompletionResult(root state.Root, msg PromptPathCompletionResultMsg) (state.Root, []Effect) {
	if !promptPathCompletionMatches(root, msg.Request) {
		return root, nil
	}
	title, items, empty := promptPathCompletionSuggestions(msg.Result, msg.Err)
	root.Shell = root.Shell.SetActivePromptSuggestions(title, items, empty)
	if msg.Request.Focus && len(items) > 0 {
		root.Shell = root.Shell.SetPromptSuggestionFocused(true)
	}
	return root.Advance(), nil
}

func promptPathCompletionTriggerEffect(root state.Root, focus bool) []Effect {
	request, ok := promptPathCompletionRequestFromRoot(root, focus)
	if !ok {
		return nil
	}
	return []Effect{FuncEffect{Run: func(context.Context) Msg { return request }}}
}

func promptCompletionHandledEffects(root state.Root, focus bool) []Effect {
	effects := []Effect{handledEffect{}}
	effects = append(effects, promptEndpointDefaultsTriggerEffect(root)...)
	return append(effects, promptPathCompletionTriggerEffect(root, focus)...)
}

func promptPathCompletionRequestFromRoot(root state.Root, focus bool) (PromptPathCompletionRequestMsg, bool) {
	shell := root.Shell.EnsureDefaults()
	if shell.Overlay.Kind != state.OverlayPrompt || !shell.Overlay.Open {
		return PromptPathCompletionRequestMsg{}, false
	}
	prompt := shell.Overlay.Prompt
	field := prompt.ActivePromptField()
	if field == nil || strings.TrimSpace(field.Key) != "workdir" {
		return PromptPathCompletionRequestMsg{}, false
	}
	endpointID, err := terminalCreateEndpointIDFromPrompt(root, prompt)
	if err != nil {
		return PromptPathCompletionRequestMsg{}, false
	}
	prefix := promptPathCompletionPrefix(field.Value, field.Cursor)
	if strings.TrimSpace(prefix) == "" {
		return PromptPathCompletionRequestMsg{}, false
	}
	return PromptPathCompletionRequestMsg{
		EndpointID:  state.NormalizeEndpointID(endpointID),
		ActiveField: prompt.ActiveField,
		Value:       field.Value,
		Cursor:      field.Cursor,
		Prefix:      prefix,
		Limit:       defaultPromptPathCompletionLimit,
		Focus:       focus,
	}, true
}

func promptEndpointDefaultsTriggerEffect(root state.Root) []Effect {
	endpointID := currentCreatePromptEndpoint(root)
	if endpointID == "" {
		return nil
	}
	if endpoint, ok := root.Endpoints.DisplayEndpoint(endpointID); ok && endpoint.DefaultsLoaded && endpoint.DefaultsError == "" {
		return nil
	}
	return endpointDefaultsRequestEffect(endpointID, false)
}

func promptPathCompletionMatches(root state.Root, request PromptPathCompletionRequestMsg) bool {
	shell := root.Shell.EnsureDefaults()
	if shell.Overlay.Kind != state.OverlayPrompt || !shell.Overlay.Open {
		return false
	}
	prompt := shell.Overlay.Prompt
	if prompt.ActiveField != request.ActiveField {
		return false
	}
	field := prompt.ActivePromptField()
	if field == nil || strings.TrimSpace(field.Key) != "workdir" {
		return false
	}
	endpointID, err := terminalCreateEndpointIDFromPrompt(root, prompt)
	if err != nil || state.NormalizeEndpointID(endpointID) != state.NormalizeEndpointID(request.EndpointID) {
		return false
	}
	return field.Value == request.Value &&
		field.Cursor == request.Cursor &&
		promptPathCompletionPrefix(field.Value, field.Cursor) == request.Prefix
}

func promptPathCompletionPrefix(value string, cursor int) string {
	runes := []rune(value)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	return string(runes[:cursor])
}

func promptPathCompletionSuggestions(result port.PathListDirectoriesResult, err error) (string, []string, string) {
	title := "path"
	if result.BasePath != "" {
		title = "path: " + result.BasePath
	}
	if err != nil {
		return title, nil, "(" + errorString(err) + ")"
	}
	if result.Missing {
		return title, nil, "(path not found)"
	}
	items := make([]string, 0, len(result.Entries))
	for _, entry := range result.Entries {
		if entry.Path != "" {
			items = append(items, entry.Path)
		}
	}
	if len(items) == 0 {
		return title, nil, "(no matching directories)"
	}
	return title, items, ""
}
