package render

import actiondomain "github.com/anytty/anytty/tui/action"

func footerActionTokensVisibleByModeAndWidth(actions []FooterActionVM, mode string, width int) []FooterActionVM {
	if mode == "resize" {
		return footerResizeActionTokensVisibleForWidth(actions, width)
	}
	if width >= 100 || mode != "live" && mode != "normal" {
		return actions
	}
	ids := []actiondomain.ID{"menu.panel", "menu.resize", "menu.tab", "menu.system"}
	if width < 72 {
		ids = []actiondomain.ID{"menu.panel", "menu.resize", "menu.system"}
	}
	if width < 56 {
		ids = []actiondomain.ID{"menu.panel", "menu.system"}
	}
	out := footerActionsByCanonicalIDs(actions, ids)
	if len(out) == 0 {
		return actions
	}
	return out
}

func footerResizeActionTokensVisibleForWidth(actions []FooterActionVM, width int) []FooterActionVM {
	if width >= 100 {
		return actions
	}
	// 中文说明：resize 窄宽 footer 不能只保留 global；center/reset 是当前 mode 仍可执行的 view-layout 入口。
	ids := []actiondomain.ID{
		"resize.center",
		"resize.left",
		"resize.right",
		"resize.layout_reset",
		"menu.system",
	}
	if width < 72 {
		ids = []actiondomain.ID{
			"resize.center",
			"resize.layout_reset",
			"menu.system",
		}
	}
	out := footerActionsByCanonicalIDs(actions, ids)
	if len(out) == 0 {
		return actions
	}
	return out
}

func footerActionsByCanonicalIDs(actions []FooterActionVM, ids []actiondomain.ID) []FooterActionVM {
	out := make([]FooterActionVM, 0, len(ids))
	for _, id := range ids {
		for _, action := range actions {
			// 合并后的多动作提示没有唯一 Invocation；ActionID 在这里仅用于选择视觉 token，
			// ClickHintOnly 保证它不会重新成为执行身份或点击 fallback。
			if action.Invocation.ID == id || action.Click == ClickHintOnly && action.ActionID == id.String() {
				out = append(out, action)
				break
			}
		}
	}
	return out
}
