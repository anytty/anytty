package endpoint

import (
	"context"
	"os"
	"strings"

	"github.com/anytty/anytty/proto/access/apipb"
)

// screenDeltaStarter 是 sessionConn 的可选扩展：attach 时声明了
// screen_stream 的会话实现它。服务端在 attachment channel 上推
// wire.TypeScreen 帧，RemotePTY 解码后交给 runtime 的共享 screenmodel
// （不再在这里把 cells 编码成 ANSI）。
type screenDeltaStarter interface {
	StartScreenDeltas(ctx context.Context, att *attachment, seed *apipb.NativeScreenResult) error
}

// tuiScreenTraceEnabled gates per-second screen-delta byte accounting.
func tuiScreenTraceEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ANYTTY_TUI_TRACE"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// screenDeltasEnabled 默认开启：TUI 走「服务端 vterm + 行级 delta」并由服务端
// 主动推送；设置 ANYTTY_TUI_SCREEN_DELTAS=0 可退回原始 PTY 字节流。
func screenDeltasEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ANYTTY_TUI_SCREEN_DELTAS"))) {
	case "0", "false", "no", "off":
		return false
	}
	return true
}
