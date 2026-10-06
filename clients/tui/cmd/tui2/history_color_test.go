package main

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/anytty/anytty/clients/tui/components/terminal"
	"github.com/anytty/anytty/clients/tui/history"
	"github.com/anytty/anytty/clients/tui/kernel"
	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

type coloredHistoryPTY struct {
	*fakePTY
	history.Backend
}

func (p *coloredHistoryPTY) HistoryBackend(context.Context) (history.Backend, *apipb.TerminalRef, error) {
	return p, &apipb.TerminalRef{TerminalId: "color"}, nil
}

func (p *coloredHistoryPTY) Window(context.Context, *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	result := &apipb.HistoryWindowResult{Token: "frozen"}
	for i := 1; i <= 3; i++ {
		result.Rows = append(result.Rows, &apipb.HistoryRow{LogicalLineId: uint64(i), Row: &apipb.ScreenRow{
			Cells: []*apipb.ScreenCell{
				{Content: "R", Width: 1, Style: &apipb.CellStyle{Foreground: "ansi:1"}},
				{Content: "界", Width: 2, Style: &apipb.CellStyle{Foreground: "idx:201", Background: "idx:17"}},
				{Content: "Z", Width: 1, Style: &apipb.CellStyle{Foreground: "#123456", Background: "#654321", Bold: true}},
			}, TailFill: &apipb.CellStyle{Background: "#112233"},
		}})
	}
	return result, nil
}

func (p *coloredHistoryPTY) Release(context.Context, *apipb.HistoryReleaseCommand) error { return nil }

// Exercise Host.placement: testing provider data alone cannot catch flattening
// styled history into plain text just before the terminal component renders it.
func TestHistoryPlacementPreservesColors(t *testing.T) {
	proc := &coloredHistoryPTY{fakePTY: newFakePTY()}
	handler := runtime.NewTerminalHandler(runtime.TerminalOptions{Cols: 12, Rows: 2, NewPTY: func(pty.Config) pty.PTY { return proc }})
	defer handler.Close()
	out, _ := handler.Handle(runtime.Request{Method: runtime.Method{Name: "terminal.attach"}, Params: &pb.MethodParams{Endpoint: "local", Id: "color"}})
	if !out.OK {
		t.Fatal(out.Error)
	}
	term, _ := handler.TerminalBySource(runtime.SourceID("local", "color"))
	session := runtime.NewSession(runtime.Options{Cols: 12, Rows: 2}, bytes.NewReader(nil), io.Discard)
	host := &Host{components: make(map[string]*terminal.Component), sizes: make(map[string][2]int)}
	box := &pb.Box{Id: "pane", Content: &pb.Content{Self: term.SourceID(), Props: map[string]string{"chrome.inset": "0"}}}
	if err := session.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: box}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := term.HistoryWindow(context.Background(), 0, 0); err != nil {
		t.Fatal(err)
	}
	for _, offset := range []int{0, 1} {
		if offset > 0 {
			if _, _, err := term.HistoryScroll(context.Background(), 1, 2); err != nil {
				t.Fatal(err)
			}
		}
		placement := host.placement(session, term.SourceID(), term, box, kernel.Rect{Width: 12, Height: 2})
		frame := render.NewFrame(12, 2)
		frame.Blit(placement.Lines...)
		for _, want := range []struct {
			col         int
			text, style string
		}{
			{0, "R", "fg:ansi:1"},
			{1, "界", "fg:idx:201;bg:idx:17"},
			{3, "Z", "fg:#123456;bg:#654321;bold"},
			{11, " ", "bg:#112233"},
		} {
			got := frame.CellAt(want.col, 0)
			if got.Text != want.text || string(got.Style) != want.style {
				t.Errorf("offset=%d col=%d: got %+v, want text=%q style=%q", offset, want.col, got, want.text, want.style)
			}
		}
		if !frame.CellAt(2, 0).Continuation {
			t.Fatal("wide character lost its continuation")
		}
		if got := term.VisibleLines()[0]; got != "R界Z" {
			t.Fatalf("display-only tail fill entered text: %q", got)
		}
	}
}
