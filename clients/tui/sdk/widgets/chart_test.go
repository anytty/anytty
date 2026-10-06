package widgets

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
)

func TestSparklineGlyphMapping(t *testing.T) {
	if got := (Sparkline{Values: []float64{0, 1, 2, 3, 4, 5, 6, 7}}).Line(); got != "▁▂▃▄▅▆▇█" {
		t.Fatalf("sparkline = %q, want full ramp", got)
	}
	if got := (Sparkline{Values: []float64{5, 5, 5}}).Line(); got != "▄▄▄" {
		t.Fatalf("all-equal sparkline = %q, want mid glyph", got)
	}
	if got := (Sparkline{Values: []float64{42}}).Line(); got != "▄" {
		t.Fatalf("one-value sparkline = %q, want mid glyph", got)
	}
	if got := (Sparkline{}).Line(); got != "" {
		t.Fatalf("empty sparkline = %q, want empty", got)
	}
	min, max := 0.0, 10.0
	if got := (Sparkline{Values: []float64{0, 5, 10}, Min: &min, Max: &max}).Line(); got != "▁▅█" {
		t.Fatalf("explicit bounds sparkline = %q", got)
	}
	if got := (Sparkline{Values: []float64{0, 0, 10, 10}, Width: 2}).Line(); got != "▁█" {
		t.Fatalf("downsampled sparkline = %q, want ▁█", got)
	}
	built := Sparkline{Values: []float64{0, 7}, Style: "accent"}.Build().Build()
	if boxText(built) != "▁█" || built.GetStyle() != "accent" {
		t.Fatalf("sparkline build = %+v", built)
	}
}

func TestBarChartVerticalDimensions(t *testing.T) {
	chart := BarChart{Values: []float64{1, 2, 3, 4}, Width: 7, Style: "bar", LabelStyle: "lbl", Labels: []string{"a", "b", "c", "d"}}
	if got := chart.MaxValue(); got != 4 {
		t.Fatalf("auto max = %v, want 4", got)
	}
	lines := chart.VerticalLines()
	if len(lines) != 2 {
		t.Fatalf("vertical lines = %d, want bar + label row", len(lines))
	}
	if lines[0] != "▂ ▄ ▆ █" {
		t.Fatalf("vertical bar row = %q, want ▂ ▄ ▆ █", lines[0])
	}
	if lines[1] != "a b c d" {
		t.Fatalf("label row = %q", lines[1])
	}
	built := chart.Build().Build()
	children := built.GetChildren()
	if len(children) != 2 {
		t.Fatalf("vertical children = %d, want bar + label", len(children))
	}
	if boxText(children[0].GetChildren()[0]) != "▂" || children[0].GetChildren()[0].GetStyle() != "bar" {
		t.Fatalf("first bar cell = %+v", children[0].GetChildren()[0])
	}
	if boxText(children[1].GetChildren()[0]) != "a" || children[1].GetChildren()[0].GetStyle() != "lbl" {
		t.Fatalf("first label cell = %+v", children[1].GetChildren()[0])
	}
}

func TestBarChartVerticalPartialAndHeight(t *testing.T) {
	if got := (BarChart{Values: []float64{0.5}, Max: 1, Width: 1}).VerticalLines()[0]; got != "▄" {
		t.Fatalf("half-height bar = %q, want ▄", got)
	}
	lines := BarChart{Values: []float64{4}, Max: 4, Width: 1, Height: 2}.VerticalLines()
	if len(lines) != 2 || lines[0] != "█" || lines[1] != "█" {
		t.Fatalf("two-row full bar = %v", lines)
	}
	lines = BarChart{Values: []float64{2}, Max: 4, Width: 1, Height: 2}.VerticalLines()
	if len(lines) != 2 || lines[0] != " " || lines[1] != "█" {
		t.Fatalf("two-row half bar = %v, want bottom full/top blank", lines)
	}
	if got := (BarChart{Values: []float64{1, 2, 3, 4}, Width: 3}).VerticalLines(); len(got[0]) == 0 {
		t.Fatalf("narrow vertical chart = %v", got)
	}
	if got := (BarChart{}).VerticalLines(); got != nil {
		t.Fatalf("empty vertical chart = %v, want nil", got)
	}
}

func TestBarChartHorizontal(t *testing.T) {
	lines := BarChart{Values: []float64{50, 100}, Width: 10, Horizontal: true}.HorizontalLines()
	if len(lines) != 2 || lines[0] != "█████" || lines[1] != "██████████" {
		t.Fatalf("horizontal lines = %v", lines)
	}
	lines = BarChart{Values: []float64{50, 100}, Width: 10, Horizontal: true, Labels: []string{"a", "bb"}, LabelStyle: "lbl"}.HorizontalLines()
	if lines[0] != "a  ████" {
		t.Fatalf("labelled horizontal row 0 = %q", lines[0])
	}
	if lines[1] != "bb ███████" {
		t.Fatalf("labelled horizontal row 1 = %q", lines[1])
	}
	if got := (BarChart{Values: []float64{1, 2, 3}, Height: 2, Horizontal: true}).HorizontalLines(); len(got) != 2 {
		t.Fatalf("height-capped horizontal rows = %d, want 2", len(got))
	}
	if got := (BarChart{Values: []float64{-1, 0}, Width: 4, Horizontal: true}).HorizontalLines(); strings.TrimSpace(got[0]) != "" || strings.TrimSpace(got[1]) != "" {
		t.Fatalf("non-positive horizontal runs = %v, want blank", got)
	}
}

func TestBarChartSelectedStyle(t *testing.T) {
	chart := BarChart{Values: []float64{1, 1}, Width: 3, Style: "base", Selected: 1, SelectedStyle: "hot"}
	children := chart.Build().Build().GetChildren()[0].GetChildren()
	if children[0].GetStyle() != "base" {
		t.Fatalf("unselected style = %q", children[0].GetStyle())
	}
	if children[2].GetStyle() != "hot" {
		t.Fatalf("selected style = %q", children[2].GetStyle())
	}
}

func TestHeatmapShadeRampAndRagged(t *testing.T) {
	heat := Heatmap{Values: [][]float64{{0, 1}, {2, 3}}}
	if heat.Shades != nil {
		t.Fatal("default shades must be nil for the default ramp")
	}
	grid := heat.Grid()
	if len(grid) != 2 || grid[0] != " ░" || grid[1] != "▒█" {
		t.Fatalf("heatmap grid = %v", grid)
	}
	if got := heat.Level(-100); got != 0 {
		t.Fatalf("low clamp level = %d, want 0", got)
	}
	if got := heat.Level(100); got != 5 {
		t.Fatalf("high clamp level = %d, want 5", got)
	}
	if got := (Heatmap{Values: [][]float64{{5, 5}}}).Level(5); got != 3 {
		t.Fatalf("all-equal level = %d, want mid ramp 3", got)
	}
	ragged := Heatmap{Values: [][]float64{{1, 2, 3}, {4}}, ColLabels: []string{"x", "y", "z"}, RowLabels: []string{"r1", "r2"}}
	if ragged.Cols() != 3 || ragged.Rows() != 2 {
		t.Fatalf("ragged dims = %dx%d, want 2x3", ragged.Rows(), ragged.Cols())
	}
	if got := ragged.Cell(1, 1); got != " " {
		t.Fatalf("missing cell = %q, want lowest shade", got)
	}
	grid = ragged.Grid()
	if len(grid) != 3 {
		t.Fatalf("ragged grid rows = %d, want header + 2", len(grid))
	}
	if !strings.HasPrefix(grid[1], "r1 ") {
		t.Fatalf("row label missing: %q", grid[1])
	}
	if got := (Heatmap{Values: [][]float64{{1}, {2}}, Shades: []string{"a", "b", "c"}}).Grid(); len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("custom shade grid = %v, want [a c]", got)
	}
	if got := (Heatmap{}).Grid(); got != nil {
		t.Fatalf("empty heatmap grid = %v, want nil", got)
	}
	built := Heatmap{Values: [][]float64{{1}}, Style: "heat", LabelStyle: "lbl", ColLabels: []string{"c"}}.Build().Build()
	if children := built.GetChildren(); len(children) != 2 || children[0].GetStyle() != "lbl" || children[1].GetStyle() != "heat" {
		t.Fatalf("heatmap build = %+v", built)
	}
}

func TestMeterAndGauge(t *testing.T) {
	meter := Meter{Value: 3, Max: 10, Width: 20, Style: "fill", TrackStyle: "track", ShowValue: true, Label: "cpu "}
	if got := meter.Fraction(); got != 0.3 {
		t.Fatalf("meter fraction = %v", got)
	}
	if strings.Count(meter.BarText(), "█") != 6 || len([]rune(meter.BarText())) != 20 {
		t.Fatalf("meter bar = %q", meter.BarText())
	}
	if got := meter.ValueText(); got != "3/10" {
		t.Fatalf("meter value = %q", got)
	}
	children := meter.Build().Build().GetChildren()
	if len(children) != 4 {
		t.Fatalf("meter children = %d, want label/fill/track/value", len(children))
	}
	if got := (Gauge{Value: 1, Max: 0}).Fraction(); got != 0 {
		t.Fatalf("gauge non-positive max = %v", got)
	}
	if got := (Meter{Value: 20, Max: 10}).Fraction(); got != 1 {
		t.Fatalf("meter clamps = %v", got)
	}
	if got := (Meter{Value: 1, Max: 2}).ValueText(); got != "" {
		t.Fatalf("meter value hidden by default = %q", got)
	}
}

func TestLegend(t *testing.T) {
	legend := Legend{Style: "muted", Items: []LegendItem{
		{Label: "go", Color: "accent", Marker: "x"},
		{Label: "ui"},
	}}
	text := legend.Text()
	if text != "x go ■ ui" {
		t.Fatalf("legend text = %q", text)
	}
	children := legend.Build().Build().GetChildren()
	if len(children) != 5 {
		t.Fatalf("legend children = %d, want marker/label/sep/marker/label", len(children))
	}
	if boxText(children[0]) != "x" || children[0].GetStyle() != "accent" {
		t.Fatalf("legend marker 0 = %+v", children[0])
	}
	if boxText(children[1]) != " go" || children[1].GetStyle() != "muted" {
		t.Fatalf("legend label 0 = %+v", children[1])
	}
	if boxText(children[2]) != " " {
		t.Fatalf("legend separator = %+v", children[2])
	}
	if boxText(children[3]) != "■" || children[3].GetStyle() != "" {
		t.Fatalf("legend default marker = %+v", children[3])
	}
	if got := (Legend{Separator: " | ", Items: []LegendItem{{Label: "a"}}}).Text(); got != "■ a" {
		t.Fatalf("single legend = %q", got)
	}
	if got := (Legend{Items: []LegendItem{{Label: "a"}, {Label: "b"}}, Separator: "|"}).Text(); got != "■ a|■ b" {
		t.Fatalf("separated legend = %q", got)
	}
}

func TestChartPlainTextIsAnsiFree(t *testing.T) {
	texts := []string{
		Sparkline{Values: []float64{1, 2, 3}}.Line(),
		strings.Join(BarChart{Values: []float64{1, 2}, Width: 3, Horizontal: true}.HorizontalLines(), "\n"),
		strings.Join(Heatmap{Values: [][]float64{{1, 2}}}.Grid(), "\n"),
		Legend{Items: []LegendItem{{Label: "x"}}}.Text(),
	}
	for _, text := range texts {
		if strings.Contains(text, "\x1b") {
			t.Fatalf("chart text carries ANSI: %q", text)
		}
		if sdk.DisplayWidth(text) < 0 {
			t.Fatalf("negative display width: %q", text)
		}
	}
}
