package main

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// update rewrites the Go-owned screen goldens (go test -update). The shared
// python-shell goldens are never rewritten: footer/1.txt checks stay pinned to
// the reference files.
var update = flag.Bool("update", false, "rewrite testdata/golden screens")

// goldenDir is the Go-owned screen golden directory. The original has no
// separator gutter between sibling cards (adjacent borders), which differs
// from the Python reference's older 1-cell-divider approximation, so the
// screen goldens are regenerated here (go test -update) while the geometry is
// pinned by the formula tests below and the 1.txt oracle.
func goldenDir() string { return filepath.Join("testdata", "golden") }

// pythonGoldenDir is the shared reference golden directory (footer text and
// colors, the 1.txt capture).
func pythonGoldenDir() string { return filepath.Join("..", "python-shell", "golden") }

func demoModel(cols, rows int) *model {
	m := newModel(nil, true)
	m.viewID = "selftest"
	m.cols, m.rows = cols, rows
	return m
}

func screenLines(m *model) []string {
	lines, _ := m.rasterize(m.View())
	return lines
}

func rstrip(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = strings.TrimRight(line, " ")
	}
	return out
}

func compareGolden(t *testing.T, filename string, lines []string) {
	t.Helper()
	path := filepath.Join(goldenDir(), filename)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		content := strings.Join(rstrip(lines), "\n") + "\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", filename, err)
		}
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", filename, err)
	}
	want := rstrip(strings.Split(strings.TrimRight(string(data), "\n"), "\n"))
	got := rstrip(lines)
	if len(got) != len(want) {
		t.Fatalf("%s: %d rows, want %d", filename, len(got), len(want))
	}
	for y := range got {
		if got[y] != want[y] {
			col := 0
			for col < len(got[y]) && col < len(want[y]) && got[y][col] == want[y][col] {
				col++
			}
			t.Fatalf("%s row %d col %d: got %q want %q\n  got  %q\n  want %q",
				filename, y, col,
				cellAt(got[y], col), cellAt(want[y], col), got[y], want[y])
		}
	}
}

func cellAt(line string, col int) string {
	runes := []rune(line)
	if col < 0 || col >= len(runes) {
		return ""
	}
	return string(runes[col])
}

func TestGoldenDemoLive(t *testing.T) {
	for _, vp := range []struct{ cols, rows int }{{120, 32}, {181, 56}} {
		m := demoModel(vp.cols, vp.rows)
		compareGolden(t, "demo_live_"+strconv.Itoa(vp.cols)+"x"+strconv.Itoa(vp.rows)+".txt", screenLines(m))
	}
}

func TestGoldenDemoTab2(t *testing.T) {
	for _, vp := range []struct{ cols, rows int }{{120, 32}, {181, 56}} {
		m := demoModel(vp.cols, vp.rows)
		m.ws().active = 1
		compareGolden(t, "demo_tab2_"+strconv.Itoa(vp.cols)+"x"+strconv.Itoa(vp.rows)+".txt", screenLines(m))
	}
}

// TestSplitGeometry pins the original card geometry: sibling cards are
// adjacent (their own borders touch) and there is no extra separator
// column/row; the drag hit box is the b-side card's shared border.
func TestSplitGeometry(t *testing.T) {
	m := demoModel(120, 32)
	tab := m.activeTab()
	m.closePane(tab, tab.panes[1])
	runCmd(t, m, m.splitPane("row"))
	_, entries := m.paneRects(tab)
	var rects []rect
	for _, entry := range entries {
		if entry.pane != nil {
			rects = append(rects, entry.r)
		}
	}
	if len(rects) != 2 {
		t.Fatalf("rects=%v", rects)
	}
	if rects[0] != (rect{0, 1, 60, 30}) || rects[1] != (rect{60, 1, 60, 30}) {
		t.Fatalf("row split rects=%v, want [0..60) + [60..120)", rects)
	}
	sp := m.splitEntries(tab)
	if len(sp) != 1 || sp[0].boundary != (rect{60, 1, 1, 30}) {
		t.Fatalf("row boundary=%v", sp)
	}
	lines := screenLines(m)
	if cellAt(lines[1], 59) != "\u2510" || cellAt(lines[1], 60) != "\u250c" {
		t.Fatalf("card top corners must touch: %q", lines[1])
	}
	if cellAt(lines[2], 59) != "\u2502" || cellAt(lines[2], 60) != "\u2502" {
		t.Fatalf("card borders must be adjacent (││, no gutter): %q", lines[2])
	}
	if cellAt(lines[2], 61) != " " {
		t.Fatalf("no third vertical line: %q", lines[2])
	}

	// Column split: the cards stack directly, the hit box is the bottom
	// card's top border row.
	m2 := demoModel(120, 32)
	tab2 := m2.activeTab()
	m2.closePane(tab2, tab2.panes[1])
	runCmd(t, m2, m2.splitPane("col"))
	_, entries2 := m2.paneRects(tab2)
	rects = nil
	for _, entry := range entries2 {
		if entry.pane != nil {
			rects = append(rects, entry.r)
		}
	}
	if len(rects) != 2 || rects[0] != (rect{0, 1, 120, 15}) || rects[1] != (rect{0, 16, 120, 15}) {
		t.Fatalf("col split rects=%v", rects)
	}
	sp2 := m2.splitEntries(tab2)
	if len(sp2) != 1 || sp2[0].boundary != (rect{0, 16, 120, 1}) {
		t.Fatalf("col boundary=%v", sp2)
	}
	lines = screenLines(m2)
	if cellAt(lines[15], 0) != "\u2514" || cellAt(lines[16], 0) != "\u250c" {
		t.Fatalf("stacked card borders must touch: %q / %q", lines[15], lines[16])
	}
}

func TestGoldenFloatChrome(t *testing.T) {
	m := demoModel(120, 32)
	m.newFloating()
	compareGolden(t, "float_120x32.txt", screenLines(m))
	m.floatings[0].collapsed = true
	compareGolden(t, "float_collapsed_120x32.txt", screenLines(m))
}

// TestLeft1Right2Geometry: split row then split the right leaf column-wise;
// only the focused leaf is replaced, the left card keeps its rect.
func TestLeft1Right2Geometry(t *testing.T) {
	m := demoModel(120, 32)
	tab := m.activeTab()
	m.closePane(tab, tab.panes[1])
	runCmd(t, m, m.splitPane("row"))
	runCmd(t, m, m.splitPane("col"))
	_, entries := m.paneRects(tab)
	var rects []rect
	for _, entry := range entries {
		if entry.pane != nil {
			rects = append(rects, entry.r)
		}
	}
	want := []rect{{0, 1, 60, 30}, {60, 1, 60, 15}, {60, 16, 60, 15}}
	if len(rects) != len(want) {
		t.Fatalf("rects=%v", rects)
	}
	for i := range want {
		if rects[i] != want[i] {
			t.Fatalf("rects=%v, want %v", rects, want)
		}
	}
	lines := screenLines(m)
	// Left card stays full height; the right column splits at row 16.
	if cellAt(lines[1], 0) != "\u250c" || cellAt(lines[16], 60) != "\u250c" {
		t.Fatalf("left1right2 chrome: %q / %q", lines[1], lines[16])
	}
	if cellAt(lines[2], 119) != "\u2502" || cellAt(lines[2], 59) != "\u2502" {
		t.Fatalf("frame columns: %q", lines[2])
	}
}

// TestGoldenFooterScenes pins the 8-scene footer text table.
func TestGoldenFooterScenes(t *testing.T) {
	m := demoModel(120, 32)
	got := map[string]string{}
	got["LIVE"] = m.footerLine()
	m.mode = modePane
	got["PANE"] = m.footerLine()
	m.mode = modeResize
	got["RESIZE"] = m.footerLine()
	m.mode = modeTab
	got["TAB"] = m.footerLine()
	m.mode = modeSystem
	got["SYSTEM"] = m.footerLine()
	m.mode = modeLive
	m.overlay = overlayPicker
	got["PICKER"] = m.footerLine()
	m.overlay = overlayPrompt
	got["PROMPT"] = m.footerLine()
	m.overlay = overlayHelp
	got["HELP"] = m.footerLine()

	data, err := os.ReadFile(filepath.Join(pythonGoldenDir(), "v3_footer_120x32.txt"))
	if err != nil {
		t.Fatalf("read footer golden: %v", err)
	}
	want := map[string]string{}
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, rest, _ := strings.Cut(line, "|")
		want[name] = strings.TrimSuffix(rest, "|")
	}
	if len(want) != len(got) {
		t.Fatalf("footer scenes: got %d, want %d", len(got), len(want))
	}
	for name, line := range want {
		if got[name] != line {
			t.Fatalf("footer scene %s:\n  got  %q\n  want %q", name, got[name], line)
		}
	}
}

// TestGoldenFooterColors compares every footer run's text and style with the
// reference run table and pins the distinct action colors per scene.
func TestGoldenFooterColors(t *testing.T) {
	for _, vp := range []struct{ cols, rows int }{{120, 32}, {181, 56}} {
		filename := "v3_footer_colors_" + strconv.Itoa(vp.cols) + "x" + strconv.Itoa(vp.rows) + ".txt"
		data, err := os.ReadFile(filepath.Join(pythonGoldenDir(), filename))
		if err != nil {
			t.Fatalf("read %s: %v", filename, err)
		}
		type wantRun struct {
			name  string
			style string
			text  string
		}
		var want []wantRun
		for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) < 4 {
				t.Fatalf("%s: bad row %q", filename, line)
			}
			want = append(want, wantRun{name: parts[0], style: parts[2], text: parts[3]})
		}

		scenarios := []struct {
			name    string
			mode    string
			overlay string
		}{
			{"LIVE", modeLive, ""},
			{"PANE", modePane, ""},
			{"PICKER", modeLive, overlayPicker},
		}
		index := 0
		for _, sc := range scenarios {
			m := demoModel(vp.cols, vp.rows)
			m.mode = sc.mode
			m.overlay = sc.overlay
			runs, _ := m.footerRuns()
			runs = trimRuns(runs, vp.cols)
			actionStyles := map[string]bool{}
			for i, run := range runs {
				if index >= len(want) {
					t.Fatalf("%s: too many runs", filename)
				}
				expected := want[index]
				if expected.name != sc.name || expected.style != run.style || expected.text != run.text {
					t.Fatalf("%s run %d: got {%s %q %q} want {%s %q %q}",
						filename, index, sc.name, run.style, run.text,
						expected.name, expected.style, expected.text)
				}
				if i > 0 && run.style != stFooter {
					actionStyles[run.style] = true
				}
				index++
			}
			switch sc.name {
			case "LIVE":
				wantDistinct := 7
				if vp.cols >= 181 {
					wantDistinct = 9
					if !actionStyles[stFooterKeyCopy] {
						t.Fatalf("LIVE missing CLIPBOARD copy color")
					}
				}
				if len(actionStyles) != wantDistinct {
					t.Fatalf("LIVE distinct action colors %d, want %d", len(actionStyles), wantDistinct)
				}
			case "PANE":
				if len(actionStyles) != 5 {
					t.Fatalf("PANE distinct action colors %d, want 5", len(actionStyles))
				}
			case "PICKER":
				expected := map[string]bool{
					stFooterKeyFloat: true, stFooterKeyTab: true,
					stFooterKeyResize: true, stFooterAccent: true,
				}
				if len(actionStyles) != len(expected) {
					t.Fatalf("PICKER action colors %v, want %v", actionStyles, expected)
				}
				for style := range expected {
					if !actionStyles[style] {
						t.Fatalf("PICKER missing action color %q", style)
					}
				}
			}
		}
		if index != len(want) {
			t.Fatalf("%s: %d runs, want %d", filename, index, len(want))
		}
	}
}

// TestGoldenCapture1Txt uses the real 1.txt capture as an independent oracle
// for a 3-tab state (header, frame and footer rows must match literally).
func TestGoldenCapture1Txt(t *testing.T) {
	m := demoModel(181, 56)
	tab := m.activeTab()
	m.closePane(tab, tab.panes[1])
	m.tabSeq = 3
	m.ws().tabs = append(m.ws().tabs, makeTab("tab-3", "tab 3", []*pane{m.newPane("empty", nil)}, "row"))
	m.ws().active = 0

	data, err := os.ReadFile(filepath.Join("..", "python-shell", "1.txt"))
	if err != nil {
		t.Fatalf("read 1.txt: %v", err)
	}
	oracle := strings.Split(string(data), "\n")
	lines := screenLines(m)
	for _, y := range []int{0, 1, m.rows - 1} {
		if strings.TrimRight(lines[y], " ") != strings.TrimRight(oracle[y], " ") {
			t.Fatalf("1.txt row %d:\n  got  %q\n  want %q", y, strings.TrimRight(lines[y], " "), strings.TrimRight(oracle[y], " "))
		}
	}
}

// TestCardActions pins the card button ids and that the split button mutates
// the split tree (not just the chrome).
func TestCardActions(t *testing.T) {
	m := demoModel(120, 32)
	tab := m.activeTab()
	p := m.focusPane()
	runs := m.paneRuns(p, true, 120)
	ids := map[string]bool{}
	for _, run := range runs {
		if run.node != "" {
			ids[run.node] = true
		}
	}
	for _, action := range []string{"zoom", "split-v", "split-h", "close", "lock"} {
		node := "pane:" + p.id + ":" + action
		if !ids[node] {
			t.Fatalf("missing card action %s (ids=%v)", node, ids)
		}
	}
	before := len(tab.panes)
	m.onMouse(&pb.MouseEvent{Action: "press", Node: "pane:" + p.id + ":split-h"})
	if len(tab.panes) != before+1 {
		t.Fatalf("split-h button did not add a pane: %d -> %d", before, len(tab.panes))
	}
}
