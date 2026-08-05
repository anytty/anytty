package vterm

import (
	charmvt "github.com/anytty/anytty/vterm/internal/vt"
	"strings"
	"sync"
	"testing"
)

func TestVTermBasicBehavior(t *testing.T) {
	vt := New(5, 2, 2, nil)

	if _, err := vt.Write([]byte("\x1b[31mA\x1b[0m")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	cell := vt.CellAt(0, 0)
	if cell.Content != "A" {
		t.Fatalf("unexpected content: %#v", cell)
	}
	if cell.Style.FG == "" {
		t.Fatal("expected foreground color")
	}

	if _, err := vt.Write([]byte("1\n2\n3\n4\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	scrollback := vt.ScrollbackContent()
	if len(scrollback) == 0 {
		t.Fatal("expected scrollback")
	}
	found := false
	for _, row := range scrollback {
		if strings.TrimSpace(rowToString(row)) != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected scrollback to contain content")
	}

	if _, err := vt.Write([]byte("\x1b[?1049h")); err != nil {
		t.Fatalf("alt screen write failed: %v", err)
	}
	if !vt.IsAltScreen() {
		t.Fatal("expected alt screen")
	}
}

func TestVTermConcurrentAccess(t *testing.T) {
	vt := New(80, 24, 10, nil)
	var wg sync.WaitGroup

	for i := 0; i < 32; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = vt.Write([]byte("hello"))
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = vt.CellAt(0, 0)
			}
		}()
	}

	wg.Wait()
}

func TestVTermTracksApplicationCursorMode(t *testing.T) {
	vt := New(80, 24, 10, nil)

	if _, err := vt.Write([]byte("\x1b[?1h")); err != nil {
		t.Fatalf("enable application cursor failed: %v", err)
	}
	if !vt.Modes().ApplicationCursor {
		t.Fatal("expected application cursor mode to be enabled")
	}

	if _, err := vt.Write([]byte("\x1b[?1l")); err != nil {
		t.Fatalf("disable application cursor failed: %v", err)
	}
	if vt.Modes().ApplicationCursor {
		t.Fatal("expected application cursor mode to be disabled")
	}
}

func TestVTermPreservesAnsiIndexedColorSemantic(t *testing.T) {
	vt := New(10, 3, 10, nil)
	vt.SetIndexedColor(1, "#123456")

	if _, err := vt.Write([]byte("\x1b[31mR\x1b[0m")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	cell := vt.CellAt(0, 0)
	if cell.Style.FG != "ansi:1" {
		t.Fatalf("expected ANSI semantic color token, got %#v", cell.Style)
	}
}

func TestVTermNormalizesPlainUTF8CombiningText(t *testing.T) {
	vt := New(20, 5, 10, nil)

	if _, err := vt.Write([]byte("e\u0301🙂한글")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	row := vt.ScreenContent().Cells[0]
	if got := rowToString(row); !strings.Contains(got, "é🙂한글") {
		t.Fatalf("expected normalized text in row, got %q", got)
	}
}

func TestVTermEmojiVariationSelectorFootprint(t *testing.T) {
	vt := New(12, 2, 0, nil)
	if _, err := vt.Write([]byte("♻️♻️♻️··")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	row := vt.ScreenContent().Cells[0]
	got := make([]struct {
		Content string
		Width   int
	}, 8)
	for i := range got {
		got[i].Content = row[i].Content
		got[i].Width = row[i].Width
	}
	want := []struct {
		Content string
		Width   int
	}{
		{Content: "♻️", Width: 2},
		{Content: "", Width: 0},
		{Content: "♻️", Width: 2},
		{Content: "", Width: 0},
		{Content: "♻️", Width: 2},
		{Content: "", Width: 0},
		{Content: "·", Width: 1},
		{Content: "·", Width: 1},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cell %d footprint mismatch: got %#v want %#v full=%#v", i, got[i], want[i], got)
		}
	}
}

func TestVTermWriteRecoversFromEmulatorPanic(t *testing.T) {
	vt := New(20, 5, 10, nil)

	prev := safeEmulatorWrite
	prevWithDamage := safeEmulatorWriteWithDamage
	safeEmulatorWrite = func(_ *charmvt.SafeEmulator, _ []byte) (int, error) {
		panic("boom")
	}
	safeEmulatorWriteWithDamage = func(_ *charmvt.SafeEmulator, _ []byte) (int, error, []charmvt.Damage, bool) {
		panic("boom")
	}
	t.Cleanup(func() {
		safeEmulatorWrite = prev
		safeEmulatorWriteWithDamage = prevWithDamage
	})

	if _, err := vt.Write([]byte("hello")); err == nil {
		t.Fatal("expected write to convert emulator panic into error")
	} else if !strings.Contains(err.Error(), "panic") {
		t.Fatalf("expected panic context in error, got %v", err)
	}
}

func TestVTermPreservesRowTimestampAcrossScroll(t *testing.T) {
	vt := New(4, 2, 10, nil)

	if _, err := vt.Write([]byte("abcd\r\nefgh")); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}

	firstRowTS := vt.ScreenRowTimestampAt(0)
	if firstRowTS.IsZero() {
		t.Fatal("expected first row timestamp to be set")
	}

	if _, err := vt.Write([]byte("\r\nijkl")); err != nil {
		t.Fatalf("scroll write failed: %v", err)
	}

	if got := strings.TrimSpace(rowToString(vt.ScrollbackRow(0))); got != "abcd" {
		t.Fatalf("expected first row to scroll into scrollback, got %q", got)
	}
	if got := vt.ScrollbackRowTimestampAt(0); !got.Equal(firstRowTS) {
		t.Fatalf("expected scrollback timestamp %v, got %v", firstRowTS, got)
	}
}

func TestVTermRowViewsReuseCacheWithoutExposingMutableRows(t *testing.T) {
	vt := New(4, 2, 10, nil)

	if _, err := vt.Write([]byte("abcd\r\nefgh\r\nijkl")); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}

	screenViewA := vt.ScreenRowView(1)
	screenViewB := vt.ScreenRowView(1)
	if len(screenViewA) == 0 || len(screenViewB) == 0 {
		t.Fatal("expected cached screen row view")
	}
	if &screenViewA[0] != &screenViewB[0] {
		t.Fatal("expected screen row view to reuse cached backing storage")
	}

	screenCopy := vt.ScreenRow(1)
	screenCopy[0].Content = "z"
	if got := strings.TrimSpace(rowToString(vt.ScreenRow(1))); got != "ijkl" {
		t.Fatalf("expected ScreenRow to return a copy, got %q", got)
	}
	if got := strings.TrimSpace(rowToString(vt.ScreenRowView(1))); got != "ijkl" {
		t.Fatalf("expected ScreenRowView to remain unchanged, got %q", got)
	}

	scrollViewA := vt.ScrollbackRowView(0)
	scrollViewB := vt.ScrollbackRowView(0)
	if len(scrollViewA) == 0 || len(scrollViewB) == 0 {
		t.Fatal("expected cached scrollback row view")
	}
	if &scrollViewA[0] != &scrollViewB[0] {
		t.Fatal("expected scrollback row view to reuse cached backing storage")
	}

	scrollCopy := vt.ScrollbackRow(0)
	scrollCopy[0].Content = "z"
	if got := strings.TrimSpace(rowToString(vt.ScrollbackRow(0))); got != "abcd" {
		t.Fatalf("expected ScrollbackRow to return a copy, got %q", got)
	}
	if got := strings.TrimSpace(rowToString(vt.ScrollbackRowView(0))); got != "abcd" {
		t.Fatalf("expected ScrollbackRowView to remain unchanged, got %q", got)
	}
}

func rowToString(row []Cell) string {
	var b strings.Builder
	for _, cell := range row {
		b.WriteString(cell.Content)
	}
	return b.String()
}
