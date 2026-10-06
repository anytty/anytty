package widgets

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
)

func TestProgressBarFractionAndBar(t *testing.T) {
	bar := ProgressBar{Value: 50, Max: 200, Width: 10, Label: "load ", Style: "fill", TrackStyle: "track", PercentStyle: "pct"}
	if got := bar.Fraction(); got != 0.25 {
		t.Fatalf("Fraction = %v, want 0.25", got)
	}
	if got := bar.Percent(); got != 25 {
		t.Fatalf("Percent = %d, want 25", got)
	}
	if text := bar.BarText(); len([]rune(text)) != 10 || strings.Count(text, "█") != 3 {
		t.Fatalf("BarText = %q, want 3 filled / 10", text)
	}
	children := bar.Build().Build().GetChildren()
	if len(children) != 4 {
		t.Fatalf("progress children = %d, want label/fill/track/percent", len(children))
	}
	if boxText(children[0]) != "load " || children[0].GetStyle() != "" {
		t.Fatalf("label = %+v", children[0])
	}
	if boxText(children[1]) != "███" || children[1].GetStyle() != "fill" {
		t.Fatalf("fill = %+v", children[1])
	}
	if boxText(children[2]) != "░░░░░░░" || children[2].GetStyle() != "track" {
		t.Fatalf("track = %+v", children[2])
	}
	if boxText(children[3]) != "25%" || children[3].GetStyle() != "pct" {
		t.Fatalf("percent = %+v", children[3])
	}
	if barWidth := len([]rune(ProgressBar{}.BarText())); barWidth != DefaultProgressWidth {
		t.Fatalf("default bar width = %d", barWidth)
	}
}

func TestProgressBarClampsAndRounds(t *testing.T) {
	cases := []struct {
		value, max, percent int
	}{
		{-5, 10, 0},
		{30, 10, 100},
		{1, 3, 33},
		{2, 3, 67},
		{0, 0, 0},
		{5, -1, 0},
	}
	for _, tc := range cases {
		bar := ProgressBar{Value: tc.value, Max: tc.max}
		if got := bar.Percent(); got != tc.percent {
			t.Errorf("Percent(%d/%d) = %d, want %d", tc.value, tc.max, got, tc.percent)
		}
	}
	if got := (ProgressBar{Value: 3, Max: 4, HidePercent: true}).Build().Build().GetChildren(); len(got) != 2 {
		t.Fatalf("HidePercent children = %d, want fill + track", len(got))
	}
}

func TestSpinnerFrameAndLabel(t *testing.T) {
	spinner := Spinner{Frame: 11, Frames: []string{"a", "b", "c"}, Style: "spin", Label: "working", LabelStyle: "muted"}
	if got := spinner.FrameText(); got != "c" {
		t.Fatalf("FrameText = %q, want c", got)
	}
	if got := (Spinner{Frame: -1, Frames: []string{"a", "b", "c"}}).FrameText(); got != "c" {
		t.Fatalf("negative frame = %q, want c", got)
	}
	if got := (Spinner{}).FrameText(); got == "" {
		t.Fatal("default spinner frames must not be empty")
	}
	children := spinner.Build().Build().GetChildren()
	if len(children) != 2 || boxText(children[0]) != "c" || children[0].GetStyle() != "spin" {
		t.Fatalf("spinner row = %+v", children)
	}
	if boxText(children[1]) != "working" || children[1].GetStyle() != "muted" {
		t.Fatalf("spinner label = %+v", children[1])
	}
}

func TestBadgeAndTags(t *testing.T) {
	badge := Badge{ID: "badge:1", Text: "3", Style: "accent", Input: []string{"mouse"}}.Build().Build()
	if badge.GetId() != "badge:1" || boxText(badge) != "3" || badge.GetStyle() != "accent" {
		t.Fatalf("badge = %+v", badge)
	}
	if len(badge.GetInput()) != 1 || badge.GetInput()[0] != "mouse" {
		t.Fatalf("badge input = %v", badge.GetInput())
	}

	tags := Tags{ID: "tags", Style: "base", SepStyle: "sep", Items: []Tag{
		{Text: "go", ID: "tag:go"},
		{Text: "ui", Style: "hot"},
		{Text: "cjk 中文"},
	}}.Build().Build()
	children := tags.GetChildren()
	if len(children) != 5 {
		t.Fatalf("tags children = %d, want tag/sep/tag/sep/tag", len(children))
	}
	if boxText(children[0]) != "go" || children[0].GetStyle() != "base" || children[0].GetId() != "tag:go" {
		t.Fatalf("tag 0 = %+v", children[0])
	}
	if children[2].GetStyle() != "hot" {
		t.Fatalf("tag 1 style = %q", children[2].GetStyle())
	}
	if got := sdk.DisplayWidth(rowText(tags)); got != 14 {
		t.Fatalf("tags width = %d", got)
	}
}
