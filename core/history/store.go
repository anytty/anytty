package history

import "context"

// HistoryStore 是权威 history query/copy 的外部契约。R436 后默认实现是 linehist：
// 写入通过 ApplyTransaction 消费 vterm eviction，查询按 frozen logical-line 边界分页。
type HistoryStore interface {
	LatestWindow(req HistoryWindowRequest) (HistoryWindow, error)
	OlderWindow(req HistoryWindowRequest) (HistoryWindow, error)
	OldestWindow(req HistoryWindowRequest) (HistoryWindow, error)
	NewerWindow(req HistoryWindowRequest) (HistoryWindow, error)
	Freeze(req FreezeHistoryRequest) (FrozenHistorySnapshot, error)
	Copy(req HistoryCopyRequest) (string, error)
	CopyChunk(context.Context, HistoryCopyChunkRequest) (HistoryCopyChunkResult, error)
	Search(context.Context, HistorySearchRequest) (HistorySearchResult, error)
	Release(token HistoryToken) error
}
