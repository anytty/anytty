package core

import (
	"time"

	vterm "github.com/anytty/anytty/vterm/vterm"
)

// LiveRevision 是 core-v2 为单个 terminal native screen 维护的单调版本。
// 它只描述 live display 投影版本，不是 logical-line history generation，也不能作为 history window stale guard。
type LiveRevision uint64

// NativeScreenSize 是 core-v2 live native screen 的尺寸投影。
// truth 来源是 terminal 当前 live SurfaceTrack/vterm size，不来自 TUI pane 或 renderer frame。
type NativeScreenSize struct {
	Cols int
	Rows int
}

// NativeScreenRow 是 core-v2 对外暴露的 native screen 单行 cell matrix。
// 这里保留 vterm cell 语义属性，protocol/TUI 只能把它当实时屏幕 projection，不能当 history truth。
type NativeScreenRow struct {
	Index   int
	Cells   []vterm.Cell
	Wrapped bool
}

// NativeScreenRowCopy reuses exact rows from the client's base screen. All
// copies are evaluated from the unchanged base before replacement rows apply.
type NativeScreenRowCopy struct {
	SourceRow      int
	DestinationRow int
	Count          int
}

// NativeScreenSnapshot 是 core-v2 当前 native screen 的 latest-only 快照。
// 它由 Terminal 从 live SurfaceTrack 读取；不携带 scrollback、history token 或 TUI view 状态。
type NativeScreenSnapshot struct {
	TerminalID   string
	BaseRevision LiveRevision
	Revision     LiveRevision
	FullReplace  bool
	Size         NativeScreenSize
	RowCopies    []NativeScreenRowCopy
	Rows         []NativeScreenRow
	Cursor       vterm.CursorState
	Modes        vterm.TerminalModes
	AltScreen    bool
	Timestamp    time.Time
}

// nativeScreenBaseline is a compact, session-owned description of one frame
// the client may have merged. It intentionally stores row fingerprints instead
// of another cell matrix; the current screen remains owned by Terminal.
type nativeScreenBaseline struct {
	terminal   *Terminal
	revision   LiveRevision
	generation uint64
	size       NativeScreenSize
	rowHashes  []uint64
	altScreen  bool
}

func nativeScreenDeltaRows(base *nativeScreenBaseline, current []uint64) ([]NativeScreenRowCopy, []int, bool) {
	if base == nil || len(base.rowHashes) != len(current) {
		return nil, nil, false
	}
	sourcesByHash := make(map[uint64]int, len(base.rowHashes))
	for source, hash := range base.rowHashes {
		if _, exists := sourcesByHash[hash]; !exists {
			sourcesByHash[hash] = source
		}
	}
	sources := make([]int, len(current))
	for destination, hash := range current {
		sources[destination] = -1
		if base.rowHashes[destination] == hash {
			sources[destination] = destination
			continue
		}
		if source, ok := sourcesByHash[hash]; ok {
			sources[destination] = source
		}
	}
	var copies []NativeScreenRowCopy
	var replacements []int
	for destination, source := range sources {
		if source < 0 {
			replacements = append(replacements, destination)
			continue
		}
		if source == destination {
			continue
		}
		if len(copies) > 0 {
			last := &copies[len(copies)-1]
			if last.SourceRow+last.Count == source && last.DestinationRow+last.Count == destination {
				last.Count++
				continue
			}
		}
		copies = append(copies, NativeScreenRowCopy{SourceRow: source, DestinationRow: destination, Count: 1})
	}
	return copies, replacements, true
}

// LiveScreenInvalidated 是 core 发给客户端的 live screen 唤醒信号。
// 它只表示 terminal 的 latest native screen 至少已经变到 Revision；客户端应自行拉取最新 snapshot。
type LiveScreenInvalidated struct {
	TerminalID string
	Revision   LiveRevision
}
