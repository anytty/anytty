package history

import (
	"errors"
	"fmt"
)

// ErrHistoryInvalidMutation 表示 renderer 发出的 mutation batch 不满足
// HistoryStore 的领域约束。它只能用于新 renderer/store 边界，不能作为旧
// projector fallback 的兜底错误。
var ErrHistoryInvalidMutation = errors.New("invalid history mutation")

// ErrHistoryUnsupportedWindowMode 表示 history window 请求的分页方向不是
// authoritative store 支持的模式。调用方不能因此退回 TUI 本地 scrollback。
var ErrHistoryUnsupportedWindowMode = errors.New("unsupported history window mode")

// ErrHistoryWindowLimit 表示 limit 未落在公共与内部统一的 1..512 范围。
var ErrHistoryWindowLimit = errors.New("history window limit must be between 1 and 512")

// ErrHistoryWindowTooLarge 表示靠近请求 cursor 的单条 logical line 已超过窗口预算。
var ErrHistoryWindowTooLarge = errors.New("history window line exceeds response budget")

// ErrHistoryCopyTooLarge 是稳定、不可重试的 copy 上限错误；调用方不得返回部分文本。
var ErrHistoryCopyTooLarge = errors.New("history selection exceeds copy limit")

// ErrHistoryStaleWindow 表示 frozen token 不存在、不再有效或不属于当前 protocol session。
var ErrHistoryStaleWindow = errors.New("stale history window")

// ErrHistoryRendererNotImplemented 表示 R319 已清掉旧 projector/store，但新的
// logical renderer 尚未接入。这个错误用于防止旧错误模型继续对外提供历史。
var ErrHistoryRendererNotImplemented = errors.New("history logical renderer not implemented")

var ErrHistorySyncLost = errors.New("history crosses an output gap")

// SyncGapError identifies the durable logical-line boundary that a query tried
// to cross. Bytes on opposite sides were parsed by different VT parser epochs.
type SyncGapError struct {
	GapAfterLine LogicalLineID
}

func (err *SyncGapError) Error() string {
	if err == nil {
		return ErrHistorySyncLost.Error()
	}
	return fmt.Sprintf("history crosses output gap after logical line %d", err.GapAfterLine)
}

func (err *SyncGapError) Is(target error) bool { return target == ErrHistorySyncLost }
