package main

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/runtime"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// footerGoldenPath is the shared golden both the Go shell and the Python
// legacy.py recommended preset must match (M3). The format is
// "<SCENE>|<exact footer line>|" so leading/trailing spaces are visible.
const footerGoldenPath = "../../examples/python-shell/golden/recommended_footer_120x32.txt"

// footerGoldenScenes builds the canonical 120x32 recommended-profile state
// for every v2 footer scene. The states match legacy.py's recommended footer
// scenarios so both programs can be checked against the same golden.
func footerGoldenScenes(t *testing.T) map[string]*model {
	t.Helper()
	scenes := map[string]*model{}

	normal := newModel()
	normal.hello(&pb.Hello{Schema: 1, ViewId: "view:local:1", Epoch: 1, Cols: 120, Rows: 32})
	normal.mode = modeNormal
	scenes["NORMAL"] = normal

	normal2 := newModel()
	normal2.hello(&pb.Hello{Schema: 1, ViewId: "view:local:1", Epoch: 1, Cols: 120, Rows: 32})
	normal2.mode = modeNormal
	normal2.split("row")
	scenes["NORMAL_2PANE"] = normal2

	pane := newModel()
	pane.hello(&pb.Hello{Schema: 1, ViewId: "view:local:1", Epoch: 1, Cols: 120, Rows: 32})
	pane.split("row")
	pane.mode = modePane
	scenes["PANE"] = pane

	picker := newModel()
	picker.hello(&pb.Hello{Schema: 1, ViewId: "view:local:1", Epoch: 1, Cols: 120, Rows: 32})
	picker.mode = modePicker
	scenes["PICKER"] = picker

	prompt := newModel()
	prompt.hello(&pb.Hello{Schema: 1, ViewId: "view:local:1", Epoch: 1, Cols: 120, Rows: 32})
	prompt.mode = modePrompt
	scenes["PROMPT"] = prompt

	help := newModel()
	help.hello(&pb.Hello{Schema: 1, ViewId: "view:local:1", Epoch: 1, Cols: 120, Rows: 32})
	help.mode = modeHelp
	scenes["HELP"] = help

	scroll := newModel()
	scroll.hello(&pb.Hello{Schema: 1, ViewId: "view:local:1", Epoch: 1, Cols: 120, Rows: 32})
	scroll.mode = modeScroll
	scenes["SCROLL"] = scroll

	exited := newModel()
	exited.hello(&pb.Hello{Schema: 1, ViewId: "view:local:1", Epoch: 1, Cols: 120, Rows: 32})
	exited.sources = []*pb.Source{terminalSource("terminal:local:term-1", "term-1", true)}
	exited.focusSlot().sourceID = "terminal:local:term-1"
	exited.mode = modeNormal
	scenes["EXITED"] = exited

	return scenes
}

// footerFrameLine composes the real host pipeline (kernel solve + compositor)
// and returns the final footer row as plain text, spaces included.
func footerFrameLine(t *testing.T, m *model) string {
	t.Helper()
	root, keys := m.view()
	view := &pb.View{
		Epoch: 1,
		Rev:   1,
		Keys:  &pb.Keys{Claim: append([]string(nil), keys.Claim...), All: keys.All},
		Root:  root.Build(),
	}
	session := runtime.NewSession(runtime.Options{ViewID: m.viewID, Cols: m.cols, Rows: m.rows}, nil, nil)
	if err := session.HandleView(view); err != nil {
		t.Fatalf("HandleView: %v", err)
	}
	frame := session.ComposeFrame(nil, nil)
	y := frame.Rows() - 1
	var b strings.Builder
	for x := 0; x < frame.Cols(); x++ {
		b.WriteString(frame.CellAt(x, y).Text)
	}
	return b.String()
}

// TestRecommendedFooterGolden pins M3: under the default (recommended) config
// every v2 scene's final 120x32 footer row matches the shared golden byte for
// byte, the view tree and the composed frame agree, and no raw key/action
// name (spacebar, split-h, resize.layout_toggle, …) ever reaches the row.
func TestRecommendedFooterGolden(t *testing.T) {
	golden := readFooterGolden(t)
	scenes := footerGoldenScenes(t)

	names := make([]string, 0, len(scenes))
	for name := range scenes {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) != len(golden) {
		t.Fatalf("scenes = %v, golden has %d lines", names, len(golden))
	}

	banned := []string{"spacebar", "split-h", "split-v", "layout_toggle", "resize.", "floating", "zoom", "detach", "kill", "clipboard"}
	for _, name := range names {
		m := scenes[name]
		want, ok := golden[name]
		if !ok {
			t.Fatalf("golden misses scene %s", name)
		}
		line := footerFrameLine(t, m)
		if line != want {
			t.Errorf("%s footer mismatch:\n got %q\nwant %q", name, line, want)
		}
		if got := m.footerLine(); got != want {
			t.Errorf("%s footerLine mismatch:\n got %q\nwant %q", name, got, want)
		}
		lower := strings.ToLower(line)
		for _, token := range banned {
			if strings.Contains(lower, token) {
				t.Errorf("%s footer leaks %q: %q", name, token, line)
			}
		}
	}
}

func readFooterGolden(t *testing.T) map[string]string {
	t.Helper()
	data, err := os.ReadFile(filepath.FromSlash(footerGoldenPath))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	out := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, rest, ok := strings.Cut(line, "|")
		if !ok || !strings.HasSuffix(rest, "|") {
			t.Fatalf("malformed golden line %q", line)
		}
		out[name] = strings.TrimSuffix(rest, "|")
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan golden: %v", err)
	}
	return out
}
