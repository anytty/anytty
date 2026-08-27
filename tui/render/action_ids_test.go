package render

import (
	"testing"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/state"
)

func TestProjectionCatalogIsSingleSourceForRenderedActions(t *testing.T) {
	specs := ProjectionCatalog()
	if len(specs) == 0 {
		t.Fatal("action spec catalog must not be empty")
	}
	seen := map[ProjectionID]struct{}{}
	for _, spec := range specs {
		if spec.ID == "" {
			t.Fatal("action id catalog must not contain empty id")
		}
		if _, ok := seen[spec.ID]; ok {
			t.Fatalf("duplicate action id %q", spec.ID)
		}
		if len(spec.Surfaces) == 0 {
			t.Fatalf("action spec %q must declare at least one surface", spec.ID)
		}
		if spec.CanonicalActionID != "" {
			if canonical, ok := actiondomain.SpecByID(spec.CanonicalActionID); !ok || canonical.ID != spec.CanonicalActionID {
				t.Fatalf("projection %q references non-canonical action %q", spec.ID, spec.CanonicalActionID)
			}
		}
		seen[spec.ID] = struct{}{}
	}
	for _, id := range []ProjectionID{
		ActionPaneFocus, ActionPaneResize, ActionPaneSplitDown, ActionPaneSplitRight, ActionPaneZoom, ActionPaneClose,
		ActionResizeLayoutLock, ActionTerminalTakeResizeOwner,
		ActionTabCreate, ActionTabSwitch, ActionTabClose,
		ActionFloatingRaise, ActionFloatingSummon, ActionFloatingClose, ActionFloatingCenter, ActionFloatingCollapse,
		ActionFloatingMoveDrag, ActionFloatingResizeDrag,
		ActionEmptyAttach, ActionEmptyCreate, ActionEmptyManager, ActionEmptyClose,
		ActionExitedRestart, ActionExitedReconnect, ActionExitedClose,
		ActionDisconnectedReconnect, ActionDisconnectedDisconnect,
		ActionPickerAttach, ActionPickerNew, ActionPickerEndpointSelect, ActionPickerStatusSelect,
		ActionPickerTags, ActionPickerTagToggle, ActionPoolSelect, ActionWorkbenchOpen,
		ActionClipboardHistorySelect, ActionClipboardHistoryDividerDrag, ActionHelpClose,
	} {
		if _, ok := seen[id]; !ok {
			t.Fatalf("action id %q must be registered in catalog", id)
		}
	}
}

func TestProjectionCatalogClassifiesVisibleActions(t *testing.T) {
	assertSpec := func(id ProjectionID, surface ActionSurface) ProjectionSpec {
		t.Helper()
		spec, ok := ProjectionByID(id)
		if !ok {
			t.Fatalf("missing action spec %q", id)
		}
		if !spec.HasSurface(surface) {
			t.Fatalf("action %q must include surface %q, got %#v", id, surface, spec.Surfaces)
		}
		return spec
	}

	pane := assertSpec(ActionPaneClose, ActionSurfacePaneChrome)
	if pane.ChromeGlyph == "" || !pane.Danger {
		t.Fatalf("pane close action should carry chrome glyph and danger metadata, got %#v", pane)
	}

	floating := assertSpec(ActionFloatingClose, ActionSurfaceFloatingChrome)
	if floating.ChromeGlyph == "" || !floating.Danger {
		t.Fatalf("floating close action should carry chrome glyph and danger metadata, got %#v", floating)
	}

	assertSpec(ActionHelpClose, ActionSurfaceContent)
	assertSpec(ActionPoolSelect, ActionSurfaceContent)
	assertSpec(ActionClipboardHistoryDividerDrag, ActionSurfaceContent)
}

func TestProjectionByIDKeepsDynamicChromeGlyphsCurrent(t *testing.T) {
	t.Cleanup(ResetPaneChromeGlyphs)
	SetPaneChromeGlyphs(PaneChromeGlyphs{Close: "❌", Zoom: "🔎", SplitVertical: "↕", SplitHorizontal: "↔"})

	for _, tt := range []struct {
		id   ProjectionID
		want string
	}{
		{id: ActionPaneClose, want: paneChromeCloseActionText()},
		{id: ActionFloatingClose, want: paneChromeCloseGlyph()},
		{id: ActionPaneZoom, want: paneChromeZoomGlyph()},
		{id: ActionFloatingRaise, want: paneChromeZoomGlyph()},
		{id: ActionPaneSplitRight, want: paneChromeSplitVerticalActionText()},
		{id: ActionPaneSplitDown, want: paneChromeSplitHorizontalActionText()},
	} {
		spec, ok := ProjectionByID(tt.id)
		if !ok {
			t.Fatalf("missing action spec %q", tt.id)
		}
		if spec.ChromeGlyph != tt.want {
			t.Fatalf("action %q chrome glyph got=%q want=%q", tt.id, spec.ChromeGlyph, tt.want)
		}
	}
}

func TestProjectionCatalogCoversVisibleVMAndHitRegionActions(t *testing.T) {
	root := actionCatalogRoot()
	vm := NewRenderVMBuilder().Build(root)
	plan := MeasureLayout(vm.Shell, Rect{W: 80, H: 24})

	for _, action := range vm.Shell.Footer.ActionTokens {
		if _, ok := actiondomain.SpecByID(actiondomain.ID(action.ActionID)); !ok {
			t.Fatalf("footer action %q must use canonical action identity", action.ActionID)
		}
	}
	for _, panel := range vm.Shell.Layout.Panels {
		for _, action := range panel.Chrome.Actions {
			assertRegisteredAction(t, action.ActionID, ActionSurfacePaneChrome)
		}
	}
	for _, floating := range vm.Shell.Layout.Floating {
		for _, action := range floatingChromeActionItems(floating.Rect.W) {
			assertRegisteredAction(t, action.ActionID, ActionSurfaceFloatingChrome)
		}
	}
	for _, region := range plan.HitRegions {
		if region.ActionID == "" {
			continue
		}
		if region.Invocation.ID != "" {
			if _, ok := actiondomain.SpecByID(region.Invocation.ID); !ok {
				t.Fatalf("hit region invocation %q must be canonical", region.Invocation.ID)
			}
			continue
		}
		assertRegisteredAction(t, region.ActionID, "")
	}
}

func actionCatalogRoot() state.Root {
	shell, _ := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneEmpty}, state.SplitDirectionVertical).
		ApplyFloatingCommand(state.FloatingCommand{
			Action:   state.FloatingCommandCreate,
			TargetID: "float-1",
			Pane:     state.PaneState{ID: "float-pane-1", Title: "floating", Kind: state.PaneEmpty},
			Title:    "floating",
			BoundsW:  80,
			BoundsH:  24,
		})
	return state.Root{
		Viewport: state.ViewportStore{Cols: 80, Rows: 24, Valid: true},
		Shell:    shell,
	}
}

func assertRegisteredAction(t *testing.T, id string, surface ActionSurface) ProjectionSpec {
	t.Helper()
	if id == "" {
		return ProjectionSpec{}
	}
	spec, ok := ProjectionByID(ProjectionID(id))
	if !ok {
		t.Fatalf("action %q must be registered in ProjectionCatalog", id)
	}
	if surface != "" && !spec.HasSurface(surface) {
		t.Fatalf("action %q must include surface %q, got %#v", id, surface, spec.Surfaces)
	}
	return spec
}
