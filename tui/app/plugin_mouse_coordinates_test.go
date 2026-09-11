package app

import (
	"testing"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/render"
)

func TestPluginMouseUsesSGROneBasedRenderedCoordinates(t *testing.T) {
	root, deps, _ := pluginFixture(t)
	vm := render.NewRenderVMBuilder().Build(root)
	plan := render.MeasureLayout(vm.Shell, render.Rect{W: 100, H: 30})
	runtime := &AppRuntime{state: root, lastHitRegions: plan.HitRegions}
	var first, second render.HitRegion
	for _, hit := range plan.HitRegions {
		if hit.Kind == render.HitRegionPlugin && hit.PluginNodeID == "a" {
			first = hit
		}
		if hit.Kind == render.HitRegionPlugin && hit.PluginNodeID == "b" {
			second = hit
		}
	}
	if first.Rect.H != 1 || second.Rect.Y != first.Rect.Y+1 {
		t.Fatal("missing rendered row regions")
	}
	for _, test := range []struct {
		name     string
		row, col int
		want     string
	}{
		{"first row left edge", first.Rect.Y + 1, first.Rect.X + 1, "a"},
		{"first row right edge", first.Rect.Y + 1, first.Rect.X + first.Rect.W, "a"},
		{"second row", second.Rect.Y + 1, second.Rect.X + 1, "b"},
		{"hint is not a row", first.Rect.Y, first.Rect.X + 1, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := runtime.dispatchMouseHitRegion(InputMsg{Event: input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseLeft, Row: test.row, Col: test.col}})
			plugin, ok := msg.(PluginInputMsg)
			if !ok || plugin.NodeID != test.want {
				t.Fatalf("SGR (%d,%d) selected wrong rendered node: %#v", test.col, test.row, msg)
			}
			if test.want != "" {
				next, _ := NewPluginReducer(deps)(root, plugin)
				if next.Plugins.Mounts["agents"].SelectedID != test.want {
					t.Fatal("mouse selection differs from rendered hit row")
				}
			}
		})
	}
}
