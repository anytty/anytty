package port

import (
	"context"
)

// ClipboardService 是 TUI effect 访问宿主剪贴板的 application port。
type ClipboardService interface {
	Read(context.Context) (ClipboardReadResult, error)
	Write(context.Context, ClipboardWriteRequest) error
	LastCopy() string
}

// ClipboardReadResult 是宿主剪贴板读取结果，不代表 reducer clipboard history 已持久化。
type ClipboardReadResult struct {
	Text string
}

// ClipboardWriteRequest 描述一次显式宿主剪贴板写入。
type ClipboardWriteRequest struct {
	Text string
}
