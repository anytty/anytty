// Package runtimepath 提供本机 daemon 与客户端共享的运行时路径策略。
package runtimepath

import (
	"os"
	"path/filepath"
	"strings"
)

// SocketPath 返回当前用户的默认本地 socket 路径。
// XDG_RUNTIME_DIR 是显式 truth；否则使用系统临时目录下的用户专属子目录，
// 让 daemon 可以安全收紧目录权限，同时避免多用户争用同一 listener。
func SocketPath(name string) string {
	name = strings.TrimSpace(name)
	if runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeDir != "" {
		return filepath.Join(runtimeDir, name)
	}
	return filepath.Join(os.TempDir(), "anytty-"+userDiscriminator(), name)
}
