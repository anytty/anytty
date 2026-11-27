package endpoint

import (
	"context"
	"sync"
	"testing"
	"time"

	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	"google.golang.org/protobuf/proto"
)

func seedScreen(rows ...string) *apipb.NativeScreenResult {
	replacements := make([]*apipb.ScreenRowReplace, 0, len(rows))
	for index, line := range rows {
		cells := make([]*apipb.ScreenCell, 0, len(line))
		for _, r := range line {
			cells = append(cells, &apipb.ScreenCell{Content: string(r), Width: 1})
		}
		replacements = append(replacements, &apipb.ScreenRowReplace{
			RowIndex: int32(index),
			Row:      &apipb.ScreenRow{Cells: cells},
		})
	}
	return &apipb.NativeScreenResult{
		Size:            &apipb.TerminalSize{Cols: 10, Rows: uint32(len(rows))},
		LiveRevision:    1,
		FullReplace:     true,
		RowReplacements: replacements,
		Cursor:          &apipb.TerminalCursor{Row: 2, Col: 2, Visible: true, Shape: apipb.CursorShape_CURSOR_SHAPE_BLOCK},
	}
}

func deltaScreen(revision uint64, base uint64, rows []*apipb.ScreenRowReplace, copies []*apipb.ScreenRowCopy) *apipb.NativeScreenResult {
	return &apipb.NativeScreenResult{
		Size:            &apipb.TerminalSize{Cols: 10, Rows: 3},
		LiveRevision:    revision,
		BaseRevision:    base,
		RowReplacements: rows,
		RowCopies:       copies,
		Cursor:          &apipb.TerminalCursor{Row: 1, Col: int32(revision % 5), Visible: true, Shape: apipb.CursorShape_CURSOR_SHAPE_BLOCK},
	}
}

// TestRemotePTYForwardsPushedScreenFramesToModel 覆盖推流落地：TypeScreen 帧
// 解码后交给 runtime 安装的共享 screenmodel sink。
func TestRemotePTYForwardsPushedScreenFramesToModel(t *testing.T) {
	p := &RemotePTY{mgr: &Manager{baseCtx: context.Background()}, notify: make(chan struct{}, 1), closeCh: make(chan struct{}), doneCh: make(chan struct{})}
	p.cond = sync.NewCond(&p.mu)
	received := make(chan *apipb.NativeScreenResult, 1)
	p.SetScreenSink(func(screen *apipb.NativeScreenResult) {
		select {
		case received <- screen:
		default:
		}
	})

	seed := seedScreen("aa", "bb")
	raw, err := proto.Marshal(seed)
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	att := &attachment{stream: newStream()}
	p.pending = att
	go p.run()
	att.stream.push(wire.TypeScreen, internalprotocol.EncodeScreenStreamPayload(raw))

	select {
	case screen := <-received:
		if screen.GetLiveRevision() != seed.GetLiveRevision() || len(screen.GetRowReplacements()) != len(seed.GetRowReplacements()) {
			t.Fatalf("sink got %#v", screen)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pushed screen frame did not reach the model sink")
	}
	att.stream.close(nil)
	select {
	case <-p.doneCh:
	case <-time.After(time.Second):
		t.Fatal("remote pty pump did not stop")
	}
}
