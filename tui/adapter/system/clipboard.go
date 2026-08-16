// Package systemadapter 实现 TUI 对宿主系统能力的 port。
package systemadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/anytty/anytty/tui/port"
)

// ClipboardService 通过宿主系统命令实现剪贴板读写。
// 它只拥有当前进程最近一次成功提交的文本副本；系统剪贴板内容仍由宿主平台持有。
type ClipboardService struct {
	lastCopied string
}

const clipboardCommandTimeout = 1500 * time.Millisecond

// Read 从当前宿主平台的剪贴板读取文本；没有可用命令时明确失败，不回退到 TUI 状态。
func (service *ClipboardService) Read(ctx context.Context) (port.ClipboardReadResult, error) {
	readCtx, cancel := context.WithTimeout(ctx, clipboardCommandTimeout)
	defer cancel()
	text, err := readSystemClipboard(readCtx)
	if err != nil {
		return port.ClipboardReadResult{}, fmt.Errorf("read system clipboard: %w", err)
	}
	return port.ClipboardReadResult{Text: text}, nil
}

// Write 把文本写入当前宿主平台剪贴板；没有可用命令时明确失败。
func (service *ClipboardService) Write(ctx context.Context, req port.ClipboardWriteRequest) error {
	if req.Text == "" {
		service.lastCopied = ""
		return nil
	}
	writeCtx, cancel := context.WithTimeout(ctx, clipboardCommandTimeout)
	defer cancel()
	if err := writeSystemClipboard(writeCtx, req.Text); err != nil {
		return fmt.Errorf("write system clipboard: %w", err)
	}
	service.lastCopied = req.Text
	return nil
}

// LastCopy 返回该 adapter 最近一次接收的写入文本，用于 TUI 内部粘贴投影。
func (service *ClipboardService) LastCopy() string {
	return service.lastCopied
}
