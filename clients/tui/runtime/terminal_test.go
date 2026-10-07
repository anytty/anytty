package runtime

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/components/terminal"
	"github.com/anytty/anytty/clients/tui/history"
	"github.com/anytty/anytty/clients/tui/kernel"
	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render/ansi"
	"github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

type bottomTraceBackend struct {
	mu       sync.Mutex
	windows  int
	released chan struct{}
}

func (b *bottomTraceBackend) Window(ctx context.Context, _ *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b.mu.Lock()
	b.windows++
	b.mu.Unlock()
	result := &apipb.HistoryWindowResult{Token: "bottom-trace"}
	for i := 0; i < 6; i++ {
		result.Rows = append(result.Rows, &apipb.HistoryRow{
			LogicalLineId: uint64(i + 1),
			Row:           &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: fmt.Sprintf("FROZEN-%d", i), Width: 1}}},
		})
	}
	return result, nil
}

func (b *bottomTraceBackend) Search(context.Context, *apipb.HistorySearchCommand) (*apipb.HistorySearchResult, error) {
	return &apipb.HistorySearchResult{}, nil
}

func (b *bottomTraceBackend) Copy(context.Context, *apipb.HistoryCopyCommand) (*apipb.HistoryCopyResult, error) {
	return &apipb.HistoryCopyResult{Done: true}, nil
}

func (b *bottomTraceBackend) Release(context.Context, *apipb.HistoryReleaseCommand) error {
	select {
	case <-b.released:
	default:
		close(b.released)
	}
	return nil
}

type bottomTracePTY struct {
	backend history.Backend
	cols    int
	rows    int
}

type snapshotResetPTY struct {
	*bottomTracePTY
	reset func()
}

func (p *snapshotResetPTY) SetSnapshotReset(reset func()) { p.reset = reset }

func (p *bottomTracePTY) HistoryBackend(context.Context) (history.Backend, *apipb.TerminalRef, error) {
	return p.backend, &apipb.TerminalRef{TerminalId: "bottom-trace"}, nil
}

func (p *bottomTracePTY) Start() error                   { return nil }
func (p *bottomTracePTY) Read([]byte) (int, error)       { return 0, io.EOF }
func (p *bottomTracePTY) Write(data []byte) (int, error) { return len(data), nil }
func (p *bottomTracePTY) Resize(cols, rows int) error    { p.cols, p.rows = cols, rows; return nil }
func (p *bottomTracePTY) Size() (int, int, error)        { return p.cols, p.rows, nil }
func (p *bottomTracePTY) ExitCode() int                  { return -1 }
func (p *bottomTracePTY) Close() error                   { return nil }

var _ pty.PTY = (*bottomTracePTY)(nil)
var _ history.Source = (*bottomTracePTY)(nil)

func TestSnapshotResetDropsPublishedHistoryViewport(t *testing.T) {
	backend := &bottomTraceBackend{released: make(chan struct{})}
	proc := &snapshotResetPTY{bottomTracePTY: &bottomTracePTY{backend: backend, cols: 20, rows: 2}}
	term := newTerminal("terminal:remote:history-resync", "history-resync", nil, proc, 20, 2, nil)
	if _, _, err := term.HistoryScroll(context.Background(), "", 1, 2); err != nil {
		t.Fatalf("enter history: %v", err)
	}
	if !term.HistoryActive("") || !strings.Contains(strings.Join(term.VisibleLines(""), "|"), "FROZEN") {
		t.Fatal("history viewport was not published")
	}
	if proc.reset == nil {
		t.Fatal("terminal did not install snapshot reset hook")
	}
	proc.reset()
	if term.HistoryActive("") {
		t.Fatal("reconnect left the old history viewport active")
	}
	select {
	case <-backend.released:
	case <-time.After(time.Second):
		t.Fatal("old history snapshot was not released after reconnect")
	}
}

// TestPersistentHistoryBottomTrace is a small old/new behavior detector. It
// records the same gesture used in the bug report: enter history, return to
// the live bottom, then keep scrolling down. Once the bottom is reached every
// observation must remain live and must not issue another provider window.
func TestPersistentHistoryBottomTrace(t *testing.T) {
	backend := &bottomTraceBackend{released: make(chan struct{})}
	proc := &bottomTracePTY{backend: backend, cols: 20, rows: 2}
	term := newTerminal("terminal:remote:bottom-trace", "bottom-trace", nil, proc, 20, 2, nil)
	term.parser.Write([]byte("\x1b[1;1HLIVE-A\x1b[2;1HLIVE-B"))

	type observation struct {
		delta  int
		offset int
		active bool
		lines  string
	}
	var trace []observation
	for _, delta := range []int{1, -1, -1, -1} {
		if _, offset, err := term.HistoryScroll(context.Background(), "", delta, 2); err != nil {
			t.Fatalf("delta=%d: %v", delta, err)
		} else {
			trace = append(trace, observation{
				delta: delta, offset: offset, active: term.HistoryActive(""),
				lines: strings.Join(term.VisibleLines(""), "|")})
		}
	}
	t.Logf("bottom trace: %+v", trace)
	if trace[0].offset != 1 || !trace[0].active || !strings.Contains(trace[0].lines, "FROZEN") {
		t.Fatalf("history entry trace = %+v", trace[0])
	}
	for i := 1; i < len(trace); i++ {
		if trace[i].offset != 0 || trace[i].active || trace[i].lines != "LIVE-A|LIVE-B" {
			t.Fatalf("live bottom trace[%d] = %+v", i, trace[i])
		}
	}
	backend.mu.Lock()
	windows := backend.windows
	backend.mu.Unlock()
	if windows != 1 {
		t.Fatalf("provider window calls after bottom = %d, want 1", windows)
	}
	select {
	case <-backend.released:
	case <-time.After(time.Second):
		t.Fatal("history snapshot was not released after returning to live")
	}
}

func TestPersistentHistoryPinnedBottomUnpinsInsteadOfReturningStaleRows(t *testing.T) {
	backend := &bottomTraceBackend{released: make(chan struct{})}
	proc := &bottomTracePTY{backend: backend, cols: 20, rows: 2}
	term := newTerminal("terminal:remote:pinned-bottom", "pinned-bottom", nil, proc, 20, 2, nil)
	term.parser.Write([]byte("old\r\nlive\r\n"))

	// Simulate a parser-backed frozen window left by a previous capability
	// transition. A downward wheel must release that local pin before the
	// persistent history path can claim the terminal again.
	term.mu.Lock()
	total := term.contentTotalLocked()
	v := term.viewLocked("")
	v.pinned = true
	v.viewEnd = total - 1
	term.mu.Unlock()

	lines, offset, err := term.HistoryScroll(context.Background(), "", -1, 2)
	if err != nil {
		t.Fatalf("pinned downward scroll: %v", err)
	}
	if offset != 0 || term.HistoryActive("") {
		t.Fatalf("after pinned bottom scroll offset=%d active=%v, want 0/false", offset, term.HistoryActive(""))
	}
	want, _ := term.Window("", 0, 2)
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("live rows after unpin = %q, want %q", strings.Join(lines, "|"), strings.Join(want, "|"))
	}
	backend.mu.Lock()
	windows := backend.windows
	backend.mu.Unlock()
	if windows != 0 {
		t.Fatalf("pinned bottom opened provider window calls=%d, want 0", windows)
	}
}

type staleGenerationBackend struct {
	onOlder func()
}

func (b *staleGenerationBackend) Window(ctx context.Context, req *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.GetMode() == apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER && b.onOlder != nil {
		b.onOlder()
	}
	result := &apipb.HistoryWindowResult{Token: "stale-trace"}
	for i := 0; i < 2; i++ {
		result.Rows = append(result.Rows, &apipb.HistoryRow{
			LogicalLineId: uint64(i + 1),
			Row:           &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: fmt.Sprintf("ROW-%d", i), Width: 1}}},
		})
	}
	if req.GetMode() == apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER {
		result.Rows = nil
		result.HasMore = false
	} else {
		result.HasMore = true
	}
	return result, nil
}

func (b *staleGenerationBackend) Search(context.Context, *apipb.HistorySearchCommand) (*apipb.HistorySearchResult, error) {
	return &apipb.HistorySearchResult{}, nil
}

func (b *staleGenerationBackend) Copy(context.Context, *apipb.HistoryCopyCommand) (*apipb.HistoryCopyResult, error) {
	return &apipb.HistoryCopyResult{Done: true}, nil
}

func (b *staleGenerationBackend) Release(context.Context, *apipb.HistoryReleaseCommand) error {
	return nil
}

func TestLatestScrollGenerationDropsLateProviderResult(t *testing.T) {
	backend := &staleGenerationBackend{}
	proc := &bottomTracePTY{backend: backend, cols: 20, rows: 2}
	term := newTerminal("terminal:remote:stale-trace", "stale-trace", nil, proc, 20, 2, nil)
	term.historyQueueMu.Lock()
	term.historyScrollGeneration = 1
	term.historyQueueMu.Unlock()
	backend.onOlder = func() {
		term.historyQueueMu.Lock()
		term.historyScrollGeneration = 2
		term.historyQueueMu.Unlock()
	}
	ctx := context.WithValue(context.Background(), historyScrollGenerationKey{}, uint64(1))
	if _, _, err := term.HistoryScroll(ctx, "", 1, 2); err != context.Canceled {
		t.Fatalf("late scroll error = %v, want context canceled", err)
	}
	if term.HistoryActive("") {
		t.Fatal("late provider result republished a stale history viewport")
	}
}

func TestEnqueueLatestHistoryCancelsStaleScroll(t *testing.T) {
	term := &Terminal{historyContext: context.Background()}
	started := make(chan struct{})
	canceled := make(chan struct{})
	latest := make(chan struct{})
	if !term.EnqueueLatestHistory(func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(canceled)
	}) {
		t.Fatal("first latest history request was rejected")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first history request did not start")
	}
	if !term.EnqueueLatestHistory(func(context.Context) { close(latest) }) {
		t.Fatal("replacement latest history request was rejected")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("stale history request was not canceled")
	}
	select {
	case <-latest:
	case <-time.After(time.Second):
		t.Fatal("replacement history request did not run")
	}
}

func TestHistoryRoutingStaysHostOwnedUntilLatestScrollCompletes(t *testing.T) {
	term := &Terminal{}
	first := term.BeginHistoryScroll("", 1)
	if !term.HistoryRoutingActive("") {
		t.Fatal("pending upward scroll must keep wheel routing in the host")
	}

	// A reverse wheel supersedes the first request. Completing the stale
	// request must not hand routing back to the mouse-aware child.
	second := term.BeginHistoryScroll("", -1)
	term.EndHistoryScroll("", first)
	if !term.HistoryRoutingActive("") {
		t.Fatal("stale completion cleared newer history routing")
	}
	term.EndHistoryScroll("", second)
	if term.HistoryRoutingActive("") {
		t.Fatal("latest completion left history routing active")
	}
}

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

func TestTerminalDetachKeepsProcessAndReattachRestoresInput(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Command: []string{"cat"}})
	defer h.Close()
	term := attachTerminal(t, h, "detach")
	if err := h.Detach("local", "detach"); err != nil {
		t.Fatalf("detach: %v", err)
	}
	if term.Attached() {
		t.Fatal("detached terminal still reports attached")
	}
	if _, ok := h.TerminalBySource(SourceID("local", "detach")); ok {
		t.Fatal("detached terminal still accepts input routing")
	}
	if _, pending := h.Handle(Request{Method: Method{Name: "terminal.attach"}, Params: &pb.MethodParams{Endpoint: "local", Id: "detach"}}); pending {
		t.Fatal("reattach unexpectedly pending")
	}
	if !term.Attached() {
		t.Fatal("reattach did not restore attachment")
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
	if term.Offset("") != 1 {
		t.Fatalf("offset = %d, want 1", term.Offset(""))
	}

	h.Handle(Request{
		Method: Method{Name: "terminal.scrollEnd"},
		Params: &pb.MethodParams{Endpoint: "local", Id: "t4"},
	})
	if term.Offset("") != 0 {
		t.Fatalf("offset after scrollEnd = %d, want 0", term.Offset(""))
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

func TestTerminalOwnerLeaseAllowsTakeoverAfterSilence(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Command: []string{"cat"}, OwnerID: "view:a", OwnerLeaseTTL: 20 * time.Millisecond})
	defer h.Close()
	term := attachTerminal(t, h, "lease")
	if _, err := term.ClaimOwner("view:b", 0); err != ErrOwnerConflict {
		t.Fatalf("early takeover error = %v, want owner conflict", err)
	}
	time.Sleep(30 * time.Millisecond)
	if _, err := term.ClaimOwner("view:b", 0); err != nil {
		t.Fatalf("expired lease takeover: %v", err)
	}
	if owner, _ := term.Owner(); owner != "view:b" {
		t.Fatalf("owner = %q, want view:b", owner)
	}
}

func TestSharedHandlerUsesPerRequestViewOwner(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{Command: []string{"cat"}, OwnerLeaseTTL: time.Second})
	defer h.Close()
	fit := true
	params := &pb.MethodParams{Endpoint: "local", Id: "shared", Fit: &fit}
	out, _ := h.Handle(Request{OwnerID: "view:a", Method: Method{Name: "terminal.attach"}, Params: params})
	if !out.OK {
		t.Fatal(out.Error)
	}
	term, _ := h.TerminalBySource(SourceID("local", "shared"))
	if owner, _ := term.Owner(); owner != "view:a" {
		t.Fatalf("initial owner = %q, want view:a", owner)
	}
	if out, _ := h.Handle(Request{OwnerID: "view:b", Method: Method{Name: "terminal.attach"}, Params: params}); out.OK || out.Error != ErrOwnerConflict.Error() {
		t.Fatalf("second view attach = %+v, want owner conflict", out)
	}
}

func TestTerminalHandlerOutputListenersAreIndependent(t *testing.T) {
	var first, second int
	h := NewTerminalHandler(TerminalOptions{OnOutput: func(string) { first++ }})
	remove := h.SubscribeOutput(func(string) { second++ })
	h.notifyOutput("terminal:local:t")
	if first != 1 || second != 1 {
		t.Fatalf("first notification = %d/%d, want 1/1", first, second)
	}
	remove()
	h.notifyOutput("terminal:local:t")
	if first != 2 || second != 1 {
		t.Fatalf("after unsubscribe = %d/%d, want 2/1", first, second)
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
	if !placement.DisableScrollOptimization {
		t.Fatal("live terminal placement must disable guessed physical scrolling")
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

func TestCopyWindowSelection(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{
		Cols:    12,
		Rows:    2,
		Command: []string{"sh", "-c", "printf 'one\\ntwo\\nthree'; sleep 10"},
	})
	defer h.Close()
	term := attachTerminal(t, h, "tcopy")
	waitFor(t, "three lines", func() bool { return screenContains(term, "three") })

	// Live window (offset 0): rows are "two", "three".
	charSpec := CopySpec{Mode: "char", StartRow: 0, StartCol: 0, EndRow: 1, EndCol: 2}
	if got, ok := term.CopyWindow("", charSpec); !ok || got != "two\nthr" {
		t.Fatalf("char copy = %q ok=%v, want %q", got, ok, "two\nthr")
	}
	lineSpec := CopySpec{Mode: "line", StartRow: 0, StartCol: 0, EndRow: 1, EndCol: 0}
	if got, _ := term.CopyWindow("", lineSpec); got != "two\nthree" {
		t.Fatalf("line copy = %q, want %q", got, "two\nthree")
	}
	blockSpec := CopySpec{Mode: "block", StartRow: 0, StartCol: 0, EndRow: 1, EndCol: 2}
	if got, _ := term.CopyWindow("", blockSpec); got != "two\nthr" {
		t.Fatalf("block copy = %q, want %q", got, "two\nthr")
	}
	// Reversed char selection normalizes to reading order.
	reversed := CopySpec{Mode: "char", StartRow: 1, StartCol: 2, EndRow: 0, EndCol: 0}
	if got, _ := term.CopyWindow("", reversed); got != "two\nthr" {
		t.Fatalf("reversed copy = %q, want %q", got, "two\nthr")
	}
	// A scrolled window resolves against what the view shows.
	if _, offset := term.Scroll("", 1, 2); offset != 1 {
		t.Fatalf("scroll offset = %d, want 1", offset)
	}
	if got, _ := term.CopyWindow("", CopySpec{Mode: "char", StartRow: 0, StartCol: 0, EndRow: 0, EndCol: 2}); got != "one" {
		t.Fatalf("scrolled copy = %q, want %q", got, "one")
	}
}

func TestLocalHistoryVisibleScreenPreservesStyles(t *testing.T) {
	term := &Terminal{parser: ansi.New(12, 2)}
	term.parser.Write([]byte("\x1b[38;2;18;52;86;48;2;101;67;33;1m界red\x1b[0m"))
	live := term.VisibleScreen("").Line(0)
	term.parser.Write([]byte("\r\nsecond\r\nlatest"))
	if _, offset := term.Scroll("", 1, 2); offset != 1 {
		t.Fatalf("offset=%d", offset)
	}
	frozen := term.VisibleScreen("").Line(0)
	if len(live) != len(frozen) {
		t.Fatalf("live/history widths changed: %d/%d", len(live), len(frozen))
	}
	for i := range live {
		if frozen[i] != live[i] {
			t.Fatalf("cell %d lost its style: live=%+v history=%+v", i, live[i], frozen[i])
		}
	}
	if live[0].Style == "" {
		t.Fatal("test did not generate colored content")
	}
}

func TestTerminalCopySelectionWritesClipboard(t *testing.T) {
	var copied []string
	h := NewTerminalHandler(TerminalOptions{
		Cols:      12,
		Rows:      2,
		Command:   []string{"sh", "-c", "printf 'alpha\\nbeta'; sleep 10"},
		Clipboard: func(text string) error { copied = append(copied, text); return nil },
	})
	defer h.Close()
	term := attachTerminal(t, h, "tsel")
	waitFor(t, "beta", func() bool { return screenContains(term, "beta") })
	cols := term.Cols()
	// The live window is ["alpha", "beta"]; select the second row (inclusive).
	start := 1*cols + 0
	end := 1*cols + 3
	outcome, pending := h.Handle(Request{
		Method: Method{Name: "terminal.copy"},
		Params: &pb.MethodParams{
			Endpoint: "local", Id: "tsel",
			Sel: &pb.Selection{Mode: "char", Start: int32(start), End: int32(end)},
		},
	})
	if !outcome.OK || pending {
		t.Fatalf("copy = %+v pending=%v", outcome, pending)
	}
	if len(copied) != 1 || copied[0] != "beta" {
		t.Fatalf("clipboard = %q, want [beta]", copied)
	}
	// Without sel the whole visible window is copied.
	h.Handle(Request{
		Method: Method{Name: "terminal.copy"},
		Params: &pb.MethodParams{Endpoint: "local", Id: "tsel"},
	})
	if len(copied) != 2 || copied[1] != "alpha\nbeta" {
		t.Fatalf("visible copy = %q", copied)
	}
}

// TestFrozenWindowDoesNotFollowOutput pins the old frozen copy/history
// semantics: once scrolled, new terminal output must not move the window the
// user is reading (the offset simply grows), and ScrollEnd returns to live.
func TestFrozenWindowDoesNotFollowOutput(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{
		Cols:    12,
		Rows:    2,
		Command: []string{"sh", "-c", "printf 'one\\ntwo\\nthree\\n'; sleep 1; printf 'four\\nfive\\n'; sleep 10"},
	})
	defer h.Close()
	term := attachTerminal(t, h, "tfrozen")
	waitFor(t, "three", func() bool { return screenContains(term, "three") })

	frozen, offset := term.Scroll("", 1, 2)
	if offset != 1 || len(frozen) != 2 || frozen[0] != "two" || frozen[1] != "three" {
		t.Fatalf("scroll window = %q offset=%d, want [two three] offset 1", frozen, offset)
	}
	// New output arrives; the frozen window must not move. The visible window
	// is requested with offset 0 (relative to the frozen bottom).
	waitFor(t, "five", func() bool { return screenContains(term, "five") })
	still, clamped := term.Window("", 0, 2)
	if len(still) != 2 || still[0] != "two" || still[1] != "three" {
		t.Fatalf("frozen window drifted with output: %q", still)
	}
	if clamped != 0 {
		t.Fatalf("visible window request must clamp to offset 0, got %d", clamped)
	}
	if distance := term.Offset(""); distance <= offset {
		t.Fatalf("live distance must grow while the view is frozen: %d -> %d", offset, distance)
	}
	// A larger scan window ends at the same frozen row; a positive offset
	// pages further back.
	scan, _ := term.Window("", 0, 10)
	if len(scan) < 2 || scan[len(scan)-1] != "three" {
		t.Fatalf("scan window must end at the frozen row: %q", scan)
	}
	paged, pagedOffset := term.Window("", 2, 2)
	if pagedOffset != 2 || len(paged) == 0 || paged[len(paged)-1] == "three" {
		t.Fatalf("paged scan window = %q offset=%d, want rows above the frozen view", paged, pagedOffset)
	}
	term.ScrollEnd("")
	live, offsetLive := term.Window("", 0, 2)
	if offsetLive != 0 || !strings.Contains(strings.Join(live, "\n"), "five") {
		t.Fatalf("live window after scrollEnd = %q offset=%d", live, offsetLive)
	}
}

// TestPerViewScrollIsolation pins the per-pane frozen viewport: two views bound
// to one shared Terminal must not share scroll/copy state. Scrolling view "a"
// freezes only "a"; view "b" stays live, renders the live tail and reports no
// offset. ScrollEnd("a") then returns only "a" to live.
func TestPerViewScrollIsolation(t *testing.T) {
	h := NewTerminalHandler(TerminalOptions{
		Cols:    12,
		Rows:    2,
		Command: []string{"sh", "-c", "printf 'one\\ntwo\\nthree\\n'; sleep 10"},
	})
	defer h.Close()
	term := attachTerminal(t, h, "tviews")
	waitFor(t, "three", func() bool { return screenContains(term, "three") })

	// Scrolling one view must not leak into a sibling on the same source.
	if _, offset := term.Scroll("a", 1, 2); offset == 0 {
		t.Fatalf("scroll view a offset = %d, want > 0", offset)
	}
	if term.Offset("a") == 0 {
		t.Fatalf("view a offset = %d, want > 0 after scroll", term.Offset("a"))
	}
	if term.Offset("b") != 0 {
		t.Fatalf("view b offset = %d, want 0 (live)", term.Offset("b"))
	}
	if !term.HistoryActive("a") {
		t.Fatal("view a must be frozen after its own scroll")
	}
	if term.HistoryActive("b") {
		t.Fatal("view b must stay live when only view a scrolls")
	}
	// The sibling renders the live tail, not view a's frozen window.
	if got := strings.Join(term.VisibleScreen("b").TextLines(), "|"); !strings.Contains(got, "three") {
		t.Fatalf("view b visible screen = %q, want the live tail", got)
	}

	term.ScrollEnd("a")
	if term.Offset("a") != 0 || term.HistoryActive("a") {
		t.Fatalf("after ScrollEnd(a) offset=%d active=%v, want 0/false", term.Offset("a"), term.HistoryActive("a"))
	}
}
