package widgets

import "testing"

func TestScrollbarThumbProportional(t *testing.T) {
	bar := Scrollbar{Total: 100, Visible: 10, Height: 10}
	start, size := bar.Thumb()
	if start != 0 || size != 1 {
		t.Fatalf("top thumb = (%d, %d), want (0, 1)", start, size)
	}
	bar.Offset = 90
	start, size = bar.Thumb()
	if start != 9 || size != 1 {
		t.Fatalf("bottom thumb = (%d, %d), want (9, 1)", start, size)
	}
	bar.Offset = 45
	start, _ = bar.Thumb()
	if start != 4 {
		t.Fatalf("mid thumb start = %d, want 4", start)
	}

	half := Scrollbar{Total: 100, Visible: 50, Height: 10}
	start, size = half.Thumb()
	if start != 0 || size != 5 {
		t.Fatalf("half thumb = (%d, %d), want (0, 5)", start, size)
	}
}

func TestScrollbarThumbDegenerate(t *testing.T) {
	fits := Scrollbar{Total: 5, Visible: 10, Height: 10}
	if start, size := fits.Thumb(); start != 0 || size != 10 {
		t.Fatalf("total<=visible thumb = (%d, %d), want full track", start, size)
	}
	if offset := fits.OffsetAt(7); offset != 0 {
		t.Fatalf("total<=visible OffsetAt = %d, want 0", offset)
	}

	empty := Scrollbar{Total: 0, Visible: 0, Height: 4}
	if start, size := empty.Thumb(); start != 0 || size != 4 {
		t.Fatalf("empty thumb = (%d, %d), want full track", start, size)
	}

	zeroVisible := Scrollbar{Total: 10, Visible: 0, Height: 10, Offset: 5}
	if _, size := zeroVisible.Thumb(); size != 1 {
		t.Fatalf("visible=0 thumb size = %d, want the 1-cell minimum", size)
	}

	noTrack := Scrollbar{Total: 10, Visible: 3}
	if start, size := noTrack.Thumb(); start != 0 || size != 0 {
		t.Fatalf("no-length thumb = (%d, %d), want (0, 0)", start, size)
	}
}

func TestScrollbarClampsOffset(t *testing.T) {
	bar := Scrollbar{Total: 100, Visible: 10, Height: 10, Offset: 1000}
	if start, _ := bar.Thumb(); start != 9 {
		t.Fatalf("overflow offset start = %d, want the bottom cell", start)
	}
	bar.Offset = -50
	if start, _ := bar.Thumb(); start != 0 {
		t.Fatalf("negative offset start = %d, want 0", start)
	}
}

func TestScrollbarOffsetAt(t *testing.T) {
	bar := Scrollbar{Total: 100, Visible: 10, Height: 10}
	if got := bar.OffsetAt(0); got != 0 {
		t.Fatalf("OffsetAt(0) = %d", got)
	}
	if got := bar.OffsetAt(9); got != 90 {
		t.Fatalf("OffsetAt(9) = %d, want 90", got)
	}
	if got := bar.OffsetAt(100); got != 90 {
		t.Fatalf("OffsetAt(overflow) = %d, want the clamped 90", got)
	}

	wide := Scrollbar{Total: 40, Visible: 10, Height: 10}
	start, size := wide.Thumb()
	if start != 0 || size != 2 {
		t.Fatalf("wide thumb = (%d, %d), want (0, 2)", start, size)
	}
	mid := wide.OffsetAt(5)
	if mid <= 0 || mid >= 30 {
		t.Fatalf("OffsetAt(5) = %d, want an interior offset", mid)
	}
}

func TestScrollbarHorizontalBuild(t *testing.T) {
	bar := Scrollbar{ID: "sb", Total: 20, Visible: 5, Width: 10}
	if bar.IsVertical() {
		t.Fatal("Width-only scrollbar must be horizontal")
	}
	if bar.TrackLength() != 10 {
		t.Fatalf("TrackLength = %d", bar.TrackLength())
	}
	row := bar.Build().Build()
	if len(row.GetChildren()) != 10 {
		t.Fatalf("horizontal track cells = %d, want 10", len(row.GetChildren()))
	}
	if text := boxText(row.GetChildren()[0]); text != ScrollbarThumbHorizontal {
		t.Fatalf("first horizontal cell = %q, want thumb", text)
	}
	start, size := bar.Thumb()
	if start != 0 || size != 2 {
		t.Fatalf("horizontal thumb = (%d, %d), want (0, 2)", start, size)
	}
}

func TestScrollbarVerticalBuildCaps(t *testing.T) {
	bar := Scrollbar{ID: "sb", Total: 10, Visible: 2, Height: 5, Up: "▲", Down: "▼"}
	children := bar.Build().Build().GetChildren()
	if len(children) != 7 {
		t.Fatalf("vertical children = %d, want 2 caps + 5 track", len(children))
	}
	if boxText(children[0]) != "▲" || boxText(children[6]) != "▼" {
		t.Fatalf("caps = %q/%q", boxText(children[0]), boxText(children[6]))
	}
	if start, size := bar.Thumb(); start != 0 || size != 1 {
		t.Fatalf("thumb = (%d, %d), want (0, 1)", start, size)
	}
}
