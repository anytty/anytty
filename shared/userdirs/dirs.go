// Package userdirs 提供 CLI、TUI 和 daemon 共享的用户级配置与状态目录策略。
package userdirs

import (
	"os"
	"path/filepath"
	"strings"
)

// ConfigHome 返回当前用户配置根目录。
// XDG_CONFIG_HOME 是跨平台显式覆盖；未覆盖时由目标操作系统决定路径，调用方只在其下追加 anytty 领域目录。
func ConfigHome() string {
	if path := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); path != "" {
		return filepath.Clean(path)
	}
	return platformConfigHome()
}

// StateHome 返回当前用户持久运行状态根目录。
// XDG_STATE_HOME 是显式真值；Windows 默认使用 LocalAppData，Unix 保持 ~/.local/state，账号目录不可解析时才使用临时目录。
func StateHome() string {
	if path := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); path != "" {
		return filepath.Clean(path)
	}
	return platformStateHome()
}
