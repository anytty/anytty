package render

import actiondomain "github.com/anytty/anytty/tui/action"

// ProjectionID 是 render-local 的视觉投影编号。
// 它只能定位 footer/chrome/content metadata，不能作为 canonical action identity 或 app handler key。
type ProjectionID string

func (id ProjectionID) String() string {
	return string(id)
}

const (
	ActionPaneFocus               ProjectionID = "pane.focus"
	ActionPaneResize              ProjectionID = "pane.resize"
	ActionPaneSplitDown           ProjectionID = "pane.split-down"
	ActionPaneSplitRight          ProjectionID = "pane.split-right"
	ActionPaneZoom                ProjectionID = "pane.zoom"
	ActionPaneClose               ProjectionID = "pane.close"
	ActionResizeLayoutLock        ProjectionID = "resize.layout-lock"
	ActionTerminalTakeResizeOwner ProjectionID = "terminal.resize-owner.take"

	ActionTabCreate ProjectionID = "tab.create"
	ActionTabSwitch ProjectionID = "tab.switch"
	ActionTabClose  ProjectionID = "tab.close"

	ActionFloatingRaise      ProjectionID = "floating.raise"
	ActionFloatingSummon     ProjectionID = "floating.summon"
	ActionFloatingClose      ProjectionID = "floating.close"
	ActionFloatingCenter     ProjectionID = "floating.center"
	ActionFloatingCollapse   ProjectionID = "floating.collapse"
	ActionFloatingMoveDrag   ProjectionID = "floating.move-drag"
	ActionFloatingResizeDrag ProjectionID = "floating.resize-drag"

	ActionEmptyAttach  ProjectionID = "empty.attach"
	ActionEmptyCreate  ProjectionID = "empty.create"
	ActionEmptyManager ProjectionID = "empty.manager"
	ActionEmptyClose   ProjectionID = "empty.close"

	ActionExitedRestart   ProjectionID = "exited.restart"
	ActionExitedReconnect ProjectionID = "exited.reconnect"
	ActionExitedClose     ProjectionID = "exited.close"

	ActionDisconnectedReconnect  ProjectionID = "disconnected.reconnect"
	ActionDisconnectedDisconnect ProjectionID = "disconnected.disconnect"

	ActionPickerAttach         ProjectionID = "picker.attach"
	ActionPickerNew            ProjectionID = "picker.new"
	ActionPickerEndpointSelect ProjectionID = "picker.endpoint-select"
	ActionPickerStatusSelect   ProjectionID = "picker.status-select"
	ActionPickerTags           ProjectionID = "picker.tags"
	ActionPickerTagToggle      ProjectionID = "picker.tag-toggle"

	ActionPoolSelect ProjectionID = "pool.select"

	ActionWorkbenchOpen ProjectionID = "workbench.open"

	ActionClipboardHistorySelect      ProjectionID = "clipboard-history.select"
	ActionClipboardHistoryDividerDrag ProjectionID = "clipboard-history.divider-drag"

	ActionHelpClose ProjectionID = "help.close"
)

type ActionSurface string

const (
	ActionSurfacePaneChrome     ActionSurface = "pane-chrome"
	ActionSurfaceFloatingChrome ActionSurface = "floating-chrome"
	ActionSurfaceContent        ActionSurface = "content"
	ActionSurfaceHelp           ActionSurface = "help"
	ActionSurfaceLayout         ActionSurface = "layout"
)

// ProjectionSpec 只描述 render surface 的视觉和布局元数据。
// Canonical identity/label 属于 tui/action，handler/dispatch 属于 app；本类型不能重新声明执行语义。
type ProjectionSpec struct {
	ID                ProjectionID
	CanonicalActionID actiondomain.ID
	Surfaces          []ActionSurface
	ChromeGlyph       string
	Danger            bool
}

func (spec ProjectionSpec) HasSurface(surface ActionSurface) bool {
	for _, candidate := range spec.Surfaces {
		if candidate == surface {
			return true
		}
	}
	return false
}

func projectionSpec(id ProjectionID, surfaces ...ActionSurface) ProjectionSpec {
	return ProjectionSpec{ID: id, CanonicalActionID: canonicalActionForProjection(id), Surfaces: surfaces}
}

func (spec ProjectionSpec) withChromeGlyph(glyph string) ProjectionSpec {
	spec.ChromeGlyph = glyph
	return spec
}

func (spec ProjectionSpec) withDanger() ProjectionSpec {
	spec.Danger = true
	return spec
}

// ProjectionCatalog 返回现有 render surface 的视觉/布局投影清单。
// ProjectionID 只在 render 内定位 metadata；每个生产投影都必须用 CanonicalActionID 引用 tui/action。
func ProjectionCatalog() []ProjectionSpec {
	return []ProjectionSpec{
		projectionSpec(ActionPaneFocus, ActionSurfacePaneChrome, ActionSurfaceLayout),
		projectionSpec(ActionPaneResize, ActionSurfaceLayout),
		projectionSpec(ActionPaneSplitDown, ActionSurfacePaneChrome, ActionSurfaceHelp).withChromeGlyph(paneChromeSplitHorizontalActionText()),
		projectionSpec(ActionPaneSplitRight, ActionSurfacePaneChrome, ActionSurfaceHelp).withChromeGlyph(paneChromeSplitVerticalActionText()),
		projectionSpec(ActionPaneZoom, ActionSurfacePaneChrome, ActionSurfaceFloatingChrome, ActionSurfaceHelp).withChromeGlyph(paneChromeZoomGlyph()),
		projectionSpec(ActionPaneClose, ActionSurfacePaneChrome, ActionSurfaceHelp).withChromeGlyph(paneChromeCloseActionText()).withDanger(),
		projectionSpec(ActionResizeLayoutLock, ActionSurfacePaneChrome),
		projectionSpec(ActionTerminalTakeResizeOwner, ActionSurfacePaneChrome, ActionSurfaceHelp).withChromeGlyph(paneChromeTakeOwnerText()),
		projectionSpec(ActionTabCreate, ActionSurfaceLayout),
		projectionSpec(ActionTabSwitch, ActionSurfaceLayout),
		projectionSpec(ActionTabClose, ActionSurfaceLayout).withDanger(),
		projectionSpec(ActionFloatingRaise, ActionSurfaceFloatingChrome, ActionSurfaceContent, ActionSurfaceHelp).withChromeGlyph(paneChromeZoomGlyph()),
		projectionSpec(ActionFloatingSummon, ActionSurfaceContent),
		projectionSpec(ActionFloatingClose, ActionSurfaceFloatingChrome).withChromeGlyph(paneChromeCloseGlyph()).withDanger(),
		projectionSpec(ActionFloatingCenter, ActionSurfaceFloatingChrome).withChromeGlyph(paneChromeFloatingCenterGlyph()),
		projectionSpec(ActionFloatingCollapse, ActionSurfaceFloatingChrome).withChromeGlyph(paneChromeFloatingCollapseGlyph()),
		projectionSpec(ActionFloatingMoveDrag, ActionSurfaceFloatingChrome, ActionSurfaceLayout, ActionSurfaceHelp),
		projectionSpec(ActionFloatingResizeDrag, ActionSurfaceFloatingChrome, ActionSurfaceLayout),
		projectionSpec(ActionEmptyAttach, ActionSurfaceContent),
		projectionSpec(ActionEmptyCreate, ActionSurfaceContent),
		projectionSpec(ActionEmptyManager, ActionSurfaceContent),
		projectionSpec(ActionEmptyClose, ActionSurfaceContent).withDanger(),
		projectionSpec(ActionExitedRestart, ActionSurfaceContent),
		projectionSpec(ActionExitedReconnect, ActionSurfaceContent),
		projectionSpec(ActionExitedClose, ActionSurfaceContent).withDanger(),
		projectionSpec(ActionDisconnectedReconnect, ActionSurfaceContent),
		projectionSpec(ActionDisconnectedDisconnect, ActionSurfaceContent).withDanger(),
		projectionSpec(ActionPickerAttach, ActionSurfaceContent),
		projectionSpec(ActionPickerNew, ActionSurfaceContent),
		projectionSpec(ActionPickerEndpointSelect, ActionSurfaceContent),
		projectionSpec(ActionPickerStatusSelect, ActionSurfaceContent),
		projectionSpec(ActionPickerTags, ActionSurfaceContent),
		projectionSpec(ActionPickerTagToggle, ActionSurfaceContent),
		projectionSpec(ActionPoolSelect, ActionSurfaceContent),
		projectionSpec(ActionWorkbenchOpen, ActionSurfaceContent),
		projectionSpec(ActionClipboardHistorySelect, ActionSurfaceContent),
		projectionSpec(ActionClipboardHistoryDividerDrag, ActionSurfaceContent),
		projectionSpec(ActionHelpClose, ActionSurfaceContent),
	}
}

// ProjectionByID 返回 render-local 投影；动态 glyph 只改变视觉文本，不改变 canonical action 引用。
func ProjectionByID(id ProjectionID) (ProjectionSpec, bool) {
	spec, ok := projectionByIDCatalog[id]
	if !ok {
		return ProjectionSpec{}, false
	}
	return projectionWithCurrentGlyph(spec), true
}

var projectionByIDCatalog = buildProjectionByIDCatalog()

func buildProjectionByIDCatalog() map[ProjectionID]ProjectionSpec {
	specs := ProjectionCatalog()
	out := make(map[ProjectionID]ProjectionSpec, len(specs))
	for _, spec := range specs {
		if spec.CanonicalActionID == "" {
			panic("render projection has no canonical action " + spec.ID.String())
		}
		if _, ok := actiondomain.SpecByID(spec.CanonicalActionID); !ok {
			panic("render projection references unknown canonical action " + spec.CanonicalActionID.String())
		}
		out[spec.ID] = spec
	}
	return out
}

func projectionWithCurrentGlyph(spec ProjectionSpec) ProjectionSpec {
	switch spec.ID {
	case ActionPaneSplitDown:
		spec.ChromeGlyph = paneChromeSplitHorizontalActionText()
	case ActionPaneSplitRight:
		spec.ChromeGlyph = paneChromeSplitVerticalActionText()
	case ActionPaneZoom, ActionFloatingRaise:
		spec.ChromeGlyph = paneChromeZoomGlyph()
	case ActionPaneClose:
		spec.ChromeGlyph = paneChromeCloseActionText()
	case ActionFloatingClose:
		spec.ChromeGlyph = paneChromeCloseGlyph()
	case ActionFloatingCenter:
		spec.ChromeGlyph = paneChromeFloatingCenterGlyph()
	case ActionFloatingCollapse:
		spec.ChromeGlyph = paneChromeFloatingCollapseGlyph()
	case ActionTerminalTakeResizeOwner:
		spec.ChromeGlyph = paneChromeTakeOwnerText()
	}
	return spec
}
