package render

import (
	"github.com/anytty/anytty/tui/state"
	"strings"
	"testing"
)

func TestPluginSidebarReservesTerminalSpaceAndHits(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell(), Viewport: state.ViewportStore{Cols: 100, Rows: 30, Valid: true}}
	m := state.PluginMount{ID: "agents", PluginID: "p", DaemonID: "d", Owner: state.PluginOwner{Kind: "workspace", WorkspaceID: root.Shell.Workspace.ID}, Slot: "sidebar", Title: "Agents", Revision: 1, Interactive: true, Nodes: []state.PluginNode{{ID: "a", Text: "Codex", Action: "open"}}}
	var err error
	root.Plugins, err = root.Plugins.Apply(root.Shell, m, 0)
	if err != nil {
		t.Fatal(err)
	}
	vm := NewRenderVMBuilder().Build(root)
	plan := MeasureLayout(vm.Shell, Rect{W: 100, H: 30})
	if len(plan.Plugins) != 1 || plan.Body.X < 30 || plan.Panels[0].ContentRect.X <= plan.Plugins[0].Rect.X {
		t.Fatal("sidebar must reserve real terminal layout space")
	}
	found := false
	for _, h := range plan.HitRegions {
		if h.Kind == HitRegionPlugin && h.PluginNodeID == "a" {
			found = true
		}
	}
	if !found {
		t.Fatal("plugin row missing hit target")
	}
	frame := NewRenderer(DefaultTheme()).Render(vm)
	_ = frame
}

func TestPluginCardsUseDeclaredSpacingStylesAndHitBounds(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell(), Viewport: state.ViewportStore{Cols: 100, Rows: 30, Valid: true}}
	m := state.PluginMount{
		ID: "cards", PluginID: "demo", DaemonID: "d",
		Owner: state.PluginOwner{Kind: "workspace", WorkspaceID: root.Shell.Workspace.ID},
		Slot:  "sidebar", Title: "Agents", Revision: 1, Interactive: true, SelectedID: "second",
		Nodes: []state.PluginNode{
			{ID: "first", Kind: "card", Text: "First", Description: "one", Layout: state.PluginNodeLayout{PaddingTop: 1, PaddingBottom: 1, GapAfter: 1}, Style: state.PluginNodeStyle{ForegroundRole: "success", BackgroundRole: "surface"}},
			{ID: "second", Kind: "card", Text: "Second", Description: "two", Layout: state.PluginNodeLayout{PaddingTop: 1, PaddingBottom: 1, GapAfter: 1}, SelectedStyle: state.PluginNodeStyle{ForegroundRole: "primary", BackgroundRole: "selected", Bold: true}},
		},
	}
	var err error
	root.Plugins, err = root.Plugins.Apply(root.Shell, m, 0)
	if err != nil {
		t.Fatal(err)
	}
	result := NewRenderer(DefaultTheme()).RenderResult(NewRenderVMBuilder().Build(root))
	text := strings.Join(result.Lines(), "\n")
	for _, want := range []string{"First", "one", "Second", "two"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing card content %q", want)
		}
	}
	var cards []HitRegion
	for _, hit := range result.HitRegions {
		if hit.Kind == HitRegionPlugin && hit.PluginNodeID != "" {
			cards = append(cards, hit)
		}
	}
	if len(cards) != 2 {
		t.Fatalf("expected two card hit regions, got %#v", cards)
	}
	if cards[0].Rect.H != 4 || cards[1].Rect.Y != cards[0].Rect.Y+5 {
		t.Fatalf("card spacing should follow declared layout, got %#v", cards)
	}
	foundSelectedStyle := false
	for _, line := range result.StyledLines() {
		for _, cell := range line.Cells {
			if cell.Style == "plugin/selected/primary/true/false" {
				foundSelectedStyle = true
			}
		}
	}
	if !foundSelectedStyle {
		t.Fatal("selected card should use its declared semantic style")
	}
}

func TestPluginFormControlsRenderValuesAndOwnCursor(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell(), Viewport: state.ViewportStore{Cols: 100, Rows: 30, Valid: true}}
	m := state.PluginMount{ID: "form", PluginID: "demo", DaemonID: "d", Owner: state.PluginOwner{Kind: "workspace", WorkspaceID: root.Shell.Workspace.ID}, Slot: "overlay", Title: "Settings", Revision: 1, Interactive: true, SelectedID: "name", EditingID: "name", EditCursor: 2, Nodes: []state.PluginNode{{ID: "fields", Kind: "form", Children: []state.PluginNode{{ID: "name", Kind: "input", Text: "Name", Value: "demo"}, {ID: "enabled", Kind: "checkbox", Text: "Enabled", Value: "true"}, {ID: "save", Kind: "button", Text: "Save", Action: "save"}}}}}
	root.Plugins = root.Plugins.Set(m)
	root.Plugins.FocusedMountID = "form"
	result := NewRenderer(DefaultTheme()).RenderResult(NewRenderVMBuilder().Build(root))
	text := strings.Join(result.Lines(), "\n")
	for _, want := range []string{"Name: de│mo", "[x] Enabled", "[ Save ]"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing form control %q", want)
		}
	}
	if result.Cursor.Visible {
		t.Fatal("plugin editor must not show background terminal cursor")
	}
}
