package cli

import (
	"fmt"
	"strings"

	endpointdomain "github.com/anytty/anytty/access/engine/endpoint"
	"github.com/spf13/cobra"
)

func v3AttachCommand(socket *string, logFile *string) *cobra.Command {
	return &cobra.Command{
		Use:  "attach <id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLocalAttachCommand(cmd, args[0], *socket, *logFile, "")
		},
	}
}

func runLocalAttachCommand(cmd *cobra.Command, terminalID, socket, logFile, configPath string) error {
	return runAttachCommand(cmd, string(endpointdomain.DefaultEndpointID), terminalID, socket, logFile, configPath)
}

// runAttachCommand 校验 endpoint 后启动新 TUI，并把目标 terminal 通过
// -shell ... --attach 传给布局程序（缺失时退回 picker）。
func runAttachCommand(cmd *cobra.Command, endpointID, terminalID, socket, logFile, configPath string) error {
	if !isInteractiveTerminal() {
		return usageCLIError("anytty terminal attach requires an interactive terminal")
	}
	if err := rejectNestedTUI(); err != nil {
		return err
	}
	registry, err := loadV3ConnectionRegistry()
	if err != nil {
		return err
	}
	resolvedID := strings.TrimSpace(endpointID)
	if resolvedID == "" {
		resolvedID = string(registry.Default)
	}
	endpoint, ok := registry.Endpoints[endpointdomain.EndpointID(resolvedID)]
	if !ok {
		return &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", resolvedID)}
	}
	if !endpoint.Enabled {
		return &cliError{code: 4, message: fmt.Sprintf("endpoint %s is disabled", resolvedID)}
	}
	entry := tui2EntryConfig{
		EndpointID: resolvedID,
		TerminalID: terminalID,
		LogFile:    logFile,
		ConfigPath: configPath,
	}
	if explicit := strings.TrimSpace(socket); explicit != "" {
		entry.SocketOverride = resolveV3Socket(explicit)
		entry.LocalSocket = entry.SocketOverride
	} else if localSocket, ok, err := resolveTUI2LocalSocket(registry, resolvedID); err != nil {
		return err
	} else if ok {
		entry.LocalSocket = localSocket
	}
	return runTUI2(cmd.Context(), entry)
}
