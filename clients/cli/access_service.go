package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// accessStatusView 是 access 角色的独立状态视图。
type accessStatusView struct {
	State      string `json:"state"`
	PID        int    `json:"pid,omitempty"`
	SocketPath string `json:"socket_path"`
	LogPath    string `json:"log_path"`
}

// currentAccessStatus 从 daemon runtime record 投影 access 角色状态。
// record 缺失时回退为一次 socket 探测；access 独立于 daemon 生命周期。
func currentAccessStatus(socketPath string) (accessStatusView, daemonRuntimeRecord, error) {
	view := accessStatusView{State: "stopped", SocketPath: accessHealthSocket(socketPath), LogPath: accessLogPath()}
	record, err := readDaemonRuntimeRecord(daemonRecordPath(socketPath))
	switch {
	case err == nil:
		view.State = accessProcessState(record)
		if view.State != "stale" && view.State != "stopped" {
			view.PID = record.AccessPID
		}
		if record.AccessLogPath != "" {
			view.LogPath = record.AccessLogPath
		}
		return view, record, nil
	case os.IsNotExist(err):
		if probeErr := probeV3Socket(accessHealthSocket(socketPath)); probeErr == nil {
			view.State = "running"
		}
		return view, daemonRuntimeRecord{}, nil
	default:
		return accessStatusView{}, daemonRuntimeRecord{}, err
	}
}

func writeAccessStatus(out io.Writer, view accessStatusView, jsonOutput bool) error {
	if jsonOutput {
		return json.NewEncoder(out).Encode(view)
	}
	fields := []cliField{{Label: "State", Value: view.State}}
	if view.PID > 0 {
		fields = append(fields, cliField{Label: "PID", Value: fmt.Sprintf("%d", view.PID)})
	}
	fields = append(fields,
		cliField{Label: "Socket", Value: view.SocketPath},
		cliField{Label: "Log", Value: view.LogPath},
	)
	return writeCLIFields(out, fields...)
}

// accessStartCommand 只启动 access 角色；daemon 必须已经在运行。
func accessStartCommand(socket *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "start", Short: "Start the access process for the running daemon", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			socketPath := resolveV3Socket(*socket)
			status, _, err := daemonStatus(socketPath, "", "")
			if err != nil {
				return err
			}
			if status.State != "running" {
				return &cliError{code: 3, message: "daemon is not running; run `anytty daemon start` first"}
			}
			view, record, err := currentAccessStatus(socketPath)
			if err != nil {
				return err
			}
			if view.State == "running" {
				return &cliError{code: 4, message: fmt.Sprintf("access is already running (pid %d)", record.AccessPID)}
			}
			if err := startManagedAccess(socketPath); err != nil {
				return classifyCLIError(err)
			}
			view, _, err = currentAccessStatus(socketPath)
			if err != nil {
				return err
			}
			if view.State != "running" {
				return &cliError{code: 6, message: "access did not become ready"}
			}
			return writeAccessStatus(cmd.OutOrStdout(), view, jsonOutput)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

// accessStopCommand 只停止 access 角色；daemon 与终端继续运行。
func accessStopCommand(socket *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "stop", Short: "Stop the access process; daemon and terminals keep running", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			socketPath := resolveV3Socket(*socket)
			view, record, err := currentAccessStatus(socketPath)
			if err != nil {
				return err
			}
			if view.State != "running" && view.State != "starting" {
				return &cliError{code: 3, message: "access is not running"}
			}
			if err := stopManagedAccess(record); err != nil {
				return classifyCLIError(err)
			}
			view, _, err = currentAccessStatus(socketPath)
			if err != nil {
				return err
			}
			if !jsonOutput {
				fmt.Fprintln(cmd.OutOrStdout(), "Access stopped; daemon and terminals keep running")
			}
			return writeAccessStatus(cmd.OutOrStdout(), view, jsonOutput)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

// accessRestartCommand 只重启 access 角色：这是 access 独立升级入口。
// daemon PID 与终端记录保持不变，客户端重新连接即可。
func accessRestartCommand(socket *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "restart", Short: "Restart the access process without touching daemon or terminals", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			socketPath := resolveV3Socket(*socket)
			_, record, err := currentAccessStatus(socketPath)
			if err != nil {
				return err
			}
			if state := accessProcessState(record); state == "running" || state == "starting" {
				if err := stopManagedAccess(record); err != nil {
					return classifyCLIError(err)
				}
			}
			status, _, err := daemonStatus(socketPath, "", "")
			if err != nil {
				return err
			}
			if status.State != "running" {
				return &cliError{code: 3, message: "daemon is not running; run `anytty daemon start` first"}
			}
			if err := startManagedAccess(socketPath); err != nil {
				return classifyCLIError(err)
			}
			view, _, err := currentAccessStatus(socketPath)
			if err != nil {
				return err
			}
			if view.State != "running" {
				return &cliError{code: 6, message: "access did not become ready"}
			}
			return writeAccessStatus(cmd.OutOrStdout(), view, jsonOutput)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}
