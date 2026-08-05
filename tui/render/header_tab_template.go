package render

import (
	"bytes"
	"strings"
	texttemplate "text/template"
)

const (
	headerTabTemplateEscapedLeftBracket  = "\x00anytty-header-tab-left-bracket\x00"
	headerTabTemplateEscapedRightBracket = "\x00anytty-header-tab-right-bracket\x00"
)

type headerTabTemplateContext struct {
	Index        int
	Title        string
	TabID        string
	Workspace    string
	WorkspaceID  string
	Active       bool
	SwitchAction string
	CloseAction  string
	CloseTarget  string
	CloseIcon    string
	CreateIcon   string
	DefaultStyle StyleToken
}

type headerTabTemplateData struct {
	Index       int
	TabID       string
	ID          string
	Title       string
	Workspace   string
	WorkspaceID string
	Active      bool
	NotActive   bool
	Marker      string
	CloseIcon   string
	CreateIcon  string
}

type headerTabTemplateState struct {
	style         StyleToken
	ansi          ANSICellStyle
	actionID      string
	targetID      string
	defaultStyle  StyleToken
	defaultID     string
	defaultTarget string
}

type headerTabANSIUpdate struct {
	fg               string
	bg               string
	setFG            bool
	setBG            bool
	bold             bool
	italic           bool
	underline        bool
	blink            bool
	reverse          bool
	strikethrough    bool
	setBold          bool
	setItalic        bool
	setUnderline     bool
	setBlink         bool
	setReverse       bool
	setStrikethrough bool
}

func headerTabTemplateSegments(format string, ctx headerTabTemplateContext) []barSegment {
	if strings.TrimSpace(format) == "" {
		return nil
	}
	rendered, ok := executeHeaderTabTemplate(format, ctx)
	if !ok || rendered == "" {
		return nil
	}
	// tab format 属于 render 投影配置：这里只把模板编译成带样式和 ActionID 的 segment，
	// 条件/函数交给标准库 text/template，点击后的状态变化仍必须回到 reducer 处理。
	defaultStyle := StyleHeaderInactiveTitle
	if ctx.Active {
		defaultStyle = StyleHeaderActiveTitle
	}
	if ctx.DefaultStyle != "" {
		defaultStyle = ctx.DefaultStyle
	}
	state := headerTabTemplateState{
		style:         defaultStyle,
		actionID:      ctx.SwitchAction,
		targetID:      ctx.TabID,
		defaultStyle:  defaultStyle,
		defaultID:     ctx.SwitchAction,
		defaultTarget: ctx.TabID,
	}
	segments := make([]barSegment, 0, 8)
	for len(rendered) > 0 {
		next := strings.Index(rendered, "[")
		if next < 0 {
			segments = appendHeaderTemplateText(segments, rendered, state)
			break
		}
		if next > 0 {
			segments = appendHeaderTemplateText(segments, rendered[:next], state)
			rendered = rendered[next:]
		}
		end := strings.Index(rendered, "]")
		if end < 0 {
			segments = appendHeaderTemplateText(segments, rendered[:1], state)
			rendered = rendered[1:]
			continue
		}
		tag := rendered[1:end]
		if nextState, ok := applyHeaderTabTemplateTag(tag, state, ctx); ok {
			state = nextState
			rendered = rendered[end+1:]
			continue
		}
		segments = appendHeaderTemplateText(segments, rendered[:end+1], state)
		rendered = rendered[end+1:]
	}
	if len(segments) == 0 {
		return nil
	}
	return segments
}

func headerWorkspaceTemplateSegments(format string, workspace string) []barSegment {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		workspace = "anytty"
	}
	return headerTabTemplateSegments(format, headerTabTemplateContext{
		Title:        workspace,
		TabID:        workspace,
		Workspace:    workspace,
		WorkspaceID:  workspace,
		Active:       true,
		SwitchAction: "menu.workbench_tree",
	})
}

func headerCreateTemplateSegments(format string, icon string) []barSegment {
	return headerTabTemplateSegments(format, headerTabTemplateContext{
		Title:        "create",
		Active:       true,
		SwitchAction: ActionTabCreate.String(),
		CreateIcon:   strings.TrimSpace(icon),
		DefaultStyle: StyleHeaderCreate,
	})
}

func executeHeaderTabTemplate(format string, ctx headerTabTemplateContext) (string, bool) {
	tmpl, err := texttemplate.New("header_tab").Option("missingkey=zero").Funcs(headerTabTemplateFuncs(ctx)).Parse(format)
	if err != nil {
		return "", false
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, headerTabTemplateData{
		Index:       ctx.Index,
		TabID:       headerTabTemplateEscapeText(ctx.TabID),
		ID:          headerTabTemplateEscapeText(ctx.TabID),
		Title:       headerTabTemplateEscapeText(ctx.Title),
		Workspace:   headerTabTemplateEscapeText(ctx.Workspace),
		WorkspaceID: headerTabTemplateEscapeText(ctx.WorkspaceID),
		Active:      ctx.Active,
		NotActive:   !ctx.Active,
		Marker:      headerTabTemplateEscapeText(headerTabMarker(ctx.Active)),
		CloseIcon:   headerTabTemplateEscapeText(ctx.CloseIcon),
		CreateIcon:  headerTabTemplateEscapeText(ctx.CreateIcon),
	}); err != nil {
		return "", false
	}
	return out.String(), true
}

func headerTabTemplateFuncs(ctx headerTabTemplateContext) texttemplate.FuncMap {
	return texttemplate.FuncMap{
		"tab_id": func() string { return headerTabTemplateEscapeText(ctx.TabID) },
		"id": func() string {
			return headerTabTemplateEscapeText(ctx.TabID)
		},
		"index": func() int { return ctx.Index },
		"tab_index": func() int {
			return ctx.Index
		},
		"title": func() string {
			return headerTabTemplateEscapeText(ctx.Title)
		},
		"workspace": func() string {
			return headerTabTemplateEscapeText(ctx.Workspace)
		},
		"workspace_id": func() string {
			return headerTabTemplateEscapeText(ctx.WorkspaceID)
		},
		"active": func() bool { return ctx.Active },
		"not_active": func() bool {
			return !ctx.Active
		},
		"marker": func() string {
			return headerTabTemplateEscapeText(headerTabMarker(ctx.Active))
		},
		"close_icon": func() string {
			return headerTabTemplateEscapeText(ctx.CloseIcon)
		},
		"create_icon": func() string {
			return headerTabTemplateEscapeText(ctx.CreateIcon)
		},
		"truncate": func(width int, value string) string {
			if width <= 0 {
				return ""
			}
			return headerTabTemplateEscapeText(TruncateCells(headerTabTemplateUnescapeText(value), width))
		},
		"fg": func(value string) string {
			return headerTabTemplateInlineTag("fg", value)
		},
		"bg": func(value string) string {
			return headerTabTemplateInlineTag("bg", value)
		},
		"style": func(value string) string {
			return headerTabTemplateInlineTag("style", value)
		},
		"action": func(value string) string {
			return headerTabTemplateInlineTag("action", value)
		},
		"end_style": func() string { return "[/style]" },
		"end_action": func() string {
			return "[/action]"
		},
	}
}

func headerTabMarker(active bool) string {
	if active {
		return "▎"
	}
	return " "
}

func headerTabTemplateInlineTag(key string, value string) string {
	key = headerTabTemplateTagValue(key)
	value = headerTabTemplateTagValue(value)
	if key == "" || value == "" {
		return ""
	}
	return "[" + key + ":" + value + "]"
}

func headerTabTemplateTagValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "[", "")
	value = strings.ReplaceAll(value, "]", "")
	value = strings.ReplaceAll(value, ";", "")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func headerTabTemplateEscapeText(value string) string {
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "[", headerTabTemplateEscapedLeftBracket)
	value = strings.ReplaceAll(value, "]", headerTabTemplateEscapedRightBracket)
	return value
}

func headerTabTemplateUnescapeText(value string) string {
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, headerTabTemplateEscapedLeftBracket, "[")
	value = strings.ReplaceAll(value, headerTabTemplateEscapedRightBracket, "]")
	return value
}

func appendHeaderTemplateText(segments []barSegment, text string, state headerTabTemplateState) []barSegment {
	text = headerTabTemplateUnescapeText(text)
	if text == "" {
		return segments
	}
	segment := barText(text, state.style, 1).withAction(state.actionID).withTarget(state.targetID)
	segment.ansi = state.ansi
	return append(segments, segment)
}

func applyHeaderTabTemplateTag(tag string, state headerTabTemplateState, ctx headerTabTemplateContext) (headerTabTemplateState, bool) {
	tag = strings.TrimSpace(tag)
	switch {
	case tag == "/" || tag == "/style" || tag == "reset":
		state.style = state.defaultStyle
		state.ansi = ANSICellStyle{}
		return state, true
	case tag == "/action":
		state.actionID = state.defaultID
		state.targetID = state.defaultTarget
		return state, true
	case strings.HasPrefix(tag, "style:") || strings.HasPrefix(tag, "style="):
		if style, ok := headerTabTemplateStyleToken(headerTabTemplateTagArgument(tag)); ok {
			state.style = style
			state.ansi = ANSICellStyle{}
			return state, true
		}
		return state, false
	case strings.HasPrefix(tag, "action:") || strings.HasPrefix(tag, "action="):
		// action tag 只声明现有消息链路的 ActionID/target；未知 action 默认指向当前 tab，
		// 方便后续可编程按钮扩展，但不在模板编译阶段直接修改 reducer-owned state。
		action := headerTabTemplateTagArgument(tag)
		switch action {
		case "", "none":
			state.actionID = ""
			state.targetID = ""
		case ActionTabClose.String(), ctx.CloseAction:
			state.actionID = ctx.CloseAction
			if state.actionID == "" {
				state.actionID = ActionTabClose.String()
			}
			state.targetID = ctx.CloseTarget
		case ActionTabSwitch.String(), ctx.SwitchAction:
			state.actionID = ctx.SwitchAction
			if state.actionID == "" {
				state.actionID = ActionTabSwitch.String()
			}
			state.targetID = ctx.TabID
		default:
			state.actionID = action
			state.targetID = ctx.TabID
		}
		return state, true
	default:
		if update, ok := headerTabTemplateANSIUpdate(tag); ok {
			state.style = ""
			state.ansi = update.apply(state.ansi)
			return state, true
		}
		return state, false
	}
}

func headerTabTemplateTagArgument(tag string) string {
	if _, value, ok := strings.Cut(tag, ":"); ok {
		return strings.TrimSpace(value)
	}
	if _, value, ok := strings.Cut(tag, "="); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func headerTabTemplateStyleToken(name string) (StyleToken, bool) {
	switch StyleToken(name) {
	case StyleAccent, StyleForeground, StyleStrongForeground, StyleMuted,
		StyleStatus, StyleStatusAccent, StyleStatusMuted, StyleStatusWarning,
		StyleHeaderWorkspace, StyleHeaderWorkspaceEdge, StyleHeaderSpacer,
		StyleHeaderInactiveIndex, StyleHeaderInactiveTitle, StyleHeaderInactiveClose,
		StyleHeaderActiveEdge, StyleHeaderActiveMarker, StyleHeaderActiveIndex,
		StyleHeaderActiveTitle, StyleHeaderActiveClose, StyleHeaderCreate:
		return StyleToken(name), true
	default:
		return "", false
	}
}

func headerTabTemplateANSIUpdate(tag string) (headerTabANSIUpdate, bool) {
	update := headerTabANSIUpdate{}
	for _, part := range strings.Split(tag, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			key, value, ok = strings.Cut(part, "=")
		}
		if !ok {
			key = part
			value = "true"
		}
		key = headerTabTemplateNormalizeStyleKey(key)
		value = strings.TrimSpace(value)
		switch key {
		case "fg", "foreground", "color":
			update.fg = value
			update.setFG = true
		case "bg", "background":
			update.bg = value
			update.setBG = true
		case "bold":
			update.bold = headerTabTemplateTruthy(value)
			update.setBold = true
		case "italic":
			update.italic = headerTabTemplateTruthy(value)
			update.setItalic = true
		case "underline":
			update.underline = headerTabTemplateTruthy(value)
			update.setUnderline = true
		case "blink":
			update.blink = headerTabTemplateTruthy(value)
			update.setBlink = true
		case "reverse", "inverse":
			update.reverse = headerTabTemplateTruthy(value)
			update.setReverse = true
		case "strikethrough", "strike":
			update.strikethrough = headerTabTemplateTruthy(value)
			update.setStrikethrough = true
		case "font", "font-style", "font-styles":
			applyHeaderTabFontStyle(&update, value)
		default:
			continue
		}
	}
	return update, update.any()
}

func headerTabTemplateNormalizeStyleKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "_", "-")
	key = strings.ReplaceAll(key, " ", "-")
	return key
}

func applyHeaderTabFontStyle(update *headerTabANSIUpdate, value string) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "normal" || value == "regular" || value == "none" {
		update.bold = false
		update.italic = false
		update.underline = false
		update.setBold = true
		update.setItalic = true
		update.setUnderline = true
		return
	}
	for _, field := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '+'
	}) {
		switch field {
		case "bold":
			update.bold = true
			update.setBold = true
		case "italic":
			update.italic = true
			update.setItalic = true
		case "underline":
			update.underline = true
			update.setUnderline = true
		case "strike", "strikethrough":
			update.strikethrough = true
			update.setStrikethrough = true
		}
	}
}

func headerTabTemplateTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "1", "t", "true", "yes", "y", "on", "enabled":
		return true
	default:
		return false
	}
}

func (update headerTabANSIUpdate) any() bool {
	return update.setFG || update.setBG || update.setBold || update.setItalic ||
		update.setUnderline || update.setBlink || update.setReverse || update.setStrikethrough
}

func (update headerTabANSIUpdate) apply(style ANSICellStyle) ANSICellStyle {
	if update.setFG {
		style.FG = update.fg
	}
	if update.setBG {
		style.BG = update.bg
	}
	if update.setBold {
		style.Bold = update.bold
	}
	if update.setItalic {
		style.Italic = update.italic
	}
	if update.setUnderline {
		style.Underline = update.underline
	}
	if update.setBlink {
		style.Blink = update.blink
	}
	if update.setReverse {
		style.Reverse = update.reverse
	}
	if update.setStrikethrough {
		style.Strikethrough = update.strikethrough
	}
	return style
}
