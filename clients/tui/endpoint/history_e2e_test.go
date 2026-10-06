//go:build !windows

package endpoint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pooladapter "github.com/anytty/anytty/access/provider/pool"
	provider "github.com/anytty/anytty/access/provider/terminal"
	accessserver "github.com/anytty/anytty/access/server"
	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render"
	tuiruntime "github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/pool/core"
	poolwire "github.com/anytty/anytty/pool/provider"
	"github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// The output is produced BEFORE the UI attaches. The parser therefore never
// saw these lines: only provider-backed history can return the oldest marker.
func TestTerminalObjectReadsPersistentHistoryBeforeAttach(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	dir := t.TempDir()
	poolSocket, accessSocket := filepath.Join(dir, "p.sock"), filepath.Join(dir, "a.sock")
	pool := core.NewServer(core.WithSocketPath(poolSocket), core.WithHistoryStorageDir(filepath.Join(dir, "history")))
	defer pool.Shutdown(context.Background())
	server, err := poolwire.New(pool, poolwire.Config{Socket: poolSocket})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Shutdown(context.Background())
	go server.ListenAndServe(ctx)
	waitFor(t, 5*time.Second, func() bool { _, err := os.Stat(poolSocket); return err == nil }, "pool socket")
	access, err := accessserver.New(accessserver.Config{Socket: accessSocket, Provider: func(ctx context.Context) (provider.Provider, error) {
		return pooladapter.DialTerminal(ctx, poolSocket)
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer access.Close()
	go access.Serve(ctx)
	waitFor(t, 5*time.Second, func() bool { _, err := os.Stat(accessSocket); return err == nil }, "access socket")
	m := NewManager(Options{DialTimeout: 2 * time.Second, CallTimeout: 5 * time.Second, RegistryPath: filepath.Join(dir, "none.yaml")})
	defer m.Close()
	if err := m.Register(Config{Name: "persist", Kind: KindDaemon, Socket: accessSocket}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 5*time.Second, func() bool { return m.Health("persist") == HealthOK }, "endpoint ready")
	const id = "deep-history"
	wrappedLine := strings.Repeat("a", 79) + "界WRAP-END"
	_, err = m.Create(ctx, "persist", &apipb.TerminalCreateSpec{TerminalId: id,
		Command: []string{"/bin/sh", "-c", "printf '\\033[38;2;18;52;86;48;2;101;67;33;1mPERSIST-OLDEST\\033[0m\\n\\033[38;5;201;48;5;17m" + wrappedLine + "\\033[0m\\n'; i=0; while [ $i -lt 6200 ]; do printf 'history-%05d\\n' $i; i=$((i+1)); done; printf 'HISTORY-DONE\\n'; sleep 30"},
		Size:    &apipb.TerminalSize{Cols: 80, Rows: 12}})
	if err != nil {
		t.Fatal(err)
	}
	ref := &apipb.TerminalRef{EndpointId: "persist", TerminalId: id}
	waitFor(t, 15*time.Second, func() bool {
		result, err := m.Execute(ctx, "persist", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_LiveScreenNext{
			LiveScreenNext: &apipb.LiveScreenNextCommand{Terminal: ref}}})
		if err != nil {
			return false
		}
		for _, row := range result.GetLiveScreen().GetRowReplacements() {
			var text strings.Builder
			for _, cell := range row.GetRow().GetCells() {
				text.WriteString(cell.GetContent())
			}
			if strings.Contains(text.String(), "HISTORY-DONE") {
				return true
			}
		}
		return false
	}, "output completed before UI attach")
	handler := tuiruntime.NewTerminalHandler(tuiruntime.TerminalOptions{Cols: 80, Rows: 12,
		NewPTY: func(cfg pty.Config) pty.PTY { return m.NewRemotePTY(cfg) }})
	defer handler.Close()
	out, _ := handler.Handle(tuiruntime.Request{Method: tuiruntime.Method{Name: "terminal.attach"}, Params: &pb.MethodParams{Endpoint: "persist", Id: id}})
	if !out.OK {
		t.Fatal(out.Error)
	}
	term, ok := handler.TerminalBySource(tuiruntime.SourceID("persist", id))
	if !ok {
		t.Fatal("terminal object missing")
	}
	// Opening copy obtains a frozen provider token; a huge delta clamps at
	// the true oldest retained row, past both the old 1024 and 4096 caches.
	if _, _, err := term.HistoryWindow(ctx, 0, 0); err != nil {
		t.Fatal(err)
	}
	rows, offset, err := term.HistoryScroll(ctx, 1000000, 12)
	if err != nil {
		t.Fatal(err)
	}
	if offset < 6000 || !strings.Contains(strings.Join(rows, "\n"), "PERSIST-OLDEST") {
		t.Fatalf("offset=%d rows=%q; persistent rows before attach missing", offset, rows)
	}
	search, err := term.HistorySearch(ctx, "PERSIST-OLDEST", apipb.HistorySearchMode_HISTORY_SEARCH_MODE_TEXT, false, nil)
	if err != nil || !search.GetFound() {
		t.Fatalf("search=%v err=%v", search, err)
	}
	markerRow := -1
	for i, row := range rows {
		if row == "PERSIST-OLDEST" {
			markerRow = i
			break
		}
	}
	if markerRow < 0 {
		t.Fatalf("exact marker missing from rows: %q", rows)
	}
	for _, cell := range term.VisibleScreen().Line(markerRow) {
		style, ok := render.ParseStyle(string(cell.Style))
		if !ok || style.FG != "#123456" || style.BG != "#654321" || !style.Bold {
			t.Fatalf("persistent history lost RGB/bold: %+v", cell)
		}
	}
	text, err := term.HistoryCopy(ctx, &tuiruntime.CopySpec{Mode: "char", StartRow: markerRow, EndRow: markerRow, EndCol: len("PERSIST-OLDEST") - 1})
	if err != nil || text != "PERSIST-OLDEST" {
		t.Fatalf("copy=%q err=%v", text, err)
	}
	// Exercise the wire method used by the shell, including viewport mapping.
	for _, test := range []struct{ query, mode string }{
		{"^PERSIST-OLDEST$", "regex"}, {"PERSIST-OLDEST", "glob"}, {"history-00042", "text"},
	} {
		out, pending := handler.HandleContext(ctx, tuiruntime.Request{Method: tuiruntime.Method{Name: "terminal.search"},
			Params: &pb.MethodParams{Endpoint: "persist", Id: id, Query: test.query, SearchMode: test.mode}})
		if pending || !out.OK || !out.Data.GetFound() {
			t.Fatalf("%s search=%+v pending=%v", test.mode, out, pending)
		}
		if out.Data.GetOffset() < 6000 {
			t.Fatalf("search stayed near live: %v", out.Data)
		}
		start, end := int(out.Data.GetMatchStart()), int(out.Data.GetMatchEnd())
		copied, err := term.HistoryCopy(ctx, &tuiruntime.CopySpec{Mode: "char", StartRow: start / 80, StartCol: start % 80, EndRow: (end - 1) / 80, EndCol: (end - 1) % 80})
		want := "PERSIST-OLDEST"
		if test.mode == "text" {
			want = test.query
		}
		if err != nil || copied != want {
			t.Fatalf("search selection copied %q want %q err=%v", copied, want, err)
		}
	}
	wrapped, err := term.Search(ctx, "^a{79}界WRAP-END$", "regex", false, 0)
	if err != nil || !wrapped.GetFound() {
		t.Fatalf("wrapped search=%v err=%v", wrapped, err)
	}
	begin, end := int(wrapped.GetMatchStart()), int(wrapped.GetMatchEnd())
	if end%80 != 10 {
		t.Fatalf("wide wrap end column=%d want 10; rows=%q", end%80, wrapped.GetRows())
	}
	for row := begin / 80; row <= (end-1)/80; row++ {
		for _, cell := range term.VisibleScreen().Line(row) {
			style, ok := render.ParseStyle(string(cell.Style))
			if !ok || style.FG != "idx:201" || style.BG != "idx:17" {
				t.Fatalf("wrapped history lost palette colors at row %d: %+v", row, cell)
			}
		}
	}
	copied, err := term.HistoryCopy(ctx, &tuiruntime.CopySpec{Mode: "char", StartRow: begin / 80, StartCol: begin % 80, EndRow: (end - 1) / 80, EndCol: (end - 1) % 80})
	if err != nil || copied != wrappedLine {
		t.Fatalf("wrapped copy=%q err=%v", copied, err)
	}
	if err := term.HistoryRelease(ctx); err != nil {
		t.Fatal(err)
	}
	if term.HistoryActive() {
		t.Fatal("release did not restore live")
	}
}
