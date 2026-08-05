package terminalhost

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/tui/app"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

type fakeTerminalOps struct {
	mu       sync.Mutex
	terminal bool
	cols     int
	rows     int
	raw      int
	restored int
	sizeFD   uintptr
}

func (ops *fakeTerminalOps) IsTerminal(uintptr) bool {
	return ops.terminal
}

func (ops *fakeTerminalOps) MakeRaw(uintptr) (TerminalState, error) {
	ops.mu.Lock()
	defer ops.mu.Unlock()
	ops.raw++
	return "state", nil
}

func (ops *fakeTerminalOps) Restore(uintptr, TerminalState) error {
	ops.mu.Lock()
	defer ops.mu.Unlock()
	ops.restored++
	return nil
}

func (ops *fakeTerminalOps) GetSize(fd uintptr) (int, int, error) {
	ops.mu.Lock()
	defer ops.mu.Unlock()
	ops.sizeFD = fd
	return ops.cols, ops.rows, nil
}

func (ops *fakeTerminalOps) SizeFD() uintptr {
	ops.mu.Lock()
	defer ops.mu.Unlock()
	return ops.sizeFD
}

func (ops *fakeTerminalOps) RawCount() int {
	ops.mu.Lock()
	defer ops.mu.Unlock()
	return ops.raw
}

func (ops *fakeTerminalOps) RestoredCount() int {
	ops.mu.Lock()
	defer ops.mu.Unlock()
	return ops.restored
}

type blockingCancelReader struct {
	events chan []byte
	done   chan struct{}
}

type failNthWriter struct {
	writes int
	failAt int
	buffer bytes.Buffer
}

type shortNthWriter struct {
	writes  int
	shortAt int
	buffer  bytes.Buffer
}

func (writer *shortNthWriter) Write(data []byte) (int, error) {
	writer.writes++
	if writer.writes == writer.shortAt && len(data) > 1 {
		return len(data) - 1, nil
	}
	return writer.buffer.Write(data)
}

func (writer *failNthWriter) Write(data []byte) (int, error) {
	writer.writes++
	if writer.writes == writer.failAt {
		return 0, errors.New("enter write failed")
	}
	return writer.buffer.Write(data)
}

func newBlockingCancelReader() *blockingCancelReader {
	return &blockingCancelReader{
		events: make(chan []byte, 8),
		done:   make(chan struct{}),
	}
}

func (reader *blockingCancelReader) Read(p []byte) (int, error) {
	select {
	case data := <-reader.events:
		copy(p, data)
		return len(data), nil
	case <-reader.done:
		return 0, io.EOF
	}
}

func (reader *blockingCancelReader) Cancel() bool {
	select {
	case <-reader.done:
		return false
	default:
		close(reader.done)
		return true
	}
}

func TestHostImplementsAppTerminalHost(t *testing.T) {
	var _ app.TerminalHost = (*Host)(nil)
}

func TestHostEnterCloseRestoresTerminal(t *testing.T) {
	ops := &fakeTerminalOps{terminal: true}
	reader := newBlockingCancelReader()
	var output bytes.Buffer
	host := New(
		WithInput(strings.NewReader(""), 7),
		WithOutput(&output),
		WithTerminalOps(ops),
		WithCancelReaderFactory(func(io.Reader) (CancelReader, error) {
			return reader, nil
		}),
	)

	if err := host.Enter(context.Background()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	if got := ops.RawCount(); got != 1 {
		t.Fatalf("expected raw mode once, got %d", got)
	}
	gotEnter := output.String()
	for _, seq := range []string{enterAltScreen, hideCursor, enableKeyboardDisambiguation, queryKeyboardEnhancements, enableBracketPaste, enableMouseCell, enableMouseButton, enableMouseSGR} {
		if !strings.Contains(gotEnter, seq) {
			t.Fatalf("missing enter sequence %q in %q", seq, gotEnter)
		}
	}
	for _, seq := range []string{requestHostFG, requestHostBG, "\x1b]4;0;?\x07", "\x1b]4;15;?\x07"} {
		if !strings.Contains(gotEnter, seq) {
			t.Fatalf("missing theme probe sequence %q in %q", seq, gotEnter)
		}
	}
	if err := host.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := host.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if got := ops.RestoredCount(); got != 1 {
		t.Fatalf("expected restore once, got %d", got)
	}
	gotAll := output.String()
	for _, seq := range []string{disableMouseSGR, disableMouseButton, disableMouseCell, disableBracketPaste, disableKeyboardEnhancements, showCursor, exitAltScreen} {
		if !strings.Contains(gotAll, seq) {
			t.Fatalf("missing exit sequence %q in %q", seq, gotAll)
		}
	}
	if got := strings.Count(gotAll, disableKeyboardEnhancements); got != 1 {
		t.Fatalf("normal close and repeated close must pop keyboard protocol exactly once, got %d in %q", got, gotAll)
	}
}

func TestInputParserDecodesKittyCSIUCtrlDigits(t *testing.T) {
	parser := NewInputParser()
	got := parser.Feed([]byte("\x1b[49;5u\x1b[57;5:1u"))
	want := []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "1", Ctrl: true, RawSeq: "\x1b[49;5u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "9", Ctrl: true, RawSeq: "\x1b[57;5:1u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected CSI-u ctrl digit events\n got: %#v\nwant: %#v", got, want)
	}
}

func TestInputParserProjectsKeyboardCapabilityAndSuppressesRelease(t *testing.T) {
	got := NewInputParser().Feed([]byte("\x1b[?1u\x1b[49;5:3u"))
	want := []input.InputEvent{
		{Kind: input.EventKindHostCapability, Capability: input.HostCapabilityEvent{KeyboardDisambiguation: true}, RawSeq: "\x1b[?1u"},
		{Kind: input.EventKindKey, Key: input.KeyUnknown, Ctrl: true, RawSeq: "\x1b[49;5:3u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("capability/release decode mismatch: got=%#v want=%#v", got, want)
	}
}

func TestInputParserBuffersAndValidatesKittyCSIU(t *testing.T) {
	parser := NewInputParser()
	if got := parser.Feed([]byte("\x1b[49;")); len(got) != 0 {
		t.Fatalf("partial CSI-u must stay buffered, got %#v", got)
	}
	got := parser.Feed([]byte("5u"))
	want := []input.InputEvent{{Kind: input.EventKindKey, Key: input.KeyChar, Char: "1", Ctrl: true, RawSeq: "\x1b[49;5u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buffered CSI-u mismatch: got=%#v want=%#v", got, want)
	}

	invalid := NewInputParser().Feed([]byte("\x1b[49;0u"))
	if len(invalid) != 1 || invalid[0].Kind != input.EventKindHostControl || invalid[0].KeyboardProtocol != input.KeyboardProtocolKittyCSIU {
		t.Fatalf("invalid CSI-u must remain protocol-owned host control, got %#v", invalid)
	}
	for _, raw := range []string{"\x1b[49:97;5u", "\x1b[49;5:4u"} {
		invalid := NewInputParser().Feed([]byte(raw))
		if len(invalid) != 1 || invalid[0].Kind != input.EventKindHostControl || invalid[0].KeyboardProtocol != input.KeyboardProtocolKittyCSIU {
			t.Fatalf("unsupported CSI-u structure %q must remain protocol-owned, got %#v", raw, invalid)
		}
	}
	unsupportedModifier := NewInputParser().Feed([]byte("\x1b[49;9u"))
	if len(unsupportedModifier) != 1 || unsupportedModifier[0].Key != input.KeyUnknown || unsupportedModifier[0].KeyboardProtocol != input.KeyboardProtocolKittyCSIU {
		t.Fatalf("valid unsupported CSI-u modifier must be consumed without matching, got %#v", unsupportedModifier)
	}
}

func TestInputParserKeepsPlainDigitsUnmodified(t *testing.T) {
	parser := NewInputParser()
	got := parser.Feed([]byte("1"))
	want := []input.InputEvent{{Kind: input.EventKindKey, Key: input.KeyChar, Char: "1", RawSeq: "1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("plain digit must not be guessed as ctrl input: got=%#v want=%#v", got, want)
	}
}

func TestInputParserDecodesKittyCSIUControlKeys(t *testing.T) {
	got := NewInputParser().Feed([]byte("\x1b[27u\x1b[13u\x1b[9u\x1b[127u\x1b[9;2u\x1b[13;3u\x1b[27;3u\x1b[127;3u\x1b[9;5u"))
	want := []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyEsc, RawSeq: "\x1b[27u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyEnter, RawSeq: "\x1b[13u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyTab, RawSeq: "\x1b[9u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyBackspace, RawSeq: "\x1b[127u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyShiftTab, Shift: true, RawSeq: "\x1b[9;2u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyEnter, Alt: true, RawSeq: "\x1b[13;3u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyEsc, Alt: true, RawSeq: "\x1b[27;3u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyBackspace, Alt: true, RawSeq: "\x1b[127;3u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyTab, Ctrl: true, RawSeq: "\x1b[9;5u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CSI-u control key decode mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestKittyCSIUEscClosesConfiguredOverlays(t *testing.T) {
	events := NewInputParser().Feed([]byte("\x1b[27u"))
	if len(events) != 1 {
		t.Fatalf("expected one CSI-u esc event, got %#v", events)
	}
	cases := []struct {
		name string
		root state.Root
	}{
		{name: "create terminal prompt", root: state.Root{Shell: state.DefaultShell().OpenPrompt(state.PromptState{Title: "Create Terminal", Purpose: "terminal.create"})}},
		{name: "help", root: state.Root{Shell: state.DefaultShell().OpenHelp("most-used")}},
		{name: "terminal picker", root: state.Root{Shell: state.DefaultShell().OpenTerminalPicker()}},
		{name: "terminal manager", root: state.Root{Shell: state.DefaultShell().OpenTerminalPool()}},
		{name: "workbench tree", root: state.Root{Shell: state.DefaultShell().OpenWorkbenchTree()}},
		{name: "clipboard history", root: state.Root{Shell: state.DefaultShell().OpenClipboardHistory()}},
		{name: "floating overview", root: state.Root{Shell: state.DefaultShell().OpenFloatingOverview()}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runtime := app.NewInteractiveRuntime(tc.root, app.NewFakeTerminalHost(1), app.NewSyncEffectRunner(), app.LiveDeps{}, app.CopyModeDeps{})
			if err := runtime.Post(app.InputMsg{Event: events[0]}); err != nil {
				t.Fatalf("post CSI-u esc: %v", err)
			}
			if err := runtime.Drain(context.Background()); err != nil {
				t.Fatalf("drain CSI-u esc: %v", err)
			}
			if runtime.State().Shell.EnsureDefaults().Overlay.Open {
				t.Fatalf("CSI-u esc should close overlay: shell=%#v", runtime.State().Shell)
			}
		})
	}
}

func TestKittyCSIUEscReleaseDoesNotCloseOverlay(t *testing.T) {
	events := NewInputParser().Feed([]byte("\x1b[27;1:3u"))
	if len(events) != 1 || events[0].Key != input.KeyUnknown {
		t.Fatalf("expected one CSI-u esc release event, got %#v", events)
	}
	runtime := app.NewInteractiveRuntime(
		state.Root{Shell: state.DefaultShell().OpenHelp("most-used")},
		app.NewFakeTerminalHost(1),
		app.NewSyncEffectRunner(),
		app.LiveDeps{},
		app.CopyModeDeps{},
	)
	if err := runtime.Post(app.InputMsg{Event: events[0]}); err != nil {
		t.Fatalf("post CSI-u esc release: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain CSI-u esc release: %v", err)
	}
	if !runtime.State().Shell.EnsureDefaults().Overlay.Open {
		t.Fatal("CSI-u esc release must not close overlay")
	}
}

func TestHostThemeProbeCanBeDisabled(t *testing.T) {
	ops := &fakeTerminalOps{terminal: true}
	reader := newBlockingCancelReader()
	var output bytes.Buffer
	host := New(
		WithInput(strings.NewReader(""), 7),
		WithOutput(&output),
		WithTerminalOps(ops),
		WithThemeProbe(false),
		WithCancelReaderFactory(func(io.Reader) (CancelReader, error) {
			return reader, nil
		}),
	)
	if err := host.Enter(context.Background()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	if got := output.String(); strings.Contains(got, requestHostFG) || strings.Contains(got, "\x1b]4;0;?\x07") {
		t.Fatalf("theme probe should be disabled, got %q", got)
	}
	if err := host.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestHostEnterWriteFailureRestoresKeyboardProtocolAndTerminal(t *testing.T) {
	ops := &fakeTerminalOps{terminal: true}
	output := &failNthWriter{failAt: 1}
	host := New(
		WithInput(strings.NewReader(""), 7),
		WithOutput(output),
		WithTerminalOps(ops),
		WithThemeProbe(false),
	)
	err := host.Enter(context.Background())
	if err == nil || !strings.Contains(err.Error(), "enter write failed") {
		t.Fatalf("expected original enter write error, got %v", err)
	}
	if got := ops.RestoredCount(); got != 1 {
		t.Fatalf("enter write failure must restore terminal once, got %d", got)
	}
	if got := output.buffer.String(); strings.Contains(got, disableKeyboardEnhancements) || !strings.Contains(got, exitAltScreen) {
		t.Fatalf("failure before keyboard push must not pop an empty keyboard stack, got %q", got)
	}
}

func TestHostFailureAfterKeyboardPushPopsProtocol(t *testing.T) {
	ops := &fakeTerminalOps{terminal: true}
	output := &failNthWriter{failAt: 3}
	host := New(
		WithInput(strings.NewReader(""), 7),
		WithOutput(output),
		WithTerminalOps(ops),
		WithThemeProbe(false),
	)
	if err := host.Enter(context.Background()); err == nil {
		t.Fatal("expected capability query write failure")
	}
	if got := output.buffer.String(); !strings.Contains(got, enableKeyboardDisambiguation) || !strings.Contains(got, disableKeyboardEnhancements) {
		t.Fatalf("failure after keyboard push must pop the pushed mode, got %q", got)
	}
	if got := strings.Count(output.buffer.String(), disableKeyboardEnhancements); got != 1 {
		t.Fatalf("failure after push must pop exactly once, got %d", got)
	}
}

func TestHostKeyboardPushShortWriteDoesNotPop(t *testing.T) {
	ops := &fakeTerminalOps{terminal: true}
	output := &shortNthWriter{shortAt: 2}
	host := New(WithInput(strings.NewReader(""), 7), WithOutput(output), WithTerminalOps(ops), WithThemeProbe(false))
	if err := host.Enter(context.Background()); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("expected keyboard push short write, got %v", err)
	}
	if strings.Contains(output.buffer.String(), disableKeyboardEnhancements) {
		t.Fatalf("partial keyboard push must not pop an unconfirmed stack entry: %q", output.buffer.String())
	}
}

func TestHostKeyboardQueryShortWritePopsSuccessfulPush(t *testing.T) {
	ops := &fakeTerminalOps{terminal: true}
	output := &shortNthWriter{shortAt: 3}
	host := New(WithInput(strings.NewReader(""), 7), WithOutput(output), WithTerminalOps(ops), WithThemeProbe(false))
	if err := host.Enter(context.Background()); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("expected keyboard query short write, got %v", err)
	}
	if got := strings.Count(output.buffer.String(), disableKeyboardEnhancements); got != 1 {
		t.Fatalf("query short write after successful push must pop exactly once, got %d in %q", got, output.buffer.String())
	}
}

func TestRawKittyCSIUCtrlDigitRoutesToConfiguredTabJump(t *testing.T) {
	events := NewInputParser().Feed([]byte("\x1b[49;5u"))
	if len(events) != 1 {
		t.Fatalf("expected one parsed event, got %#v", events)
	}
	shortcuts := state.TUIShortcutConfig{Scenes: map[string]state.TUIShortcutSceneConfig{
		"global": {Bindings: map[string]state.TUIShortcutBindingConfig{"ctrl-1": {Action: "tab.jump.1"}}},
	}}
	intent := input.RouteWithOptions(events[0], input.RouteOptions{Shortcuts: shortcuts})
	if intent.Kind != input.IntentShortcutAction || intent.Invocation.ID != "tab.jump" {
		t.Fatalf("raw CSI-u ctrl-1 did not reach tab jump invocation: event=%#v intent=%#v", events[0], intent)
	}
	if index, ok := intent.Invocation.Param("index"); !ok || index != 1 {
		t.Fatalf("raw CSI-u ctrl-1 lost index: %#v", intent.Invocation)
	}
}

func TestRejectedKittyCSIUNeverLeaksToPTY(t *testing.T) {
	for _, raw := range []string{"\x1b[49:97;5u", "\x1b[49;9u", "\x1b[49;5:4u"} {
		events := NewInputParser().Feed([]byte(raw))
		if len(events) != 1 || events[0].KeyboardProtocol != input.KeyboardProtocolKittyCSIU {
			t.Fatalf("rejected CSI-u must retain protocol source: raw=%q events=%#v", raw, events)
		}
		if intent := input.RouteWithOptions(events[0], input.RouteOptions{}); intent.Kind != input.IntentNone {
			t.Fatalf("rejected CSI-u must not leak to PTY: raw=%q intent=%#v", raw, intent)
		}
	}
}

func TestHostRejectsNonTerminal(t *testing.T) {
	host := New(WithTerminalOps(&fakeTerminalOps{}))
	if err := host.Enter(context.Background()); !errors.Is(err, ErrNotTerminal) {
		t.Fatalf("expected ErrNotTerminal, got %v", err)
	}
}

func TestHostReportsWindowSize(t *testing.T) {
	ops := &fakeTerminalOps{cols: 132, rows: 43}
	host := New(WithInput(strings.NewReader(""), 7), WithSizeFD(9), WithTerminalOps(ops))
	cols, rows, err := host.Size()
	if err != nil {
		t.Fatalf("size: %v", err)
	}
	if cols != 132 || rows != 43 {
		t.Fatalf("unexpected size %dx%d", cols, rows)
	}
	if got := ops.SizeFD(); got != 9 {
		t.Fatalf("window size queried fd %d, want output fd 9", got)
	}
}

func TestHostDefaultsWindowSizeToStdout(t *testing.T) {
	host := New()
	if host.sizeFD != os.Stdout.Fd() {
		t.Fatalf("default size fd = %d, want stdout fd %d", host.sizeFD, os.Stdout.Fd())
	}
	if host.fd != os.Stdin.Fd() {
		t.Fatalf("default input fd = %d, want stdin fd %d", host.fd, os.Stdin.Fd())
	}
}

func TestHostPublishesResizeEventsFromSignal(t *testing.T) {
	ops := &fakeTerminalOps{terminal: true, cols: 90, rows: 30}
	reader := newBlockingCancelReader()
	resizeSignals := make(chan os.Signal, 2)
	host := New(
		WithInput(strings.NewReader(""), 7),
		WithOutput(io.Discard),
		WithTerminalOps(ops),
		WithThemeProbe(false),
		WithCancelReaderFactory(func(io.Reader) (CancelReader, error) {
			return reader, nil
		}),
		WithResizeSignalChannel(resizeSignals),
	)
	if err := host.Enter(context.Background()); err != nil {
		t.Fatalf("enter: %v", err)
	}

	resizeSignals <- testSignal{}
	got := readEvent(t, host)
	if got.Kind != input.EventKindResize || got.Cols != 90 || got.Rows != 30 {
		t.Fatalf("expected resize event from signal, got %#v", got)
	}

	ops.mu.Lock()
	ops.cols = 120
	ops.rows = 40
	ops.mu.Unlock()
	resizeSignals <- testSignal{}
	got = readEvent(t, host)
	if got.Kind != input.EventKindResize || got.Cols != 120 || got.Rows != 40 {
		t.Fatalf("expected updated resize event from signal, got %#v", got)
	}

	if err := host.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestHostReadsInputEventsAndStopsOnClose(t *testing.T) {
	ops := &fakeTerminalOps{terminal: true}
	reader := newBlockingCancelReader()
	host := New(
		WithInput(strings.NewReader(""), 7),
		WithOutput(io.Discard),
		WithTerminalOps(ops),
		WithCancelReaderFactory(func(io.Reader) (CancelReader, error) {
			return reader, nil
		}),
	)
	if err := host.Enter(context.Background()); err != nil {
		t.Fatalf("enter: %v", err)
	}

	reader.events <- []byte("x")
	reader.events <- []byte("\x1b[5~")
	reader.events <- []byte("\x1b[C")
	got := []input.InputEvent{readEvent(t, host), readEvent(t, host), readEvent(t, host)}
	want := []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x", RawSeq: "x"},
		{Kind: input.EventKindKey, Key: input.KeyPageUp, RawSeq: "\x1b[5~"},
		{Kind: input.EventKindKey, Key: input.KeyRight, RawSeq: "\x1b[C"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected events\n got: %#v\nwant: %#v", got, want)
	}

	if err := host.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if got := ops.RestoredCount(); got != 1 {
		t.Fatalf("expected restore once, got %d", got)
	}
}

func TestHostReassemblesSplitEscapeAndFlushesLoneEscape(t *testing.T) {
	ops := &fakeTerminalOps{terminal: true}
	reader := newBlockingCancelReader()
	host := New(
		WithInput(strings.NewReader(""), 7),
		WithOutput(io.Discard),
		WithTerminalOps(ops),
		WithThemeProbe(false),
		WithCancelReaderFactory(func(io.Reader) (CancelReader, error) { return reader, nil }),
	)
	if err := host.Enter(context.Background()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	reader.events <- []byte("\x1b")
	reader.events <- []byte("[5~")
	if got := readEvent(t, host); got.Key != input.KeyPageUp || got.RawSeq != "\x1b[5~" {
		t.Fatalf("split CSI must remain one input event, got %#v", got)
	}
	reader.events <- []byte("\x1b")
	if got := readEvent(t, host); got.Key != input.KeyEsc || got.RawSeq != "\x1b" {
		t.Fatalf("lone escape must flush after the ambiguity window, got %#v", got)
	}
	if err := host.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestHostRestoresTerminalWhenContextIsCanceled(t *testing.T) {
	ops := &fakeTerminalOps{terminal: true}
	reader := newBlockingCancelReader()
	ctx, cancel := context.WithCancel(context.Background())
	host := New(
		WithInput(strings.NewReader(""), 7),
		WithOutput(io.Discard),
		WithTerminalOps(ops),
		WithCancelReaderFactory(func(io.Reader) (CancelReader, error) {
			return reader, nil
		}),
	)
	if err := host.Enter(ctx); err != nil {
		t.Fatalf("enter: %v", err)
	}
	cancel()

	deadline := time.After(time.Second)
	for ops.RestoredCount() == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for context restore")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := host.Close(); err != nil {
		t.Fatalf("close after cancel: %v", err)
	}
	if got := ops.RestoredCount(); got != 1 {
		t.Fatalf("expected restore once, got %d", got)
	}
}

func TestInputParserConvertsMouseAndKeys(t *testing.T) {
	parser := NewInputParser()
	got := parser.Feed([]byte("\x1b[<64;10;4M\x1b[<65;11;5M\x1b[<0;12;6M\x1b[<32;13;6M\x1b[<0;13;6m\r好"))
	want := []input.InputEvent{
		{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp, Row: 4, Col: 10, RawSeq: "\x1b[<64;10;4M"},
		{Kind: input.EventKindMouse, Mouse: input.MouseWheelDown, Row: 5, Col: 11, RawSeq: "\x1b[<65;11;5M"},
		{Kind: input.EventKindMouse, Mouse: input.MouseLeft, Row: 6, Col: 12, RawSeq: "\x1b[<0;12;6M"},
		{Kind: input.EventKindMouse, Mouse: input.MouseLeftDrag, Row: 6, Col: 13, RawSeq: "\x1b[<32;13;6M"},
		{Kind: input.EventKindMouse, Mouse: input.MouseLeftUp, Row: 6, Col: 13, RawSeq: "\x1b[<0;13;6m"},
		{Kind: input.EventKindKey, Key: input.KeyEnter, RawSeq: "\r"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "好", RawSeq: "好"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected parsed events\n got: %#v\nwant: %#v", got, want)
	}
}

func TestInputParserConsumesOSCThemeResponses(t *testing.T) {
	parser := NewInputParser()
	got := parser.Feed([]byte("\x1b]10;rgb:aaaa/bbbb/cccc\a\x1b]11;#010203\x1b\\\x1b]4;5;rgb:4444/8888/cccc\a"))
	want := []input.InputEvent{
		{Kind: input.EventKindHostTheme, Theme: input.HostThemeEvent{DefaultFG: "#aabbcc"}, RawSeq: "\x1b]10;rgb:aaaa/bbbb/cccc\a"},
		{Kind: input.EventKindHostTheme, Theme: input.HostThemeEvent{DefaultBG: "#010203"}, RawSeq: "\x1b]11;#010203\x1b\\"},
		{Kind: input.EventKindHostTheme, Theme: input.HostThemeEvent{PaletteIndex: 5, PaletteColor: "#4488cc"}, RawSeq: "\x1b]4;5;rgb:4444/8888/cccc\a"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected OSC theme events\n got: %#v\nwant: %#v", got, want)
	}
	for _, event := range got {
		if event.Kind == input.EventKindKey {
			t.Fatalf("theme OSC response must not leak as key input: %#v", event)
		}
	}
}

func TestInputParserSuppressesUnknownOSCResponses(t *testing.T) {
	got := NewInputParser().Feed([]byte("\x1b]52;ignored\x07"))
	if len(got) != 1 || got[0].Kind != input.EventKindHostControl || got[0].RawSeq != "\x1b]52;ignored\x07" {
		t.Fatalf("unknown OSC must remain a host control event, got %#v", got)
	}
	if intent := input.RouteWithOptions(got[0], input.RouteOptions{}); intent.Kind != input.IntentNone {
		t.Fatalf("unknown OSC response must not leak to PTY, got %#v", intent)
	}
}

func TestInputParserConvertsExtendedKeysAndModifiers(t *testing.T) {
	parser := NewInputParser()
	got := parser.Feed([]byte("\x1b[Z\x1b[H\x1b[F\x1b[2~\x1b[3~\x1bOP\x1b[15~\x1b[1;5D\x1b[1;3C"))
	want := []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyShiftTab, Shift: true, RawSeq: "\x1b[Z"},
		{Kind: input.EventKindKey, Key: input.KeyHome, RawSeq: "\x1b[H"},
		{Kind: input.EventKindKey, Key: input.KeyEnd, RawSeq: "\x1b[F"},
		{Kind: input.EventKindKey, Key: input.KeyInsert, RawSeq: "\x1b[2~"},
		{Kind: input.EventKindKey, Key: input.KeyDelete, RawSeq: "\x1b[3~"},
		{Kind: input.EventKindKey, Key: input.KeyF1, RawSeq: "\x1bOP"},
		{Kind: input.EventKindKey, Key: input.KeyF5, RawSeq: "\x1b[15~"},
		{Kind: input.EventKindKey, Key: input.KeyLeft, Ctrl: true, RawSeq: "\x1b[1;5D"},
		{Kind: input.EventKindKey, Key: input.KeyRight, Alt: true, RawSeq: "\x1b[1;3C"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected extended key events\n got: %#v\nwant: %#v", got, want)
	}
}

func TestInputParserConvertsSGRMouseButtonsAndRelease(t *testing.T) {
	parser := NewInputParser()
	got := parser.Feed([]byte("\x1b[<1;2;3M\x1b[<33;2;3M\x1b[<1;2;3m\x1b[<2;4;5M\x1b[<34;4;5M\x1b[<2;4;5m\x1b[<35;6;7M"))
	want := []input.InputEvent{
		{Kind: input.EventKindMouse, Mouse: input.MouseMiddle, Row: 3, Col: 2, RawSeq: "\x1b[<1;2;3M"},
		{Kind: input.EventKindMouse, Mouse: input.MouseMiddleDrag, Row: 3, Col: 2, RawSeq: "\x1b[<33;2;3M"},
		{Kind: input.EventKindMouse, Mouse: input.MouseMiddleUp, Row: 3, Col: 2, RawSeq: "\x1b[<1;2;3m"},
		{Kind: input.EventKindMouse, Mouse: input.MouseRight, Row: 5, Col: 4, RawSeq: "\x1b[<2;4;5M"},
		{Kind: input.EventKindMouse, Mouse: input.MouseRightDrag, Row: 5, Col: 4, RawSeq: "\x1b[<34;4;5M"},
		{Kind: input.EventKindMouse, Mouse: input.MouseRightUp, Row: 5, Col: 4, RawSeq: "\x1b[<2;4;5m"},
		{Kind: input.EventKindMouse, Mouse: input.MouseMove, Row: 7, Col: 6, RawSeq: "\x1b[<35;6;7M"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected SGR mouse events\n got: %#v\nwant: %#v", got, want)
	}
}

func TestInputParserMarksCtrlFAndCtrlV(t *testing.T) {
	parser := NewInputParser()
	got := parser.Feed([]byte("\x06\x16"))
	want := []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x06", Ctrl: true, RawSeq: "\x06"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true, RawSeq: "\x16"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected parsed ctrl events\n got: %#v\nwant: %#v", got, want)
	}
}

func TestInputParserKeepsPartialEscapeUntilComplete(t *testing.T) {
	parser := NewInputParser()
	if got := parser.Feed([]byte("\x1b[5")); len(got) != 0 {
		t.Fatalf("expected no partial events, got %#v", got)
	}
	got := parser.Feed([]byte("~"))
	want := []input.InputEvent{{Kind: input.EventKindKey, Key: input.KeyPageUp, RawSeq: "\x1b[5~"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected completion %#v", got)
	}
}

func TestInputParserKeepsSplitEscapePrefixUntilSequenceOrFlush(t *testing.T) {
	parser := NewInputParser()
	if got := parser.Feed([]byte("\x1b")); len(got) != 0 {
		t.Fatalf("ambiguous escape prefix must wait for the next raw chunk, got %#v", got)
	}
	got := parser.Feed([]byte("[5~"))
	want := []input.InputEvent{{Kind: input.EventKindKey, Key: input.KeyPageUp, RawSeq: "\x1b[5~"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("split escape sequence mismatch: got=%#v want=%#v", got, want)
	}

	parser = NewInputParser()
	_ = parser.Feed([]byte("\x1b"))
	got = parser.FlushEscape()
	want = []input.InputEvent{{Kind: input.EventKindKey, Key: input.KeyEsc, RawSeq: "\x1b"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lone escape flush mismatch: got=%#v want=%#v", got, want)
	}

	parser = NewInputParser()
	if got := parser.Feed([]byte("\x1b[")); len(got) != 0 {
		t.Fatalf("ambiguous CSI prefix must wait for sequence completion or timeout, got %#v", got)
	}
	got = parser.FlushEscape()
	want = []input.InputEvent{{Kind: input.EventKindKey, Key: input.KeyChar, Char: "[", Alt: true, RawSeq: "\x1b["}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("timed out CSI prefix must preserve Alt-[ semantics: got=%#v want=%#v", got, want)
	}
}

func TestInputParserNormalizesTraditionalAltControlKeys(t *testing.T) {
	got := NewInputParser().Feed([]byte("\x1b\r\x1b\t\x1b\x7f\x1b\x03\x1b\x1b"))
	want := []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyEnter, Alt: true, RawSeq: "\x1b\r"},
		{Kind: input.EventKindKey, Key: input.KeyTab, Alt: true, RawSeq: "\x1b\t"},
		{Kind: input.EventKindKey, Key: input.KeyBackspace, Alt: true, RawSeq: "\x1b\x7f"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x03", Alt: true, Ctrl: true, RawSeq: "\x1b\x03"},
		{Kind: input.EventKindKey, Key: input.KeyEsc, Alt: true, RawSeq: "\x1b\x1b"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("traditional alt/control normalization mismatch:\n got=%#v\nwant=%#v", got, want)
	}
}

func TestInputParserSuppressesUnsupportedKittyFunctionalKeys(t *testing.T) {
	got := NewInputParser().Feed([]byte("\x1b[57376;5u"))
	if len(got) != 1 || got[0].Key != input.KeyUnknown || got[0].Char != "" || got[0].KeyboardProtocol != input.KeyboardProtocolKittyCSIU {
		t.Fatalf("unsupported Kitty PUA key must remain protocol-owned unknown input, got %#v", got)
	}
	if intent := input.RouteWithOptions(got[0], input.RouteOptions{}); intent.Kind != input.IntentNone {
		t.Fatalf("unsupported Kitty PUA key must not leak to PTY, got %#v", intent)
	}
}

func TestInputParserNormalizesKittyModifierLocksAndConsumesUnsupportedModifiers(t *testing.T) {
	got := NewInputParser().Feed([]byte("\x1b[99;69u\x1b[99;133u\x1b[99;13u\x1b[1;9D"))
	want := []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c", Ctrl: true, RawSeq: "\x1b[99;69u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c", Ctrl: true, RawSeq: "\x1b[99;133u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindKey, Key: input.KeyUnknown, Ctrl: true, RawSeq: "\x1b[99;13u", KeyboardProtocol: input.KeyboardProtocolKittyCSIU},
		{Kind: input.EventKindHostControl, RawSeq: "\x1b[1;9D"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Kitty modifier normalization mismatch:\n got=%#v\nwant=%#v", got, want)
	}
}

func TestInputParserFramesBracketedPasteAcrossChunks(t *testing.T) {
	parser := NewInputParser()
	if got := parser.Feed([]byte("\x1b[20")); len(got) != 0 {
		t.Fatalf("partial paste start marker must stay buffered, got %#v", got)
	}
	if got := parser.Feed([]byte("0~\x07\ntail\x1b[20")); len(got) != 0 {
		t.Fatalf("partial paste must remain atomic, got %#v", got)
	}
	got := parser.Feed([]byte("1~x"))
	want := []input.InputEvent{
		{Kind: input.EventKindPaste, Paste: "\x07\ntail"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x", RawSeq: "x"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bracketed paste framing mismatch:\n got=%#v\nwant=%#v", got, want)
	}
	intent := input.RouteWithOptions(got[0], input.RouteOptions{})
	if intent.Kind != input.IntentTerminalInput || string(intent.Bytes) != "\x07\ntail" {
		t.Fatalf("paste semantic must bypass shortcuts while retaining only its body, got %#v", intent)
	}
}

func TestInputParserRejectsInvalidSGRMouseCoordinates(t *testing.T) {
	for _, raw := range []string{"\x1b[<0;0;1M", "\x1b[<0;1;0M", "\x1b[<64;1;1m", "\x1b[<66;1;1M", "\x1b[<128;1;1M", "\x1b[<3;1;1M"} {
		got := NewInputParser().Feed([]byte(raw))
		if len(got) != 1 || got[0].Kind != input.EventKindHostControl {
			t.Fatalf("invalid SGR mouse %q must be consumed as host control, got %#v", raw, got)
		}
		if intent := input.RouteWithOptions(got[0], input.RouteOptions{}); intent.Kind != input.IntentNone {
			t.Fatalf("invalid SGR mouse %q must not leak to PTY, got %#v", raw, intent)
		}
	}
}

func TestFrameSinkWritesFrameToOutput(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	if err := sink.WriteFrame(render.Frame{Lines: []string{"one", "two"}, Metadata: render.RenderMetadata{Width: 3, Height: 2}}); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	got := output.String()
	if !strings.HasPrefix(got, synchronizedOutputBegin+hideCursor) || !strings.HasSuffix(got, synchronizedOutputEnd) {
		t.Fatalf("FrameSink should wrap frame in synchronized output and hide cursor before repaint, got %q", got)
	}
	for _, part := range []string{cursorHome, clearScreen, cursorPosition(1, 1) + clearLine + "one", cursorPosition(2, 1) + clearLine + "two"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing frame part %q in %q", part, got)
		}
	}
	if !strings.Contains(got, render.ANSIReset+hideCursor+synchronizedOutputEnd) {
		t.Fatalf("FrameSink should hide host cursor by default, got %q", got)
	}
	if strings.Contains(got, "\n") {
		t.Fatalf("FrameSink must not use linefeed row progression, got %q", got)
	}
}

func TestFrameSinkWritesFE0FLineWithModelColumnAnchors(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	line := render.Line{Cells: []render.Cell{
		{Text: "♻️", Width: 2, TerminalContent: true, Safe: true},
		{Text: "♻️", Width: 2, TerminalContent: true, Safe: true},
		{Text: "♻️", Width: 2, TerminalContent: true, Safe: true},
		{Text: "·", Width: 1, Safe: true},
		{Text: "·", Width: 1, Safe: true},
	}}
	frame := render.Frame{
		ANSILines: []string{line.ANSIString(render.DefaultTheme())},
		Metadata:  render.RenderMetadata{Width: 8, Height: 1},
	}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write frame: %v", err)
	}

	got := output.String()
	for _, part := range []string{
		"♻️\x1b[1X\x1b[3G♻️",
		"♻️\x1b[1X\x1b[5G♻️",
		"♻️\x1b[1X\x1b[7G·",
	} {
		if !strings.Contains(got, part) {
			t.Fatalf("FE0F frame should erase continuation cells and preserve model column anchors %q in %q", part, got)
		}
	}
}

func TestFrameSinkSkipsUnchangedRows(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{Lines: []string{"one", "two"}, Metadata: render.RenderMetadata{Width: 3, Height: 2}}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	output.Reset()
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write second frame: %v", err)
	}
	got := output.String()
	if got != "" {
		t.Fatalf("unchanged frame should not write to host output, got %q", got)
	}
}

func TestFrameSinkWritesOnlyChangedRows(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	if err := sink.WriteFrame(render.Frame{Lines: []string{"one", "two"}, Metadata: render.RenderMetadata{Width: 5, Height: 2}}); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	output.Reset()
	if err := sink.WriteFrame(render.Frame{Lines: []string{"one", "three"}, Metadata: render.RenderMetadata{Width: 5, Height: 2}}); err != nil {
		t.Fatalf("write second frame: %v", err)
	}
	got := output.String()
	if strings.Contains(got, clearScreen) {
		t.Fatalf("same-size changed frame should not clear the screen, got %q", got)
	}
	if strings.Contains(got, clearLine) || strings.Contains(got, cursorPosition(1, 1)+"one") || !strings.Contains(got, cursorPosition(2, 1)+"three") {
		t.Fatalf("expected only second row presenter write without clear-line, got %q", got)
	}
}

func TestFrameSinkClearsChangedANSIAddressedRows(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	first := "prefix" + "\x1b[16G" + "\x1b[48;5;236m" + strings.Repeat(" ", 10) + "\x1b[0m" + "tail"
	second := "短" + "\x1b[10Gtail"
	if err := sink.WriteFrame(render.Frame{ANSILines: []string{first}, Metadata: render.RenderMetadata{Width: 30, Height: 1}}); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	output.Reset()
	if err := sink.WriteFrame(render.Frame{ANSILines: []string{second}, Metadata: render.RenderMetadata{Width: 30, Height: 1}}); err != nil {
		t.Fatalf("write second frame: %v", err)
	}
	got := output.String()
	if strings.Contains(got, clearScreen) {
		t.Fatalf("same-size addressed row update should not clear screen, got %q", got)
	}
	want := cursorPosition(1, 1) + render.ANSIReset + clearLine + second
	if !strings.Contains(got, want) {
		t.Fatalf("addressed ANSI rows must clear the whole row before repaint, want %q in %q", want, got)
	}
}

func TestFrameSinkForceFullRepaintClearsScreen(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	if err := sink.WriteFrame(render.Frame{Lines: []string{"one", "two"}, Metadata: render.RenderMetadata{Width: 5, Height: 2}}); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	output.Reset()
	if err := sink.WriteFrame(render.Frame{Lines: []string{"one", "three"}, Metadata: render.RenderMetadata{Width: 5, Height: 2, ForceFullRepaint: true}}); err != nil {
		t.Fatalf("write repaint frame: %v", err)
	}
	got := output.String()
	if !strings.Contains(got, cursorHome+clearScreen) || !strings.Contains(got, cursorPosition(1, 1)+clearLine+"one") || !strings.Contains(got, cursorPosition(2, 1)+clearLine+"three") {
		t.Fatalf("force-full repaint must clear and rewrite complete frame, got %q", got)
	}
}

func TestFrameSinkUsesScrollRegionForOneRowShiftUp(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	if err := sink.WriteFrame(render.Frame{
		Lines:    []string{"top", "one", "two", "three", "bottom"},
		Metadata: render.RenderMetadata{Width: 8, Height: 5},
	}); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	output.Reset()
	if err := sink.WriteFrame(render.Frame{
		Lines:    []string{"top", "two", "three", "four", "bottom"},
		Metadata: render.RenderMetadata{Width: 8, Height: 5},
	}); err != nil {
		t.Fatalf("write shifted frame: %v", err)
	}
	got := output.String()
	if strings.Contains(got, clearScreen) {
		t.Fatalf("one-row scroll should not clear screen, got %q", got)
	}
	for _, part := range []string{scrollRegion(2, 4), cursorPosition(4, 1) + scrollUpOne, resetScrollRegion, cursorPosition(4, 1) + clearLine + "four"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing scroll-region part %q in %q", part, got)
		}
	}
	if strings.Contains(got, cursorPosition(2, 1)+clearLine+"two") || strings.Contains(got, cursorPosition(3, 1)+clearLine+"three") {
		t.Fatalf("one-row scroll should not repaint shifted rows, got %q", got)
	}
}

func TestFrameSinkUsesScrollRegionForOneRowShiftDown(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	if err := sink.WriteFrame(render.Frame{
		Lines:    []string{"top", "two", "three", "four", "bottom"},
		Metadata: render.RenderMetadata{Width: 8, Height: 5},
	}); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	output.Reset()
	if err := sink.WriteFrame(render.Frame{
		Lines:    []string{"top", "one", "two", "three", "bottom"},
		Metadata: render.RenderMetadata{Width: 8, Height: 5},
	}); err != nil {
		t.Fatalf("write shifted frame: %v", err)
	}
	got := output.String()
	for _, part := range []string{scrollRegion(2, 4), cursorPosition(2, 1) + scrollDownOne, resetScrollRegion, cursorPosition(2, 1) + clearLine + "one"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing scroll-region part %q in %q", part, got)
		}
	}
	if strings.Contains(got, cursorPosition(3, 1)+clearLine+"two") || strings.Contains(got, cursorPosition(4, 1)+clearLine+"three") {
		t.Fatalf("one-row scroll should not repaint shifted rows, got %q", got)
	}
}

func TestFrameSinkWritesIncrementalScrollPatch(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{
		Patch: &render.FramePatch{
			Rect:      render.Rect{X: 1, Y: 2, W: 10, H: 4},
			Dir:       render.FramePatchScrollUp,
			LineY:     5,
			LineX:     1,
			LineWidth: 10,
			LineANSI:  "new row",
		},
		Cursor:     render.Cursor{Visible: true, Shape: render.CursorShapeBlock},
		CursorRect: render.Rect{X: 3, Y: 4, W: 1, H: 1},
		Metadata:   render.RenderMetadata{Width: 20, Height: 10},
	}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write patch frame: %v", err)
	}
	got := output.String()
	for _, part := range []string{scrollRegion(3, 6), cursorPosition(6, 1) + scrollUpOne, resetScrollRegion, cursorPosition(6, 2) + render.ANSIReset + eraseChars(10) + "new row"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing patch part %q in %q", part, got)
		}
	}
	if strings.Contains(got, clearScreen) || strings.Contains(got, clearLine) {
		t.Fatalf("incremental patch must not clear whole screen or line, got %q", got)
	}
}

func TestFrameSinkWritesMultiLineIncrementalScrollPatch(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{
		Patch: &render.FramePatch{
			Rect:      render.Rect{X: 2, Y: 1, W: 8, H: 6},
			Dir:       render.FramePatchScrollUp,
			LineY:     5,
			LineX:     2,
			LineWidth: 8,
			LinesANSI: []string{"new a", "new b"},
		},
		Metadata: render.RenderMetadata{Width: 20, Height: 10},
	}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write patch frame: %v", err)
	}
	got := output.String()
	for _, part := range []string{scrollRegion(2, 7), cursorPosition(7, 1) + scrollUpOne + scrollUpOne, cursorPosition(6, 3) + render.ANSIReset + eraseChars(8) + "new a", cursorPosition(7, 3) + render.ANSIReset + eraseChars(8) + "new b"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing multi-line patch part %q in %q", part, got)
		}
	}
}

func TestFrameSinkWritesRewritePatchWithoutScrollRegion(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{
		Patch: &render.FramePatch{
			Rect:      render.Rect{X: 2, Y: 1, W: 8, H: 3},
			Rewrite:   true,
			LineY:     1,
			LineX:     2,
			LineWidth: 8,
			LinesANSI: []string{"row a", "row b", "row c"},
		},
		Cursor:     render.Cursor{Visible: true, Shape: render.CursorShapeBlock},
		CursorRect: render.Rect{X: 4, Y: 2, W: 1, H: 1},
		Metadata:   render.RenderMetadata{Width: 20, Height: 10},
	}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write rewrite patch frame: %v", err)
	}
	got := output.String()
	for _, part := range []string{cursorPosition(2, 3) + render.ANSIReset + eraseChars(8) + "row a", cursorPosition(3, 3) + render.ANSIReset + eraseChars(8) + "row b", cursorPosition(4, 3) + render.ANSIReset + eraseChars(8) + "row c"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing rewrite patch part %q in %q", part, got)
		}
	}
	if strings.Contains(got, scrollRegion(2, 4)) || strings.Contains(got, scrollUpOne) || strings.Contains(got, scrollDownOne) || strings.Contains(got, clearLine) {
		t.Fatalf("rewrite patch must not scroll or clear pane border columns, got %q", got)
	}
}

func TestFrameSinkWritesSingleRowRewritePatch(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{
		Patch: &render.FramePatch{
			Rect:      render.Rect{X: 2, Y: 1, W: 8, H: 1},
			Rewrite:   true,
			LineY:     1,
			LineX:     2,
			LineWidth: 8,
			LinesANSI: []string{"only row"},
		},
		Metadata: render.RenderMetadata{Width: 20, Height: 4},
	}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write single-row patch: %v", err)
	}
	if got := output.String(); !strings.Contains(got, cursorPosition(2, 3)+render.ANSIReset+eraseChars(8)+"only row") {
		t.Fatalf("single-row rewrite patch was not written: %q", got)
	}
}

func TestFrameSinkWritesCursorOnlyChange(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{Lines: []string{"one"}, Metadata: render.RenderMetadata{Width: 3, Height: 1}}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	output.Reset()
	frame.Cursor = render.Cursor{Visible: true, Shape: render.CursorShapeBar}
	frame.CursorRect = render.Rect{X: 1, Y: 0, W: 1, H: 1}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write cursor frame: %v", err)
	}
	got := output.String()
	if strings.Contains(got, clearScreen) || strings.Contains(got, "one") {
		t.Fatalf("cursor-only change should not repaint rows, got %q", got)
	}
	if !strings.Contains(got, cursorShapeBar+cursorPosition(1, 2)+showCursor) {
		t.Fatalf("cursor-only change should write cursor sequence, got %q", got)
	}
}

func TestFrameSinkWritesCursorOnlyPatchWithoutRowDiff(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{
		Patch:      &render.FramePatch{CursorOnly: true},
		Cursor:     render.Cursor{Visible: true, Shape: render.CursorShapeBlock},
		CursorRect: render.Rect{X: 2, Y: 3, W: 1, H: 1},
	}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write cursor-only patch: %v", err)
	}
	got := output.String()
	if strings.Contains(got, clearScreen) || strings.Contains(got, clearLine) || strings.Contains(got, scrollUpOne) || strings.Contains(got, scrollDownOne) {
		t.Fatalf("cursor-only patch must not repaint or scroll, got %q", got)
	}
	if !strings.Contains(got, cursorShapeBlock+cursorPosition(4, 3)+showCursor) {
		t.Fatalf("cursor-only patch should only project cursor, got %q", got)
	}
}

func TestFrameSinkClearsOnFrameSizeChange(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	if err := sink.WriteFrame(render.Frame{Lines: []string{"one", "two"}, Metadata: render.RenderMetadata{Width: 3, Height: 2}}); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	output.Reset()
	if err := sink.WriteFrame(render.Frame{Lines: []string{"one", "two", "three"}, Metadata: render.RenderMetadata{Width: 5, Height: 3}}); err != nil {
		t.Fatalf("write resized frame: %v", err)
	}
	got := output.String()
	if !strings.Contains(got, cursorHome+clearScreen) || !strings.Contains(got, cursorPosition(3, 1)+clearLine+"three") {
		t.Fatalf("resized frame should repaint fully, got %q", got)
	}
}

func TestFrameSinkShowsVisibleCursorAtFrameCursorRect(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{
		Lines:      []string{"one", "two"},
		Cursor:     render.Cursor{Visible: true, Row: 0, Col: 0, Shape: render.CursorShapeBar},
		CursorRect: render.Rect{X: 4, Y: 2, W: 1, H: 1},
	}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	got := output.String()
	if !strings.Contains(got, render.ANSIReset+cursorShapeBar+cursorPosition(3, 5)+showCursor+synchronizedOutputEnd) {
		t.Fatalf("FrameSink should show host cursor at global cursor rect, got %q", got)
	}
	if !strings.HasPrefix(got, synchronizedOutputBegin+hideCursor) {
		t.Fatalf("FrameSink should hide cursor during repaint before showing final cursor, got %q", got)
	}
}

func TestFrameSinkParksHiddenCursorForAnchorOnlyFrame(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{
		Lines:      []string{"one", "two"},
		Cursor:     render.Cursor{Anchor: true, Shape: render.CursorShapeBar},
		CursorRect: render.Rect{X: 4, Y: 2, W: 1, H: 1},
	}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	got := output.String()
	if !strings.Contains(got, render.ANSIReset+hideCursor+cursorShapeBar+cursorPosition(3, 5)+synchronizedOutputEnd) {
		t.Fatalf("FrameSink should park hidden host cursor at global anchor rect, got %q", got)
	}
	if strings.Contains(got, showCursor) {
		t.Fatalf("anchor-only cursor must remain hidden, got %q", got)
	}
}

func TestFrameSinkPositionsFullWidthRowsWithoutLinefeeds(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{Lines: []string{strings.Repeat("─", 8), "│      │"}, Metadata: render.RenderMetadata{Width: 8, Height: 2}}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	got := output.String()
	if strings.Contains(got, "\n") {
		t.Fatalf("full-width frame must use absolute row positioning instead of linefeeds, got %q", got)
	}
	if !strings.Contains(got, cursorPosition(2, 1)+clearLine+"│      │") {
		t.Fatalf("expected second row absolute position, got %q", got)
	}
}

func TestFrameSinkPrefersANSIStyledFrame(t *testing.T) {
	var output bytes.Buffer
	sink := NewFrameSink(&output)
	frame := render.Frame{
		Lines:     []string{"plain"},
		ANSILines: []string{"\x1b[31mstyled\x1b[0m"},
		Metadata:  render.RenderMetadata{Width: 6, Height: 1},
	}
	if err := sink.WriteFrame(frame); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	got := output.String()
	if !strings.Contains(got, "\x1b[31mstyled\x1b[0m") {
		t.Fatalf("expected ANSI styled line in output, got %q", got)
	}
	if strings.Contains(got, "plain") {
		t.Fatalf("FrameSink must not use plain snapshot when ANSI lines are present, got %q", got)
	}
	if !strings.Contains(got, render.ANSIReset+hideCursor) {
		t.Fatalf("FrameSink must reset style before final cursor sequence, got %q", got)
	}
}

func readEvent(t *testing.T, host *Host) input.InputEvent {
	t.Helper()
	select {
	case event := <-host.InputEvents():
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for input event")
		return input.InputEvent{}
	}
}

type testSignal struct{}

func (testSignal) String() string {
	return "test-signal"
}

func (testSignal) Signal() {}
