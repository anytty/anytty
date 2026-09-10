package render

import (
	"reflect"
	"strings"
	"testing"
)

func TestInactivePanelFocusSwitchPreservesContent(t *testing.T) {
	content := ContentVM{Kind: ContentTerminalLive, Lines: []Line{{Cells: []Cell{{Text: "你好", Width: 4, TerminalContent: true, ANSIStyle: ANSICellStyle{FG: "#ffffff", BG: "#404040", Bold: true, Reverse: true}, LinkURL: "https://example.com", Safe: true}}}}}
	vm := RenderVM{Shell: ShellVM{Layout: LayoutVM{Viewport: Rect{W: 60, H: 12}, Split: SplitVM{Direction: SplitVertical, Children: []SplitVM{{PaneID: "a"}, {PaneID: "b"}}}, Panels: []PanelVM{
		{ID: "a", Active: true, Rect: Rect{W: 30, H: 12}, Content: content},
		{ID: "b", Rect: Rect{X: 30, W: 30, H: 12}, Content: content},
	}}}}
	theme := DefaultTheme()
	theme.DimInactivePanels = true
	renderer := NewRenderer(theme)
	before := renderer.RenderResult(vm)
	assertDim := func(result RenderResult, want []bool) {
		t.Helper()
		var got []bool
		for _, line := range result.Content {
			for _, cell := range line.Cells {
				if cell.Text == "你" {
					got = append(got, cell.Dimmed)
					if cell.LinkURL != "https://example.com" || cell.ANSIStyle != content.Lines[0].Cells[0].ANSIStyle {
						t.Fatal("source style/link changed")
					}
				}
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("dim flags = %v, want %v", got, want)
		}
	}
	assertDim(before, []bool{false, true})
	vm.Shell.Layout.Panels[0].Active = false
	vm.Shell.Layout.Panels[1].Active = true
	after := renderer.RenderResult(vm)
	assertDim(after, []bool{true, false})
	if !reflect.DeepEqual(before.Lines(), after.Lines()) {
		t.Fatal("focus changed plain content")
	}
	if reflect.DeepEqual(before.ANSILines(), after.ANSILines()) {
		t.Fatal("focus did not repaint colors")
	}
	theme.DimInactivePanels = false
	assertDim(NewRenderer(theme).RenderResult(vm), []bool{false, false})
	if content.Lines[0].Cells[0].Dimmed {
		t.Fatal("mutated source snapshot")
	}
}

func TestInactivePanelColors(t *testing.T) {
	theme := DefaultTheme()
	theme.HostFG = "#ffffff"
	theme.HostBG = "#000000"
	theme.TerminalPalette[1] = "#ff0000"
	for _, tc := range []struct{ name, seq, want string }{
		{"default", "", "38;2;128;128;128"},
		{"truecolor", "\x1b[1;7;38;2;100;200;50;48;2;200;100;0m", "\x1b[1;7;38;2;50;100;25;48;2;100;50;0m"},
		{"host palette", "\x1b[31m", "38;2;128;0;0"},
		{"indexed host palette", "\x1b[38;5;1m", "38;2;128;0;0"},
		{"color cube", "\x1b[48;5;196m", "48;2;128;0;0"},
		{"grayscale", "\x1b[38;5;232m", "38;2;4;4;4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := dimANSISequence(tc.seq, theme); !strings.Contains(got, tc.want) {
				t.Fatalf("got %q; want %q", got, tc.want)
			}
		})
	}
	theme.HostFG = "#000000"
	theme.HostBG = "#ffffff"
	if got := dimANSISequence("", theme); !strings.Contains(got, "38;2;128;128;128") {
		t.Fatalf("light background contrast: %q", got)
	}
}

func TestInactivePanelDoesNotDimHigherLayers(t *testing.T) {
	c := newCanvas(12, 3)
	for x := 0; x < 12; x++ {
		c.writeText(x, 0, 1, "b")
	}
	c.dimRect(Rect{X: -1, Y: -1, W: 20, H: 20})
	c.writeText(2, 0, 5, "front")
	if c.rows[0][2].dimmed {
		t.Fatal("higher layer inherited dimming")
	}
	if !c.rows[0][0].dimmed {
		t.Fatal("background lost dimming")
	}
}

func TestInactivePanelDimAmount(t *testing.T) {
	for _, tc := range []struct {
		amount float64
		want   string
	}{{0, "38;2;100;200;50"}, {0.5, "38;2;50;100;25"}, {1, "38;2;0;0;0"}} {
		theme := DefaultTheme()
		theme.HostBG = "#000000"
		theme.InactivePanelDimAmount = tc.amount
		got := dimANSISequence("\x1b[38;2;100;200;50m", theme)
		if !strings.Contains(got, tc.want) {
			t.Fatalf("amount %v: got %q, want %q", tc.amount, got, tc.want)
		}
	}
}
