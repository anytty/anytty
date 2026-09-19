package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// runV3RootCommand 是 `anytty` 的默认入口：加载 endpoint registry 后启动新
// TUI（tui2 宿主 + tui2-shell 布局程序）。
func runV3RootCommand(cmd *cobra.Command, socket string, logFile string, configPath string) error {
	if !isInteractiveTerminal() {
		return fmt.Errorf("anytty TUI requires an interactive terminal; use `anytty --help` or subcommands like `new`, `ls`, `attach`, `kill`, `rm`, `pool`")
	}
	if err := rejectNestedTUI(); err != nil {
		return err
	}
	connectionRegistry, err := loadV3ConnectionRegistry()
	if err != nil {
		return err
	}
	entry := tui2EntryConfig{LogFile: logFile, ConfigPath: configPath}
	if explicit := strings.TrimSpace(socket); explicit != "" {
		entry.SocketOverride = resolveV3Socket(explicit)
		entry.LocalSocket = entry.SocketOverride
	} else {
		localSocket, ok, err := resolveTUI2LocalSocket(connectionRegistry, string(connectionRegistry.Default))
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("default endpoint %q is not a local unix endpoint", connectionRegistry.Default)
		}
		entry.LocalSocket = localSocket
	}
	return runTUI2(cmd.Context(), entry)
}
