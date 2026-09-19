package ansi

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/render"
)

func feed(p *Parser, chunks ...string) {
	for _, chunk := range chunks {
		p.Write([]byte(chunk))
	}
}

func rowText(s Screen, y int) string {
	return strings.TrimRight(s.Text(y), " ")
}

func styleAt(t *testing.T, s Screen, x, y int) render.Token {
	t.Helper()
	cell := s.CellAt(x, y)
	if cell.Continuation {
		t.Fatalf("cell (%d,%d) is a continuation cell", x, y)
	}
	return cell.Style
}

func textAt(t *testing.T, s Screen, x, y int) string {
	t.Helper()
	return s.CellAt(x, y).Text
}

func TestPrintableTextAndCursorAdvance(t *testing.T) {
	p := New(16, 2)
	feed(p, "hi\tthere")
	s := p.Screen()
	if got := rowText(s, 0); got != "hi      there" {
		t.Fatalf("row = %q, want tab stop at column 8", got)
	}
	if s.CursorX != 13 || s.CursorY != 0 {
		t.Fatalf("cursor = (%d,%d), want (13,0)", s.CursorX, s.CursorY)
	}
}

func TestSGRAttributesAndColors(t *testing.T) {
	p := New(32, 2)
	feed(p, "\x1b[1;31mbold-red\x1b[0m plain ")
	feed(p, "\x1b[38;5;196mred256\x1b[m ")
	feed(p, "\x1b[48;2;1;2;3mBG")
	s := p.Screen()

	if got := styleAt(t, s, 0, 0); string(got) != "ansi:1;31" {
		t.Fatalf("bold-red style = %q, want ansi:1;31", got)
	}
	if got := textAt(t, s, 0, 0); got != "b" {
		t.Fatalf("cell 0 = %q", got)
	}
	if got := styleAt(t, s, 9, 0); got != render.TokenDefault {
		t.Fatalf("plain style = %q, want default", got)
	}
	if got := styleAt(t, s, 15, 0); string(got) != "ansi:38;5;196" {
		t.Fatalf("256-color style = %q, want ansi:38;5;196", got)
	}
	if got := styleAt(t, s, 23, 0); string(got) != "ansi:48;2;1;2;3" {
		t.Fatalf("truecolor bg style = %q, want ansi:48;2;1;2;3", got)
	}
	if seq := styleAt(t, s, 0, 0).SGR(render.DefaultTheme()); seq != "\x1b[1;31m" {
		t.Fatalf("raw SGR = %q, want \\x1b[1;31m", seq)
	}
}

func TestSGRCombinedAttributes(t *testing.T) {
	p := New(20, 1)
	feed(p, "\x1b[1m\x1b[2m\x1b[3m\x1b[4m\x1b[7m\x1b[9mA")
	s := p.Screen()
	if got := string(styleAt(t, s, 0, 0)); got != "ansi:1;2;3;4;7;9" {
		t.Fatalf("style = %q, want all attributes", got)
	}
	feed(p, "\x1b[22m\x1b[23m\x1b[24m\x1b[27m\x1b[29mB")
	s = p.Screen()
	if got := styleAt(t, s, 1, 0); got != render.TokenDefault {
		t.Fatalf("after clears style = %q, want default", got)
	}
}

func TestSGRColonColors(t *testing.T) {
	p := New(10, 1)
	feed(p, "\x1b[38:5:42mA\x1b[38:2::10:20:30mB")
	s := p.Screen()
	if got := string(styleAt(t, s, 0, 0)); got != "ansi:38;5;42" {
		t.Fatalf("colon 256 style = %q", got)
	}
	if got := string(styleAt(t, s, 1, 0)); got != "ansi:38;2;10;20;30" {
		t.Fatalf("colon truecolor style = %q", got)
	}
}

func TestCursorPositioning(t *testing.T) {
	p := New(10, 3)
	feed(p, "abc")
	feed(p, "\x1b[1;2HX")
	s := p.Screen()
	if got := rowText(s, 0); got != "aXc" {
		t.Fatalf("CUP row = %q, want aXc", got)
	}
	feed(p, "\x1b[3;5HZ")
	s = p.Screen()
	if got := rowText(s, 2); got != "    Z" {
		t.Fatalf("CUP row 2 = %q, want 4 spaces + Z", got)
	}
	feed(p, "\x1b[3D")
	s = p.Screen()
	if s.CursorX != 2 || s.CursorY != 2 {
		t.Fatalf("cursor = (%d,%d), want (2,2)", s.CursorX, s.CursorY)
	}
}

func TestCursorMovementLetters(t *testing.T) {
	p := New(8, 4)
	feed(p, "\x1b[3;4H\x1b[2A") // up 2 -> row 0
	if x, y, _ := p.Cursor(); x != 3 || y != 0 {
		t.Fatalf("cursor after A = (%d,%d), want (3,0)", x, y)
	}
	feed(p, "\x1b[2B\x1b[3C")
	if x, y, _ := p.Cursor(); x != 6 || y != 2 {
		t.Fatalf("cursor after B/C = (%d,%d), want (6,2)", x, y)
	}
	feed(p, "\x1b[5D\x1b[2E")
	if x, y, _ := p.Cursor(); x != 0 || y != 3 {
		t.Fatalf("cursor after D/E = (%d,%d), want (0,3)", x, y)
	}
	feed(p, "\x1b[1F")
	if x, y, _ := p.Cursor(); x != 0 || y != 2 {
		t.Fatalf("cursor after F = (%d,%d), want (0,2)", x, y)
	}
	feed(p, "\x1b[4G")
	if x, _, _ := p.Cursor(); x != 3 {
		t.Fatalf("cursor after CHA = %d, want 3", x)
	}
	feed(p, "\x1b[3d")
	if _, y, _ := p.Cursor(); y != 2 {
		t.Fatalf("cursor after VPA = %d, want 2", y)
	}
}

func TestEraseLineModes(t *testing.T) {
	cases := []struct {
		name  string
		write string
		want  string
	}{
		{"to end", "abcdef\x1b[3G\x1b[0K", "ab"},
		{"to start", "abcdef\x1b[4G\x1b[1K", "    ef"},
		{"all", "abcdef\x1b[3G\x1b[2K", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New(6, 1)
			feed(p, tc.write)
			if got := rowText(p.Screen(), 0); got != tc.want {
				t.Fatalf("row = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEraseDisplayModes(t *testing.T) {
	p := New(4, 3)
	feed(p, "aaaa\r\nbbbb\r\ncccc")
	feed(p, "\x1b[2;2H\x1b[0J")
	s := p.Screen()
	if got := rowText(s, 0); got != "aaaa" {
		t.Fatalf("row 0 = %q, want untouched", got)
	}
	if got := rowText(s, 1); got != "b" {
		t.Fatalf("row 1 = %q, want b", got)
	}
	if got := rowText(s, 2); got != "" {
		t.Fatalf("row 2 = %q, want blank", got)
	}

	feed(p, "\x1b[2;2H\x1b[1J")
	s = p.Screen()
	if got := rowText(s, 0); got != "" {
		t.Fatalf("row 0 after J1 = %q, want blank", got)
	}
	if got := rowText(s, 1); got != "" {
		t.Fatalf("row 1 after J1 = %q, want blank", got)
	}

	feed(p, "\x1b[2J")
	s = p.Screen()
	for y := 0; y < 3; y++ {
		if got := rowText(s, y); got != "" {
			t.Fatalf("row %d after J2 = %q, want blank", y, got)
		}
	}
}

func TestScrollUpAndDown(t *testing.T) {
	p := New(5, 2)
	feed(p, "a\r\nb\r\nc")
	s := p.Screen()
	if got := rowText(s, 0); got != "b" {
		t.Fatalf("row 0 = %q, want b after scroll", got)
	}
	if got := rowText(s, 1); got != "c" {
		t.Fatalf("row 1 = %q, want c", got)
	}
	if sb := p.Scrollback(); len(sb) != 1 || sb[0][0].Text != "a" {
		t.Fatalf("scrollback = %+v, want [a]", sb)
	}

	feed(p, "\x1b[2S")
	s = p.Screen()
	if got := rowText(s, 0); got != "" {
		t.Fatalf("row 0 after S2 = %q, want blank", got)
	}
	if sb := p.Scrollback(); len(sb) != 3 {
		t.Fatalf("scrollback len = %d, want 3", len(sb))
	}

	feed(p, "\x1b[3;1H\x1b[2T")
	s = p.Screen()
	if got := rowText(s, 0); got != "" {
		t.Fatalf("row 0 after T2 = %q, want blank (content pushed off)", got)
	}
}

func TestWideCharOverwriteClearsOrphan(t *testing.T) {
	p := New(6, 1)
	feed(p, "你好")
	s := p.Screen()
	if s.CellAt(0, 0).Continuation || s.CellAt(0, 0).Width != 2 {
		t.Fatalf("cell 0 = %+v, want wide head", s.CellAt(0, 0))
	}
	if !s.CellAt(1, 0).Continuation {
		t.Fatalf("cell 1 = %+v, want continuation", s.CellAt(1, 0))
	}

	feed(p, "\x1b[1;2Hx")
	s = p.Screen()
	if !s.CellAt(0, 0).Blank() {
		t.Fatalf("cell 0 = %+v, want cleared head", s.CellAt(0, 0))
	}
	if got := textAt(t, s, 1, 0); got != "x" {
		t.Fatalf("cell 1 = %q, want x", got)
	}
	if got := textAt(t, s, 2, 0); got != "好" {
		t.Fatalf("cell 2 = %q, want 好", got)
	}
	if got := rowText(s, 0); got != " x好" {
		t.Fatalf("row = %q, want ' x好'", got)
	}

	p2 := New(4, 1)
	feed(p2, "ab你")
	feed(p2, "\x1b[1;3H") // erase the continuation half only
	feed(p2, "\x1b[0K")
	if got := rowText(p2.Screen(), 0); got != "ab" {
		t.Fatalf("row = %q, want ab after erasing continuation", got)
	}
}

func TestHalfPacketByteByByte(t *testing.T) {
	p := New(12, 1)
	stream := []byte("\x1b[38;5;196mhéllo你好")
	for i := 0; i < len(stream); i++ {
		p.Write(stream[i : i+1])
	}
	s := p.Screen()
	if got := rowText(s, 0); got != "héllo你好" {
		t.Fatalf("row = %q, want héllo你好", got)
	}
	if got := string(styleAt(t, s, 0, 0)); got != "ansi:38;5;196" {
		t.Fatalf("style = %q, want ansi:38;5;196", got)
	}
	if cell := s.CellAt(5, 0); cell.Text != "你" || cell.Width != 2 {
		t.Fatalf("wide cell after split feed = %+v", cell)
	}
}

func TestModesToggle(t *testing.T) {
	p := New(4, 2)
	if !p.Modes().CursorVisible {
		t.Fatal("cursor must start visible")
	}
	feed(p, "\x1b[?25l")
	if p.Modes().CursorVisible {
		t.Fatal("?25l must hide the cursor")
	}
	feed(p, "\x1b[?1000h\x1b[?1002h\x1b[?1003h\x1b[?1006h\x1b[?2004h")
	m := p.Modes()
	if !m.MouseCell || !m.MouseDrag || !m.MouseAny || !m.MouseSGR || !m.BracketPaste {
		t.Fatalf("modes after enable = %+v", m)
	}
	if !m.MouseTracking() {
		t.Fatal("MouseTracking must be true")
	}
	feed(p, "\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[?2004l\x1b[?25h")
	m = p.Modes()
	if m.MouseCell || m.MouseDrag || m.MouseAny || m.MouseSGR || m.BracketPaste || !m.CursorVisible {
		t.Fatalf("modes after disable = %+v", m)
	}
	if m.MouseTracking() {
		t.Fatal("MouseTracking must be false")
	}
	if !p.Screen().CursorVisible {
		t.Fatal("screen snapshot must reflect cursor visibility")
	}
}

func TestAltScreenSwitchesAndRestores(t *testing.T) {
	p := New(6, 1)
	feed(p, "main")
	feed(p, "\x1b[?1049h")
	if !p.Modes().AltScreen {
		t.Fatal("?1049h must enable the alternate screen")
	}
	if got := rowText(p.Screen(), 0); got != "" {
		t.Fatalf("alt row = %q, want clear", got)
	}
	feed(p, "alt")
	if got := rowText(p.Screen(), 0); got != "alt" {
		t.Fatalf("alt row = %q, want alt", got)
	}
	feed(p, "\x1b[?1049l")
	if p.Modes().AltScreen {
		t.Fatal("?1049l must leave the alternate screen")
	}
	if got := rowText(p.Screen(), 0); got != "main" {
		t.Fatalf("primary row = %q, want main restored", got)
	}
}

func TestSaveRestoreCursor(t *testing.T) {
	p := New(8, 2)
	feed(p, "ab\x1b[s")
	feed(p, "\x1b[2;1Hcd")
	feed(p, "\x1b[uX")
	if got := rowText(p.Screen(), 0); got != "abX" {
		t.Fatalf("row 0 = %q, want abX", got)
	}
	feed(p, "\x1b7\x1b[2;1HZ\x1b8Y")
	if got := rowText(p.Screen(), 0); got != "abXY" {
		t.Fatalf("row 0 = %q, want abXY", got)
	}
}

func TestCombiningAndEmojiStayOneCell(t *testing.T) {
	p := New(8, 1)
	feed(p, "e\u0301👍🏽")
	s := p.Screen()
	if got := textAt(t, s, 0, 0); got != "e\u0301" {
		t.Fatalf("combining cell = %q", got)
	}
	if cell := s.CellAt(1, 0); cell.Text != "👍🏽" || cell.Width != 2 {
		t.Fatalf("emoji cell = %+v, want width 2 cluster", cell)
	}
	if !s.CellAt(2, 0).Continuation {
		t.Fatalf("emoji continuation missing: %+v", s.CellAt(2, 0))
	}
}

func TestUnknownSequencesAreSkipped(t *testing.T) {
	p := New(8, 1)
	feed(p, "\x1b[?9999h\x1b]0;title\x07\x1b(0\x1bPpayload\x1b\\\x1b[>1uok")
	if got := rowText(p.Screen(), 0); got != "ok" {
		t.Fatalf("row = %q, want ok after unknown sequences", got)
	}
}

func TestResizePreservesOverlapAndDropsSplitWide(t *testing.T) {
	p := New(6, 2)
	feed(p, "hello\r\nworld")
	p.Resize(3, 1)
	s := p.Screen()
	if len(s.Lines) != 1 || rowText(s, 0) != "hel" {
		t.Fatalf("resize shrink = %+v, want one row 'hel'", s.Lines)
	}

	wide := New(5, 1)
	feed(wide, "ab你")
	wide.Resize(3, 1)
	if got := rowText(wide.Screen(), 0); got != "ab" {
		t.Fatalf("resize split wide row = %q, want ab", got)
	}

	wide.Resize(8, 2)
	s = wide.Screen()
	if len(s.Lines) != 2 || len(s.Lines[0]) != 8 {
		t.Fatalf("resize grow = %d rows, want 2x8", len(s.Lines))
	}
	if got := rowText(s, 0); got != "ab" {
		t.Fatalf("resize grow row = %q, want ab preserved", got)
	}
}

// TestZshPromptGolden replays a real zsh prompt: 256-color segments, resets
// and a powerline-style marker followed by the cursor.
func TestZshPromptGolden(t *testing.T) {
	p := New(40, 4)
	feed(p, "\x1b[1;38;5;39muser\x1b[0m@\x1b[1;38;5;213mhost\x1b[0m \x1b[38;5;214m~/src\x1b[0m \x1b[1;32m❯\x1b[0m ")
	s := p.Screen()

	if got := s.Text(0); got != "user@host ~/src ❯ " {
		t.Fatalf("prompt = %q", got)
	}
	if got := string(styleAt(t, s, 0, 0)); got != "ansi:1;38;5;39" {
		t.Fatalf("user style = %q", got)
	}
	if got := styleAt(t, s, 4, 0); got != render.TokenDefault {
		t.Fatalf("@ style = %q, want default", got)
	}
	if got := string(styleAt(t, s, 5, 0)); got != "ansi:1;38;5;213" {
		t.Fatalf("host style = %q", got)
	}
	if got := string(styleAt(t, s, 10, 0)); got != "ansi:38;5;214" {
		t.Fatalf("path style = %q", got)
	}
	if got := string(styleAt(t, s, 16, 0)); got != "ansi:1;32" {
		t.Fatalf("marker style = %q", got)
	}
	if cell := s.CellAt(16, 0); cell.Text != "❯" || cell.Width != 1 {
		t.Fatalf("marker cell = %+v", cell)
	}
	if s.CursorX != 18 || s.CursorY != 0 || !s.CursorVisible {
		t.Fatalf("cursor = (%d,%d,%v), want (18,0,true)", s.CursorX, s.CursorY, s.CursorVisible)
	}
}

func TestCursorOffsetFoldsWideClusters(t *testing.T) {
	p := New(8, 1)
	feed(p, "a你b")
	s := p.Screen()
	cases := []struct {
		x    int
		want int
	}{
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 3},
		{4, 4},
		{7, 7},
	}
	for _, tc := range cases {
		got, ok := s.CursorOffset(tc.x, 0)
		if !ok || got != tc.want {
			t.Fatalf("CursorOffset(%d) = %d,%v want %d,true", tc.x, got, ok, tc.want)
		}
	}
	if _, ok := s.CursorOffset(8, 0); ok {
		t.Fatal("out-of-range cursor column must report false")
	}
}

// TestStyleRunsCarryProgramColors converts the parsed row into per-style runs
// the way the terminal component does and verifies the framebuffer bytes.
func TestStyleRunsCarryProgramColors(t *testing.T) {
	p := New(8, 1)
	feed(p, "\x1b[1;31mhi\x1b[0m!")
	s := p.Screen()
	var lines []render.Line
	x := 0
	for _, cell := range s.Lines[0] {
		if cell.Continuation {
			continue
		}
		style := cell.Style
		if style == "" {
			style = render.TokenDefault
		}
		lines = append(lines, render.Line{X: x, Y: 0, Text: cell.Text, Style: style})
		x += cell.Width
	}
	frame := render.NewFrame(8, 1)
	frame.Blit(lines...)
	got := string(frame.FullBytes())
	if !strings.Contains(got, "hi") || !strings.Contains(got, "\x1b[1;31m") {
		t.Fatalf("full bytes %q must carry raw SGR and text", got)
	}
}
