package render

import "strings"

func footerModeBadgeSegments(footer FooterVM, mode string) []barSegment {
	icon := strings.TrimSpace(footer.ModeIcon)
	label := strings.TrimSpace(footer.ModeLabel)
	if icon == "" && label == "" {
		if mode == "live" || mode == "normal" || mode == string(OverlayClipboardHistory) {
			return nil
		}
		if mode == string(OverlayTerminalPool) {
			label = "TERMINALS"
		} else {
			label = strings.ToUpper(mode)
		}
	}
	text := footerTokenTemplateText(footerTemplateOrDefault(footer.ModeBadgeTemplate, defaultFooterModeBadgeTemplate), map[string]string{
		"mode":       mode,
		"mode_icon":  icon,
		"mode_label": label,
	})
	if text == "" {
		return nil
	}
	style := footer.ModeStyle
	if style == "" {
		style = StyleFooterAccent
	}
	return []barSegment{barText(" "+text+" ", style, 1)}
}

func footerActionDecorText(action FooterActionVM, template string) string {
	icon := strings.TrimSpace(action.Icon)
	label := strings.TrimSpace(action.Label)
	if icon == "" && label == "" {
		return ""
	}
	if icon == "" {
		label = strings.ToUpper(label)
	}
	if strings.TrimSpace(template) == "" {
		template = defaultFooterActionTemplate
	}
	// key 仍由 structured key segment 渲染，模板里的 {{key}} 只作为兼容占位符吞掉。
	return footerTokenTemplateText(template, map[string]string{
		"key":    "",
		"icon":   icon,
		"label":  label,
		"action": strings.TrimSpace(action.ActionID),
	})
}

func footerKeyTemplateForFooter(footer FooterVM) string {
	if footer.KeyTemplateSet {
		return footer.KeyTemplate
	}
	return defaultFooterKeyTemplate
}

func footerActionKeyText(key string, template string) string {
	key = strings.TrimSpace(key)
	if key == "" || strings.TrimSpace(template) == "" {
		return ""
	}
	return footerTokenTemplateText(template, map[string]string{
		"key": key,
	})
}

func footerSummaryTemplateText(template string, fallback string, primaryKey string, value string) string {
	template = footerTemplateOrDefault(template, fallback)
	return footerTokenTemplateText(template, map[string]string{
		primaryKey: value,
		"value":    value,
		"count":    value,
	})
}

func footerTokenTemplateText(template string, values map[string]string) string {
	template = strings.TrimSpace(template)
	if template == "" {
		return ""
	}
	out := template
	for key, value := range values {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
		out = strings.ReplaceAll(out, "{{ "+key+" }}", value)
	}
	return strings.Join(strings.Fields(out), " ")
}
