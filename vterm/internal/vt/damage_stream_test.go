package vt

import "testing"

func TestEmulatorWriteWithDamageCoalescesPrintableRun(t *testing.T) {
	emu := NewEmulator(8, 2)

	_, err, damages := emu.WriteWithDamage([]byte("abc"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	span, ok := firstDamageOfType[SpanDamage](damages)
	if !ok {
		t.Fatalf("expected span damage, got %#v", damages)
	}
	if span.X != 0 || span.Y != 0 || len(span.Cells) != 3 {
		t.Fatalf("unexpected span damage: %#v", span)
	}
	if span.Cells[0].Content != "a" || span.Cells[1].Content != "b" || span.Cells[2].Content != "c" {
		t.Fatalf("unexpected span cells: %#v", span.Cells)
	}
	text, ok := firstDamageOfType[TextDamage](damages)
	if !ok {
		t.Fatalf("expected text semantic damage, got %#v", damages)
	}
	if text.X != 0 || text.Y != 0 || text.Text != "abc" || !text.ASCII || len(text.Cells) != 0 || len(text.Runs) != 0 {
		t.Fatalf("unexpected text semantic damage: %#v", text)
	}
}

func TestEmulatorFastSGRTextTreatsExactWidthNewlineAsHardBreak(t *testing.T) {
	emu := NewEmulator(4, 3)
	if _, err := emu.Write([]byte("ABCD\r\nWXYZ")); err != nil {
		t.Fatalf("write fast text: %v", err)
	}
	if emu.ScreenLineWrapped(0) {
		t.Fatalf("exact-width CRLF row must not be marked as wrapped")
	}

	emu.Resize(8, 3)
	if got := termText(emu)[0]; got != "ABCD    " {
		t.Fatalf("expected first hard-break row preserved after resize, got %q", got)
	}
	if got := termText(emu)[1]; got != "WXYZ    " {
		t.Fatalf("expected second hard-break row preserved after resize, got %q", got)
	}
	if emu.ScreenLineWrapped(0) {
		t.Fatalf("exact-width CRLF row must remain unwrapped after resize")
	}
}

func TestEmulatorWriteWithDamageCapturesScrollUp(t *testing.T) {
	emu := NewEmulator(4, 3)
	if _, err := emu.Write([]byte("1111\r\n2222\r\n3333")); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	_, err, damages := emu.WriteWithDamage([]byte("\n"))
	if err != nil {
		t.Fatalf("scroll write: %v", err)
	}
	if len(damages) < 2 {
		t.Fatalf("expected scroll damage plus trailing clear, got %#v", damages)
	}
	scroll, ok := damages[0].(ScrollDamage)
	if !ok {
		t.Fatalf("expected first damage to be scroll, got %#v", damages[0])
	}
	if scroll.Dx != 0 || scroll.Dy != -1 || scroll.Min.X != 0 || scroll.Min.Y != 0 || scroll.Rectangle.Dx() != 4 || scroll.Rectangle.Dy() != 3 {
		t.Fatalf("unexpected scroll damage: %#v", scroll)
	}
	if _, ok := damages[1].(ClearDamage); !ok {
		t.Fatalf("expected trailing clear damage after scroll, got %#v", damages[1])
	}
}

func TestEmulatorWriteWithDamageUsesSpanForStyledErase(t *testing.T) {
	emu := NewEmulator(4, 1)
	if _, err := emu.Write([]byte("\x1b[48;2;1;2;3mAB")); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	_, err, damages := emu.WriteWithDamage([]byte("\x1b[1;1H\x1b[X"))
	if err != nil {
		t.Fatalf("erase write: %v", err)
	}
	if len(damages) == 0 {
		t.Fatal("expected erase damage")
	}
	control, ok := firstControlDamageKind(damages, "ech")
	if !ok || control.Mode != 1 {
		t.Fatalf("expected styled erase to record ECH control damage, got %#v", damages)
	}
	span, ok := firstDamageOfType[SpanDamage](damages)
	if !ok {
		t.Fatalf("expected styled erase to produce span damage, got %#v", damages)
	}
	if len(span.Cells) != 1 || span.Cells[0].Content != " " || span.Cells[0].Style.Bg == nil {
		t.Fatalf("unexpected styled erase span: %#v", span)
	}
}

func firstDamageOfType[T Damage](damages []Damage) (T, bool) {
	var zero T
	for _, damage := range damages {
		if typed, ok := damage.(T); ok {
			return typed, true
		}
	}
	return zero, false
}

func firstControlDamageKind(damages []Damage, kind string) (ControlDamage, bool) {
	for _, damage := range damages {
		control, ok := damage.(ControlDamage)
		if ok && control.Kind == kind {
			return control, true
		}
	}
	return ControlDamage{}, false
}
