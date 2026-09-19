package terminal

import (
	"errors"
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/render"
)

type historyCall struct {
	offset int
	rows   int
}

type fakeHistory struct {
	calls []historyCall
	lines []string
	err   error
}

func (f *fakeHistory) Window(offset, rows int) ([]string, error) {
	f.calls = append(f.calls, historyCall{offset: offset, rows: rows})
	if f.err != nil {
		return nil, f.err
	}
	end := len(f.lines) - offset
	if end <= 0 {
		return nil, nil
	}
	start := end - rows
	if start < 0 {
		start = 0
	}
	return f.lines[start:end], nil
}

type fakeClipboard struct {
	text   string
	writes int
}

func (f *fakeClipboard) Write(text string) error {
	f.text = text
	f.writes++
	return nil
}

func newTestComponent(t *testing.T, history *fakeHistory, clip *fakeClipboard) *Component {
	t.Helper()
	component := New(history, clip)
	component.SetScreen(ScreenFromText([]string{"live-1", "live-2", "live-3"}, render.TokenDefault))
	component.Render(12, 5) // content height = 3, so scroll asks for 3 rows
	return component
}

func TestInsetFollowsDeclaredChrome(t *testing.T) {
	component := New(nil, nil)
	if got := component.Inset(12, 4); got != DefaultInset {
		t.Fatalf("Inset(12,4) = %d, want %d", got, DefaultInset)
	}
	if got := component.Inset(2, 4); got != 0 {
		t.Fatalf("Inset(2,4) = %d, want 0 (too narrow for chrome)", got)
	}
	if got := component.Inset(12, 2); got != 0 {
		t.Fatalf("Inset(12,2) = %d, want 0 (too short for chrome)", got)
	}
	component.SetProps(Props{Inset: 2})
	if got := component.Inset(12, 5); got != 2 {
		t.Fatalf("declared Inset(12,5) = %d, want 2", got)
	}
	if got := component.Inset(12, 4); got != 0 {
		t.Fatalf("declared Inset(12,4) = %d, want 0 (needs 2*inset+1)", got)
	}
	if got := component.Inset(4, 5); got != 0 {
		t.Fatalf("declared Inset(4,5) = %d, want 0 (needs 2*inset+1)", got)
	}
}

func TestInsetPropBorderlessTerminal(t *testing.T) {
	// chrome.inset=0 is the card-pane contract: the component draws content
	// only and the runtime gets the full rect as the PTY content area.
	if inset, ok := InsetFromProps(map[string]string{PropInset: "0"}); !ok || inset != 0 {
		t.Fatalf("InsetFromProps(0) = %d ok=%v, want 0 true", inset, ok)
	}
	if inset, ok := InsetFromProps(map[string]string{PropInset: "2"}); !ok || inset != 2 {
		t.Fatalf("InsetFromProps(2) = %d ok=%v, want 2 true", inset, ok)
	}
	if _, ok := InsetFromProps(map[string]string{PropInset: "nope"}); ok {
		t.Fatal("invalid inset must be ignored")
	}
	if _, ok := InsetFromProps(map[string]string{PropInset: "-1"}); ok {
		t.Fatal("negative inset must be ignored")
	}
	if _, ok := InsetFromProps(nil); ok {
		t.Fatal("missing inset must be ignored")
	}
	component := New(nil, nil)
	component.SetProps(Props{Inset: 0, InsetSet: true})
	if got := component.Inset(3, 2); got != 0 {
		t.Fatalf("borderless Inset(3,2) = %d, want 0", got)
	}
	component.SetScreen(ScreenFromText([]string{"content"}, render.TokenDefault))
	lines := component.Render(8, 3)
	for _, line := range lines {
		if strings.Contains(line.Text, "│") || strings.Contains(line.Text, "┌") {
			t.Fatalf("borderless render must not draw a frame: %+v", lines)
		}
	}
}

func TestScrollLoadsHistoryWindow(t *testing.T) {
	history := &fakeHistory{lines: []string{"h1", "h2", "h3", "h4", "h5"}}
	component := newTestComponent(t, history, nil)

	offset, err := component.Scroll(2)
	if err != nil {
		t.Fatalf("Scroll(2) error: %v", err)
	}
	if offset != 2 {
		t.Fatalf("offset = %d, want 2", offset)
	}
	if len(history.calls) != 1 || history.calls[0] != (historyCall{offset: 2, rows: 3}) {
		t.Fatalf("history calls = %+v, want [{2 3}]", history.calls)
	}
	want := []string{"h1", "h2", "h3"}
	got := component.VisibleLines()
	if len(got) != len(want) {
		t.Fatalf("visible = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("visible[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	props := component.Props()
	if !props.Scrolled || props.ScrollOffset != 2 {
		t.Fatalf("props = %+v, want scrolled offset 2", props)
	}
}

func TestScrollFurtherAndBack(t *testing.T) {
	history := &fakeHistory{lines: []string{"h1", "h2", "h3", "h4", "h5", "h6"}}
	component := newTestComponent(t, history, nil)
	if _, err := component.Scroll(1); err != nil {
		t.Fatalf("Scroll(1): %v", err)
	}
	if _, err := component.Scroll(2); err != nil {
		t.Fatalf("Scroll(2): %v", err)
	}
	if got := component.Offset(); got != 3 {
		t.Fatalf("offset = %d, want 3", got)
	}
	if _, err := component.Scroll(-99); err != nil {
		t.Fatalf("Scroll(-99): %v", err)
	}
	if got := component.Offset(); got != 0 {
		t.Fatalf("offset after clamp = %d, want 0", got)
	}
	if props := component.Props(); props.Scrolled || props.ScrollOffset != 0 {
		t.Fatalf("props after end = %+v, want live", props)
	}
	if got := component.VisibleLines(); len(got) != 3 || got[0] != "live-1" {
		t.Fatalf("visible after end = %q, want live rows", got)
	}
}

func TestScrollEndReturnsToLive(t *testing.T) {
	history := &fakeHistory{lines: []string{"h1", "h2", "h3", "h4"}}
	component := newTestComponent(t, history, nil)
	if _, err := component.Scroll(2); err != nil {
		t.Fatalf("Scroll(2): %v", err)
	}
	component.ScrollEnd()
	if component.Offset() != 0 {
		t.Fatalf("offset = %d, want 0", component.Offset())
	}
	if got := component.VisibleLines(); got[0] != "live-1" {
		t.Fatalf("visible = %q, want live", got)
	}
}

func TestScrollWithoutHistoryPort(t *testing.T) {
	component := New(nil, nil)
	component.SetScreen(ScreenFromText([]string{"x"}, render.TokenDefault))
	component.Render(12, 5)
	if _, err := component.Scroll(1); !errors.Is(err, ErrNoHistory) {
		t.Fatalf("Scroll error = %v, want ErrNoHistory", err)
	}
	if component.Offset() != 0 {
		t.Fatalf("offset = %d, want 0 after failed scroll", component.Offset())
	}
}

func TestScrollAtHistoryTopKeepsWindow(t *testing.T) {
	history := &fakeHistory{lines: []string{"h1", "h2"}}
	component := newTestComponent(t, history, nil)
	if _, err := component.Scroll(5); err != nil {
		t.Fatalf("Scroll(5): %v", err)
	}
	if got := component.Offset(); got != 0 {
		t.Fatalf("offset = %d, want 0 (empty window keeps the view)", got)
	}
}

func TestScrollPropagatesPortError(t *testing.T) {
	boom := errors.New("boom")
	history := &fakeHistory{err: boom}
	component := newTestComponent(t, history, nil)
	if _, err := component.Scroll(1); !errors.Is(err, boom) {
		t.Fatalf("Scroll error = %v, want boom", err)
	}
	if component.Offset() != 0 {
		t.Fatalf("offset = %d, want unchanged", component.Offset())
	}
}

func TestCopyVisibleLiveRows(t *testing.T) {
	clip := &fakeClipboard{}
	component := New(nil, clip)
	component.SetScreen(ScreenFromText([]string{"hello  ", "yo", "", ""}, render.TokenDefault))
	component.Render(12, 6)
	if err := component.Copy(); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if clip.writes != 1 {
		t.Fatalf("clipboard writes = %d, want 1", clip.writes)
	}
	if clip.text != "hello\nyo" {
		t.Fatalf("clipboard text = %q, want %q", clip.text, "hello\nyo")
	}
}

func TestCopyVisibleScrolledRows(t *testing.T) {
	history := &fakeHistory{lines: []string{"h1", "h2", "h3", "h4"}}
	clip := &fakeClipboard{}
	component := newTestComponent(t, history, clip)
	if _, err := component.Scroll(1); err != nil {
		t.Fatalf("Scroll(1): %v", err)
	}
	if err := component.Copy(); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if clip.text != "h1\nh2\nh3" {
		t.Fatalf("clipboard text = %q, want scrolled window", clip.text)
	}
}

func TestCopyWithoutClipboardPort(t *testing.T) {
	component := New(nil, nil)
	if err := component.Copy(); !errors.Is(err, ErrNoClipboard) {
		t.Fatalf("Copy error = %v, want ErrNoClipboard", err)
	}
}

func TestApplySourceExitedResetsScroll(t *testing.T) {
	history := &fakeHistory{lines: []string{"h1", "h2", "h3", "h4"}}
	component := newTestComponent(t, history, nil)
	if _, err := component.Scroll(2); err != nil {
		t.Fatalf("Scroll(2): %v", err)
	}
	component.ApplySource(Source{ID: "terminal:local:main", Title: "main", Attached: true, Exited: true, ExitCode: 7})
	props := component.Props()
	if !props.Exited || props.ExitCode != 7 {
		t.Fatalf("props = %+v, want exited code 7", props)
	}
	if props.Title != "main" {
		t.Fatalf("title = %q, want source title fallback", props.Title)
	}
	if props.Scrolled || component.Offset() != 0 {
		t.Fatalf("scroll state not reset on exit: %+v offset=%d", props, component.Offset())
	}
	if source := component.Source(); source.ID != "terminal:local:main" {
		t.Fatalf("source = %+v", source)
	}
}

func TestSetPropsScrollOffsetLoadsWindow(t *testing.T) {
	history := &fakeHistory{lines: []string{"h1", "h2", "h3", "h4"}}
	component := newTestComponent(t, history, nil)
	component.SetProps(Props{Title: "main", ScrollOffset: 2, Inset: 1})
	if component.Offset() != 2 {
		t.Fatalf("offset = %d, want 2", component.Offset())
	}
	if len(history.calls) != 1 || history.calls[0] != (historyCall{offset: 2, rows: 3}) {
		t.Fatalf("history calls = %+v", history.calls)
	}
	props := component.Props()
	if !props.Scrolled || props.ScrollOffset != 2 {
		t.Fatalf("props = %+v, want scrolled 2", props)
	}
}

func TestTitleTextBadges(t *testing.T) {
	tests := []struct {
		name  string
		props Props
		want  string
	}{
		{"plain", Props{Title: "main"}, "main"},
		{"fallback", Props{}, "terminal"},
		// Legacy parity: code 0 prints the bare badge (shell/main.go era
		// exitTitle), a non-zero code keeps the number.
		{"exited zero", Props{Title: "main", Exited: true}, "main [exited]"},
		{"exited code", Props{Title: "main", Exited: true, ExitCode: 3}, "main [exited 3]"},
		{"scrolled", Props{Title: "main", Scrolled: true, ScrollOffset: 12}, "main [↑12]"},
		{"exited zero and scrolled", Props{Title: "main", Exited: true, Scrolled: true, ScrollOffset: 4}, "main [exited] [↑4]"},
		{"exited and scrolled", Props{Title: "main", Exited: true, ExitCode: 1, Scrolled: true, ScrollOffset: 4}, "main [exited 1] [↑4]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TitleText(tt.props.withDefaults()); got != tt.want {
				t.Fatalf("TitleText = %q, want %q", got, tt.want)
			}
		})
	}
}
