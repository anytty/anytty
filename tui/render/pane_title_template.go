package render

import (
	"bytes"
	"strings"
	texttemplate "text/template"
)

type terminalChromeTitleTemplateContext struct {
	Terminal      string
	TerminalTitle string
	TerminalID    string
	Endpoint      string
	EndpointLabel string
	EndpointID    string
	Pane          string
	PaneTitle     string
}

func renderTerminalChromeTitleTemplate(format string, ctx terminalChromeTitleTemplateContext) (string, bool) {
	format = strings.TrimSpace(format)
	if format == "" {
		return "", false
	}
	// 中文说明：pane title 模板只生成 chrome 展示文字；TerminalRef 路由和 pane storage
	// 仍由 reducer-owned binding 持有，模板不能注入 action 或改写 endpoint identity。
	tmpl, err := texttemplate.New("terminal_chrome_title").Option("missingkey=zero").Funcs(terminalChromeTitleTemplateFuncs(ctx)).Parse(format)
	if err != nil {
		return "", false
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, ctx); err != nil {
		return "", false
	}
	return out.String(), true
}

func terminalChromeTitleTemplateFuncs(ctx terminalChromeTitleTemplateContext) texttemplate.FuncMap {
	return texttemplate.FuncMap{
		"terminal": func() string {
			return ctx.Terminal
		},
		"terminal_title": func() string {
			return ctx.TerminalTitle
		},
		"terminal_id": func() string {
			return ctx.TerminalID
		},
		"endpoint": func() string {
			return ctx.Endpoint
		},
		"endpoint_label": func() string {
			return ctx.EndpointLabel
		},
		"endpoint_id": func() string {
			return ctx.EndpointID
		},
		"pane": func() string {
			return ctx.Pane
		},
		"pane_title": func() string {
			return ctx.PaneTitle
		},
		"truncate": func(width int, value string) string {
			if width <= 0 {
				return ""
			}
			return TruncateCells(value, width)
		},
	}
}
