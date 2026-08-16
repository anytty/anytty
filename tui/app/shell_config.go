package app

import (
	"strings"

	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

func applyConfiguredShellChrome(shell state.ShellStore, cfg state.TUIConfigStore) state.ShellStore {
	switch presentation := state.PanelPresentation(strings.TrimSpace(cfg.Chrome.PanelPresentation)); presentation {
	case state.PanelPresentationCard, state.PanelPresentationSplitLine:
		// 中文说明：panel presentation 是本地 chrome 偏好；配置在 runtime 启动/
		// workbench restore 边界写入 ShellStore，之后按键切换仍由 ShellStore 持有。
		return shell.SetPanelPresentation(presentation)
	default:
		return shell
	}
}

func applyConfiguredPaneChromeGlyphs(cfg state.TUIConfigStore) {
	glyphs := cfg.Chrome.PaneGlyphs
	if !paneChromeGlyphConfigSet(glyphs) {
		return
	}
	// 中文说明：pane chrome glyph 配置只进入 render 字形表；ActionID、hit region
	// 和 reducer-owned pane 状态仍走原有消息链路，不能被配置改写。
	render.SetPaneChromeGlyphs(render.PaneChromeGlyphs{
		ActionLeft:                glyphs.ActionLeft,
		ActionLeftSet:             glyphs.ActionLeftSet,
		ActionRight:               glyphs.ActionRight,
		ActionRightSet:            glyphs.ActionRightSet,
		ActionSeparator:           glyphs.ActionSeparator,
		ActionSeparatorSet:        glyphs.ActionSeparatorSet,
		ActionGroupLeft:           glyphs.ActionGroupLeft,
		ActionGroupLeftSet:        glyphs.ActionGroupLeftSet,
		ActionGroupRight:          glyphs.ActionGroupRight,
		ActionGroupRightSet:       glyphs.ActionGroupRightSet,
		OwnerLeft:                 glyphs.OwnerLeft,
		OwnerLeftSet:              glyphs.OwnerLeftSet,
		OwnerRight:                glyphs.OwnerRight,
		OwnerRightSet:             glyphs.OwnerRightSet,
		Owner:                     glyphs.Owner,
		OwnerSet:                  glyphs.OwnerSet,
		OwnerPending:              glyphs.OwnerPending,
		OwnerPendingSet:           glyphs.OwnerPendingSet,
		TakeOwner:                 glyphs.TakeOwner,
		TakeOwnerSet:              glyphs.TakeOwnerSet,
		Zoom:                      glyphs.Zoom,
		Unzoom:                    glyphs.Unzoom,
		SplitVertical:             glyphs.SplitVertical,
		SplitHorizontal:           glyphs.SplitHorizontal,
		Close:                     glyphs.Close,
		SizeLock:                  glyphs.SizeLock,
		SizeUnlock:                glyphs.SizeUnlock,
		CenterFloating:            glyphs.CenterFloating,
		CollapseFloating:          glyphs.CollapseFloating,
		Running:                   glyphs.Running,
		Waiting:                   glyphs.Waiting,
		Exited:                    glyphs.Exited,
		Killed:                    glyphs.Killed,
		OverflowLeft:              glyphs.OverflowLeft,
		OverflowLeftSet:           glyphs.OverflowLeftSet,
		OverflowRight:             glyphs.OverflowRight,
		OverflowRightSet:          glyphs.OverflowRightSet,
		OverflowTop:               glyphs.OverflowTop,
		OverflowTopSet:            glyphs.OverflowTopSet,
		OverflowBottom:            glyphs.OverflowBottom,
		OverflowBottomSet:         glyphs.OverflowBottomSet,
		OverflowStyle:             glyphs.OverflowStyle,
		OverflowStyleSet:          glyphs.OverflowStyleSet,
		ExtentPlaceholder:         glyphs.ExtentPlaceholder,
		ExtentPlaceholderSet:      glyphs.ExtentPlaceholderSet,
		ExtentPlaceholderStyle:    glyphs.ExtentPlaceholderStyle,
		ExtentPlaceholderStyleSet: glyphs.ExtentPlaceholderStyleSet,
	})
}

func paneChromeGlyphConfigSet(glyphs state.TUIPaneChromeGlyphsConfig) bool {
	return glyphs.ActionLeftSet ||
		glyphs.ActionRightSet ||
		glyphs.ActionSeparatorSet ||
		glyphs.ActionGroupLeftSet ||
		glyphs.ActionGroupRightSet ||
		glyphs.OwnerLeftSet ||
		glyphs.OwnerRightSet ||
		glyphs.OwnerSet ||
		glyphs.OwnerPendingSet ||
		glyphs.TakeOwnerSet ||
		strings.TrimSpace(glyphs.Zoom) != "" ||
		strings.TrimSpace(glyphs.Unzoom) != "" ||
		strings.TrimSpace(glyphs.SplitVertical) != "" ||
		strings.TrimSpace(glyphs.SplitHorizontal) != "" ||
		strings.TrimSpace(glyphs.Close) != "" ||
		strings.TrimSpace(glyphs.SizeLock) != "" ||
		strings.TrimSpace(glyphs.SizeUnlock) != "" ||
		strings.TrimSpace(glyphs.CenterFloating) != "" ||
		strings.TrimSpace(glyphs.CollapseFloating) != "" ||
		strings.TrimSpace(glyphs.Running) != "" ||
		strings.TrimSpace(glyphs.Waiting) != "" ||
		strings.TrimSpace(glyphs.Exited) != "" ||
		strings.TrimSpace(glyphs.Killed) != "" ||
		glyphs.OverflowLeftSet ||
		glyphs.OverflowRightSet ||
		glyphs.OverflowTopSet ||
		glyphs.OverflowBottomSet ||
		glyphs.OverflowStyleSet ||
		glyphs.ExtentPlaceholderSet ||
		glyphs.ExtentPlaceholderStyleSet
}
