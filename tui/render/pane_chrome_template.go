package render

import (
	"bytes"
	"strings"
	texttemplate "text/template"
)

type paneChromeTemplateContext struct {
	Glyph    string
	Text     string
	ActionID string
	Label    string
	Left     string
	Right    string
	Index    int
	Count    int
	First    bool
	Last     bool
	Active   bool
	ZoomMode bool
}

type paneChromeTemplateData struct {
	Glyph     string
	Text      string
	ActionID  string
	Action    string
	Label     string
	Left      string
	Right     string
	Index     int
	Count     int
	First     bool
	Last      bool
	Active    bool
	NotActive bool
	// IsZoomMode 只表达 pane chrome 当前是否来自 zoom 投影；
	// 它让模板做条件展示，不能绕过 ActionID 的 reducer 命令链路。
	IsZoomMode  bool
	NotZoomMode bool
}

type paneChromeTemplateState struct {
	style        StyleToken
	ansi         ANSICellStyle
	defaultStyle StyleToken
}

type paneChromeRenderedText struct {
	Text     string
	Segments []barSegment
}

func paneChromeExecuteTemplateString(format string, ctx paneChromeTemplateContext) string {
	rendered, ok := executePaneChromeTemplate(format, ctx)
	if !ok {
		return format
	}
	return rendered
}

func executePaneChromeTemplate(format string, ctx paneChromeTemplateContext) (string, bool) {
	tmpl, err := texttemplate.New("pane_chrome").Option("missingkey=zero").Funcs(paneChromeTemplateFuncs(ctx)).Parse(format)
	if err != nil {
		return "", false
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, paneChromeTemplateData{
		Glyph:       headerTabTemplateEscapeText(ctx.Glyph),
		Text:        headerTabTemplateEscapeText(ctx.Text),
		ActionID:    headerTabTemplateEscapeText(ctx.ActionID),
		Action:      headerTabTemplateEscapeText(ctx.ActionID),
		Label:       headerTabTemplateEscapeText(ctx.Label),
		Left:        headerTabTemplateEscapeText(ctx.Left),
		Right:       headerTabTemplateEscapeText(ctx.Right),
		Index:       ctx.Index,
		Count:       ctx.Count,
		First:       ctx.First,
		Last:        ctx.Last,
		Active:      ctx.Active,
		NotActive:   !ctx.Active,
		IsZoomMode:  ctx.ZoomMode,
		NotZoomMode: !ctx.ZoomMode,
	}); err != nil {
		return "", false
	}
	return out.String(), true
}

func paneChromeTemplateFuncs(ctx paneChromeTemplateContext) texttemplate.FuncMap {
	return texttemplate.FuncMap{
		"glyph": func() string {
			return headerTabTemplateEscapeText(ctx.Glyph)
		},
		"text": func() string {
			return headerTabTemplateEscapeText(ctx.Text)
		},
		"action": func() string {
			return headerTabTemplateEscapeText(ctx.ActionID)
		},
		"action_id": func() string {
			return headerTabTemplateEscapeText(ctx.ActionID)
		},
		"label": func() string {
			return headerTabTemplateEscapeText(ctx.Label)
		},
		"left": func() string {
			return headerTabTemplateEscapeText(ctx.Left)
		},
		"right": func() string {
			return headerTabTemplateEscapeText(ctx.Right)
		},
		"index": func() int {
			return ctx.Index
		},
		"count": func() int {
			return ctx.Count
		},
		"first": func() bool {
			return ctx.First
		},
		"last": func() bool {
			return ctx.Last
		},
		"active": func() bool {
			return ctx.Active
		},
		"not_active": func() bool {
			return !ctx.Active
		},
		"is_zoom_mode": func() bool {
			return ctx.ZoomMode
		},
		"not_zoom_mode": func() bool {
			return !ctx.ZoomMode
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
		"reset": func() string {
			return "[reset]"
		},
	}
}

func paneChromeTemplateSegments(rendered string, defaultStyle StyleToken) []barSegment {
	if rendered == "" {
		return nil
	}
	state := paneChromeTemplateState{style: defaultStyle, defaultStyle: defaultStyle}
	segments := make([]barSegment, 0, 4)
	for len(rendered) > 0 {
		next := strings.Index(rendered, "[")
		if next < 0 {
			segments = appendPaneChromeTemplateText(segments, rendered, state)
			break
		}
		if next > 0 {
			segments = appendPaneChromeTemplateText(segments, rendered[:next], state)
			rendered = rendered[next:]
		}
		end := strings.Index(rendered, "]")
		if end < 0 {
			segments = appendPaneChromeTemplateText(segments, rendered[:1], state)
			rendered = rendered[1:]
			continue
		}
		tag := rendered[1:end]
		if nextState, ok := applyPaneChromeTemplateTag(tag, state); ok {
			state = nextState
			rendered = rendered[end+1:]
			continue
		}
		segments = appendPaneChromeTemplateText(segments, rendered[:end+1], state)
		rendered = rendered[end+1:]
	}
	return segments
}

func appendPaneChromeTemplateText(segments []barSegment, text string, state paneChromeTemplateState) []barSegment {
	text = headerTabTemplateUnescapeText(text)
	if text == "" {
		return segments
	}
	segment := barText(text, state.style, 1)
	segment.ansi = state.ansi
	return append(segments, segment)
}

func applyPaneChromeTemplateTag(tag string, state paneChromeTemplateState) (paneChromeTemplateState, bool) {
	tag = strings.TrimSpace(tag)
	switch {
	case tag == "/" || tag == "/style" || tag == "reset":
		state.style = state.defaultStyle
		state.ansi = ANSICellStyle{}
		return state, true
	case strings.HasPrefix(tag, "style:") || strings.HasPrefix(tag, "style="):
		if style, ok := headerTabTemplateStyleToken(headerTabTemplateTagArgument(tag)); ok {
			state.style = style
			state.ansi = ANSICellStyle{}
			return state, true
		}
		return state, false
	default:
		if update, ok := headerTabTemplateANSIUpdate(tag); ok {
			state.style = ""
			state.ansi = update.apply(state.ansi)
			return state, true
		}
		return state, false
	}
}

func paneChromeSegmentsText(segments []barSegment) string {
	if len(segments) == 0 {
		return ""
	}
	var out strings.Builder
	for _, segment := range segments {
		out.WriteString(segment.text)
	}
	return out.String()
}

func paneChromeSegmentsWidth(segments []barSegment) int {
	return barSegmentsWidth(segments)
}

func paneChromeMarkupWidth(markup string) int {
	return paneChromeSegmentsWidth(paneChromeTemplateSegments(markup, ""))
}

func paneChromeLineFromSegments(segments []barSegment, fallback StyleToken) Line {
	if len(segments) == 0 {
		return Line{}
	}
	cells := make([]Cell, 0, len(segments))
	for _, segment := range segments {
		if segment.text == "" {
			continue
		}
		style := segment.style
		if style == "" && segment.ansi.IsZero() {
			style = fallback
		}
		cells = append(cells, Cell{Text: segment.text, Width: DisplayWidth(segment.text), Style: style, ANSIStyle: segment.ansi, Safe: true})
	}
	return Line{Cells: cells}
}
