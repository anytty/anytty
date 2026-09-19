package runtime

import (
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/components/terminal"
	"github.com/anytty/anytty/clients/tui/kernel"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

func attachTerminal(t *testing.T, h *TerminalHandler, id string, argv ...string) *Terminal {
	t.Helper()
	outcome, pending := h.Handle(Request{
		Method: Method{Name: "terminal.attach"},
		Params: &pb.MethodParams{Endpoint: "local", Id: id, Argv: argv},
	})
	if !outcome.OK || pending {
		t.Fatalf("attach %s = %+v pending=%v", id, outcome, pending)
	}
	term, ok := h.Terminal(id)
	if !ok {
		t.Fatalf("terminal %s not registered", id)
	}
	return term
}

func screenContains(term *Terminal, want string) bool {
	for _, line := range term.Screen().TextLines() {
		if strings.Contains(strings.TrimRight(line, " "), want) {
			return true
		}
	}
	return false
}

func TestTerminalAttachParsesPrintfScreen(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Cols: 24, Rows: 4})
	defer h.Close()

	term := attachTerminal(t, h, "t1", "printf", "hello\nworld")
	waitFor(t, "printf output", func() bool {
		return screenContains(term, "hello") && screenContains(term, "world")
	})
	lines := term.Screen().TextLines()
	if len(lines) != 4 {
		t.Fatalf("screen rows = %d, want 4", len(lines))
	}
	if got := strings.TrimRight(lines[0], " "); got != "hello" {
		t.Fatalf("row 0 = %q, want hello", got)
	}
	if got := strings.TrimRight(lines[1], " "); got != "world" {
		t.Fatalf("row 1 = %q, want world", got)
	}
}

func TestTerminalAttachIsIdempotent(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Command: []string{"cat"}})
	defer h.Close()

	first := attachTerminal(t, h, "same")
	second := attachTerminal(t, h, "same", "printf", "ignored")
	if first != second {
		t.Fatal("re-attach must return the existing terminal")
	}
}

func TestTerminalResizeChangesScreenSize(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Cols: 20, Rows: 4, Command: []string{"cat"}})
	defer h.Close()

	term := attachTerminal(t, h, "t2")
	if err := h.Resize("t2", 10, 2); err != nil {
		t.Fatalf("Resize: %v", err)
	}
	screen := term.Screen()
	if len(screen.Lines) != 2 {
		t.Fatalf("rows = %d, want 2", len(screen.Lines))
	}
	for y, line := range screen.Lines {
		width := 0
		for _, cell := range line {
			width += cell.Width
		}
		if width != 10 {
			t.Fatalf("row %d width = %d, want 10", y, width)
		}
	}
	if cols, rows, err := term.Size(); err != nil || cols != 10 || rows != 2 {
		t.Fatalf("PTY size = %d×%d err=%v, want 10×2", cols, rows, err)
	}

	if _, err := term.Write([]byte("hi\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	waitFor(t, "echo after resize", func() bool { return screenContains(term, "hi") })
}

func TestTerminalCopyVisibleText(t *testing.T) {
	var copied string
	h := NewTerminalHandler(TerminalOptions{
		Cols:      20,
		Rows:      4,
		Clipboard: func(text string) error { copied = text; return nil },
	})
	defer h.Close()

	term := attachTerminal(t, h, "t3", "sh", "-c", "printf 'copy me\\n'; sleep 5")
	waitFor(t, "copy target output", func() bool { return screenContains(term, "copy me") })

	outcome, pending := h.Handle(Request{
		Method: Method{Name: "terminal.copy"},
		Params: &pb.MethodParams{Endpoint: "local", Id: "t3"},
	})
	if !outcome.OK || pending {
		t.Fatalf("copy = %+v pending=%v", outcome, pending)
	}
	if copied != "copy me" {
		t.Fatalf("clipboard = %q, want %q", copied, "copy me")
	}
}

func TestTerminalScrollServesScrollbackWindow(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{
		Cols:    12,
		Rows:    2,
		Command: []string{"sh", "-c", "printf 'one\\ntwo\\nthree\\n'; sleep 5"},
	})
	defer h.Close()

	term := attachTerminal(t, h, "t4")
	waitFor(t, "three lines of output", func() bool { return screenContains(term, "three") })

	outcome, pending := h.Handle(Request{
		Method: Method{Name: "terminal.scroll"},
		Params: &pb.MethodParams{Endpoint: "local", Id: "t4", Delta: 1, Rows: 2},
	})
	if !outcome.OK || pending {
		t.Fatalf("scroll = %+v pending=%v", outcome, pending)
	}
	rows := outcome.Data.GetRows()
	if len(rows) != 2 || rows[0] != "two" || rows[1] != "three" {
		t.Fatalf("scroll window = %q, want [two three]", rows)
	}
	if term.Offset() != 1 {
		t.Fatalf("offset = %d, want 1", term.Offset())
	}

	h.Handle(Request{
		Method: Method{Name: "terminal.scrollEnd"},
		Params: &pb.MethodParams{Endpoint: "local", Id: "t4"},
	})
	if term.Offset() != 0 {
		t.Fatalf("offset after scrollEnd = %d, want 0", term.Offset())
	}
}

func TestTerminalOwnerCASStub(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Command: []string{"cat"}, OwnerID: "view:a"})
	defer h.Close()

	term := attachTerminal(t, h, "t5")
	if owner, epoch := term.Owner(); owner != "view:a" || epoch != 1 {
		t.Fatalf("owner = %q epoch=%d, want view:a/1", owner, epoch)
	}
	if _, err := term.ClaimOwner("view:b", 0); err != ErrOwnerConflict {
		t.Fatalf("stale CAS = %v, want ErrOwnerConflict", err)
	}
	epoch, err := term.ClaimOwner("view:b", 1)
	if err != nil || epoch != 2 {
		t.Fatalf("CAS take-over = epoch %d err %v, want 2/nil", epoch, err)
	}
}

func TestTerminalRemoveRejectsRunningProcess(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Command: []string{"cat"}})
	defer h.Close()

	attachTerminal(t, h, "t6")
	outcome, _ := h.Handle(Request{
		Method: Method{Name: "terminal.remove"},
		Params: &pb.MethodParams{Endpoint: "local", Id: "t6"},
	})
	if outcome.OK || outcome.Error != ErrStillRunning.Error() {
		t.Fatalf("remove running = %+v, want still-running error", outcome)
	}
	if err := h.terminal("local", "t6").Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	waitFor(t, "process exit", func() bool { return h.terminal("local", "t6").Exited() })
	outcome, _ = h.Handle(Request{
		Method: Method{Name: "terminal.remove"},
		Params: &pb.MethodParams{Endpoint: "local", Id: "t6"},
	})
	if !outcome.OK {
		t.Fatalf("remove exited = %+v, want ok", outcome)
	}
	if _, ok := h.Terminal("t6"); ok {
		t.Fatal("removed terminal is still registered")
	}
}

func TestTerminalPlacementCarriesCursor(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Cols: 8, Rows: 3, Command: []string{"cat"}})
	defer h.Close()

	term := attachTerminal(t, h, "t7")
	if _, err := term.Write([]byte("ab")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	waitFor(t, "echoed cursor", func() bool {
		x, _, _ := term.Cursor()
		return x == 2
	})

	component := terminal.New(nil, nil)
	component.SetProps(terminal.Props{Title: "t"})
	component.SetScreen(term.Screen())
	rect := kernel.Rect{X: 5, Y: 2, Width: 10, Height: 4}
	placement := term.Placement(component, rect, true)
	if !placement.CursorVisible || placement.CursorX != 3 || placement.CursorY != 1 {
		t.Fatalf("placement cursor = (%d,%d,%v), want (3,1,true)", placement.CursorX, placement.CursorY, placement.CursorVisible)
	}
	unfocused := term.Placement(component, rect, false)
	if unfocused.CursorVisible {
		t.Fatal("unfocused placement must not carry the terminal cursor")
	}
}

func TestTerminalPlacementTinyRectDropsChrome(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Cols: 8, Rows: 3, Command: []string{"cat"}})
	defer h.Close()

	term := attachTerminal(t, h, "t8")
	if _, err := term.Write([]byte("a")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	waitFor(t, "echoed cursor", func() bool {
		x, _, _ := term.Cursor()
		return x == 1
	})

	component := terminal.New(nil, nil)
	component.SetScreen(term.Screen())
	rect := kernel.Rect{Width: 2, Height: 2}
	placement := term.Placement(component, rect, true)
	if !placement.CursorVisible || placement.CursorX != 1 || placement.CursorY != 0 {
		t.Fatalf("tiny placement cursor = (%d,%d,%v), want (1,0,true) with inset 0",
			placement.CursorX, placement.CursorY, placement.CursorVisible)
	}
}

func TestTerminalHandlerFallsBackToStub(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{})
	defer h.Close()

	outcome, pending := h.Handle(Request{
		Method: Method{Name: "terminal.create"},
		Params: &pb.MethodParams{Endpoint: "local"},
	})
	if !outcome.OK || pending || outcome.Data.GetId() == "" {
		t.Fatalf("terminal.create = %+v pending=%v, want stub create", outcome, pending)
	}
}

// TestTerminalPlacementCarriesChromeProps pins the runtime half of the M2
// pass-through: the placement exposes exactly the program-declared props the
// component was given.
func TestTerminalPlacementCarriesChromeProps(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Cols: 8, Rows: 3, Command: []string{"cat"}})
	defer h.Close()

	term := attachTerminal(t, h, "t9")
	component := terminal.New(nil, nil)
	component.SetProps(terminal.Props{Chrome: map[string]string{terminal.PropBorder: "fg:#565f89"}})
	placement := term.Placement(component, kernel.Rect{Width: 10, Height: 4}, false)
	if got := placement.Props[terminal.PropBorder]; got != "fg:#565f89" {
		t.Fatalf("placement props = %v, want the program chrome.border", placement.Props)
	}
}
