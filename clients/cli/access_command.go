package cli

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/anytty/anytty/access/accessrun"
	"github.com/anytty/anytty/access/direct"
	"github.com/anytty/anytty/access/gateway"
	"github.com/spf13/cobra"
)

// accessRunCommand 在前台运行 access 协议服务器（终端路由 + 文件 + 转发 + 鉴权）。
// 它组合 access/accessrun，与 anytty-access 二进制共享同一装配。
func accessRunCommand(socket, logFile *string) *cobra.Command {
	var route string
	var listenSpecs []string
	var allowValues []string
	var fileRoots []string
	var pairToken string
	var pairTokenFile string
	var accessSocket string
	var providerSocket string
	var transferDir string
	command := &cobra.Command{
		Use:   "run",
		Short: "Run the access protocol server in the foreground",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			socketPath := resolveV3Socket(*socket)
			listeners := make([]gateway.ListenerSpec, 0, len(listenSpecs))
			for _, spec := range listenSpecs {
				parsed, err := gateway.ParseListenerSpec(spec)
				if err != nil {
					return usageCLIError(err.Error())
				}
				listeners = append(listeners, parsed)
			}
			allow := append([]string(nil), allowValues...)
			if _, err := gateway.ParseNetworks(allow); err != nil {
				return usageCLIError(err.Error())
			}
			if route = strings.TrimSpace(route); route != "" {
				if err := direct.ValidateListenAddress(route); err != nil {
					return usageCLIError(err.Error())
				}
			}
			var token []byte
			tokenFrom := ""
			switch {
			case strings.TrimSpace(pairTokenFile) != "":
				read, err := accessrun.ReadTokenFile(pairTokenFile)
				if err != nil {
					return err
				}
				token = read
				tokenFrom = pairTokenFile
			case strings.TrimSpace(pairToken) != "":
				token = []byte(strings.TrimSpace(pairToken))
				tokenFrom = "command line"
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			runErr := accessrun.Run(ctx, accessrun.Options{
				Socket:         socketPath,
				AccessSocket:   accessSocket,
				ProviderSocket: providerSocket,
				Listeners:      listeners,
				Allow:          allow,
				PairToken:      token,
				PairTokenFrom:  tokenFrom,
				Route:          route,
				LogFile:        resolveV3LogFilePath(*logFile),
				FileRoots:      fileRoots,
				TransferDir:    transferDir,
			})
			if runErr != nil && ctx.Err() == nil {
				return runErr
			}
			return nil
		},
	}
	command.Flags().StringVar(&route, "route", "", "Direct listen HOST:PORT; wildcard hosts enable paired-device LAN discovery")
	command.Flags().StringArrayVar(&listenSpecs, "listen", nil, "listener spec, repeatable: tcp:HOST:PORT or unix:/path")
	command.Flags().StringArrayVar(&allowValues, "allow", nil, "TCP peer allow list entry (IP or CIDR), repeatable")
	command.Flags().StringArrayVar(&fileRoots, "file-root", nil, "allowed file root, repeatable; empty allows any absolute path")
	command.Flags().StringVar(&pairToken, "pair-token", "", "optional pair token (prefer --pair-token-file)")
	command.Flags().StringVar(&pairTokenFile, "pair-token-file", "", "optional file holding the pair token")
	command.Flags().StringVar(&accessSocket, "access-socket", "", "access client protocol listener path (default: topology default)")
	command.Flags().StringVar(&providerSocket, "provider-socket", "", "daemon terminal provider socket path (default: topology default)")
	command.Flags().StringVar(&transferDir, "transfer-dir", "", "durable file-transfer resume record directory")
	return command
}

// accessStatusCommand 报告 access 进程状态（与 daemon 进程分开）。
func accessStatusCommand(socket *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "status",
		Short: "Show access protocol server status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			view, _, err := currentAccessStatus(resolveV3Socket(*socket))
			if err != nil {
				return err
			}
			return writeAccessStatus(cmd.OutOrStdout(), view, jsonOutput)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

// accessLogsCommand 读取 access 进程日志；与 daemon 日志分开。
func accessLogsCommand() *cobra.Command {
	var follow bool
	var lines int
	command := &cobra.Command{
		Use:   "logs",
		Short: "Read the access protocol server log",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := accessLogsPath()
			if lines < 0 {
				return usageCLIError("--lines cannot be negative")
			}
			if err := writeLogTail(cmd.OutOrStdout(), path, lines); err != nil {
				return err
			}
			if !follow {
				return nil
			}
			return followLog(cmd.Context(), cmd.OutOrStdout(), path)
		},
	}
	command.Flags().IntVarP(&lines, "lines", "n", 100, "number of trailing lines")
	command.Flags().BoolVarP(&follow, "follow", "f", false, "follow appended log data")
	return command
}

// accessLogsPath 返回 access 日志路径；环境变量与默认路径分开于 daemon 日志。
func accessLogsPath() string {
	if path := strings.TrimSpace(os.Getenv("ANYTTY_ACCESS_LOG_FILE")); path != "" {
		return path
	}
	return accessrun.DefaultLogFile()
}
