package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	clientendpoint "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func v3NewCommand(socket *string, logFile *string) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:  "new -- CMD [ARGS...]",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
			if err != nil {
				return err
			}
			defer closeLogger()
			if len(args) == 0 {
				args = defaultTerminalCommand()
			}
			socketPath := resolveV3Socket(*socket)
			logger.Info("creating core-v2 terminal", "socket", socketPath, "command", strings.Join(args, " "), "log_file", logPath)
			client, err := dialOrStartV3Client(socketPath, logPath, logger)
			if err != nil {
				return err
			}
			defer client.Close()
			terminalID := newV3TerminalID()
			if terminalName := strings.TrimSpace(name); terminalName != "" {
				// 中文说明：CLI first-party create 与 TUI 保持同一 identity 语义：
				// 用户名称就是 daemon-local terminal key，随机 ID 只服务无名称的兼容入口。
				terminalID = terminalName
			}
			application, err := newLocalApplicationSession(client)
			if err != nil {
				return err
			}
			created, err := application.TerminalCreate(context.Background(), &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{TerminalId: terminalID, Command: args, Name: name, Size: currentSize()}})
			if err != nil {
				logger.Error("create core-v2 terminal failed", "error", err)
				return err
			}
			createdID := created.GetTerminal().GetRef().GetTerminalId()
			logger.Info("created core-v2 terminal", "terminal_id", createdID)
			fmt.Fprintln(cmd.OutOrStdout(), createdID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "terminal name")
	return cmd
}

func newV3TerminalID() string {
	return "term-" + uuid.NewString()
}

func currentSize() *apipb.TerminalSize {
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || cols <= 0 || rows <= 0 {
		return &apipb.TerminalSize{Cols: 80, Rows: 24}
	}
	return &apipb.TerminalSize{Cols: uint32(cols), Rows: uint32(rows)}
}

func v3LsCommand(socket *string, logFile *string) *cobra.Command {
	return &cobra.Command{
		Use: "ls",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
			if err != nil {
				return err
			}
			defer closeLogger()
			socketPath := resolveV3Socket(*socket)
			client, err := dialOrStartV3Client(socketPath, logPath, logger)
			if err != nil {
				return err
			}
			defer client.Close()
			application, err := newLocalApplicationSession(client)
			if err != nil {
				return err
			}
			list, err := application.TerminalList(context.Background(), &apipb.TerminalListCommand{})
			if err != nil {
				logger.Error("list core-v2 terminals failed", "error", err)
				return err
			}
			logger.Info("listed core-v2 terminals", "count", len(list.Terminals))
			rows := make([][]string, 0, len(list.Terminals))
			for _, item := range list.Terminals {
				rows = append(rows, []string{
					item.GetRef().GetTerminalId(),
					item.GetName(),
					terminalStateString(item.GetState()),
					fmt.Sprintf("%dx%d", item.GetSize().GetCols(), item.GetSize().GetRows()),
					strings.Join(item.GetCommand(), " "),
				})
			}
			return writeCLITable(cmd.OutOrStdout(), []string{"ID", "NAME", "STATE", "SIZE", "COMMAND"}, rows)
		},
	}
}

func v3KillCommand(socket *string, logFile *string) *cobra.Command {
	return &cobra.Command{
		Use:  "kill <id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
			if err != nil {
				return err
			}
			defer closeLogger()
			socketPath := resolveV3Socket(*socket)
			logger.Info("killing core-v2 terminal", "terminal_id", args[0], "socket", socketPath, "log_file", logPath)
			client, err := dialOrStartV3Client(socketPath, logPath, logger)
			if err != nil {
				return err
			}
			defer client.Close()
			application, err := newLocalApplicationSession(client)
			if err != nil {
				return err
			}
			err = application.TerminalKill(context.Background(), &apipb.TerminalKillCommand{Terminal: &apipb.TerminalRef{EndpointId: string(clientendpoint.DefaultEndpointID), TerminalId: args[0]}})
			if err != nil {
				logger.Error("kill core-v2 terminal failed", "terminal_id", args[0], "error", err)
			}
			return err
		},
	}
}

func v3RemoveCommand(socket *string, logFile *string) *cobra.Command {
	return &cobra.Command{
		Use:     "rm <id>",
		Aliases: []string{"delete", "remove", "del"},
		Short:   "Delete a terminal from the core-v2 daemon inventory",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
			if err != nil {
				return err
			}
			defer closeLogger()
			socketPath := resolveV3Socket(*socket)
			logger.Info("removing core-v2 terminal", "terminal_id", args[0], "socket", socketPath, "log_file", logPath)
			client, err := dialOrStartV3Client(socketPath, logPath, logger)
			if err != nil {
				return err
			}
			defer client.Close()
			application, err := newLocalApplicationSession(client)
			if err != nil {
				return err
			}
			err = application.TerminalRemove(context.Background(), &apipb.TerminalRemoveCommand{Terminal: &apipb.TerminalRef{EndpointId: string(clientendpoint.DefaultEndpointID), TerminalId: args[0]}})
			if err != nil {
				logger.Error("remove core-v2 terminal failed", "terminal_id", args[0], "error", err)
			}
			return err
		},
	}
}
