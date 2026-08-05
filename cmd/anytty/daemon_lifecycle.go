package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anytty/anytty/shared/securefs"
	"github.com/spf13/cobra"
)

const daemonRecordSchemaVersion = 1

type daemonRuntimeRecord struct {
	SchemaVersion int       `json:"schema_version"`
	PID           int       `json:"pid"`
	ProcessID     string    `json:"process_identity"`
	InstanceToken string    `json:"instance_token"`
	Executable    string    `json:"executable"`
	SocketPath    string    `json:"socket_path"`
	LogPath       string    `json:"log_path"`
	ConfigPath    string    `json:"config_path,omitempty"`
	StartedAt     time.Time `json:"started_at"`
}

type daemonStatusView struct {
	State      string `json:"state"`
	PID        int    `json:"pid,omitempty"`
	SocketPath string `json:"socket_path"`
	LogPath    string `json:"log_path"`
	ConfigPath string `json:"config_path,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
}

func addDaemonLifecycleCommands(command *cobra.Command, socket, logFile, configPath *string, run func(*cobra.Command, []string) error) {
	command.AddCommand(&cobra.Command{Use: "run", Short: "Run the current-user daemon in the foreground", Args: cobra.NoArgs, RunE: run})
	command.AddCommand(newDaemonStartCommand(socket, logFile, configPath))
	command.AddCommand(newDaemonStopCommand(socket))
	command.AddCommand(newDaemonRestartCommand(socket, logFile, configPath))
	command.AddCommand(newDaemonStatusCommand(socket, logFile, configPath))
	command.AddCommand(newDaemonLogsCommand(logFile))
	command.AddCommand(newDaemonDoctorCommand(socket, logFile, configPath))
}

func openPrivateDaemonLog(path string) (*os.File, error) {
	parent := filepath.Dir(path)
	if err := ensurePrivateLogDirectory(parent); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	if err := securefs.SecureFile(path); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func acquireDaemonRuntimeRecord(socketPath, logPath, configPath string) (func(), error) {
	// record 由 daemon 进程自己原子占有，是 stop/restart 的唯一 PID truth；CLI 不扫描进程名。
	if !daemonLifecycleSupported() {
		return func() {}, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return nil, err
	}
	processID, err := daemonProcessIdentity(os.Getpid())
	if err != nil {
		return nil, fmt.Errorf("resolve daemon process identity: %w", err)
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	record := daemonRuntimeRecord{
		SchemaVersion: daemonRecordSchemaVersion, PID: os.Getpid(), ProcessID: processID,
		InstanceToken: hex.EncodeToString(tokenBytes), Executable: executable,
		SocketPath: socketPath, LogPath: logPath, ConfigPath: strings.TrimSpace(configPath),
		StartedAt: time.Now().UTC(),
	}
	path := daemonRecordPath(socketPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := securefs.SecureDirectory(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("secure daemon runtime directory: %w", err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		file, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if openErr == nil {
			if secureErr := securefs.SecureFile(path); secureErr != nil {
				_ = file.Close()
				_ = os.Remove(path)
				return nil, fmt.Errorf("secure daemon runtime record: %w", secureErr)
			}
			encoder := json.NewEncoder(file)
			encodeErr := encoder.Encode(record)
			closeErr := file.Close()
			if encodeErr != nil || closeErr != nil {
				_ = os.Remove(path)
				return nil, errors.Join(encodeErr, closeErr)
			}
			return func() { removeDaemonRecordIfOwned(path, record.InstanceToken) }, nil
		}
		if !errors.Is(openErr, os.ErrExist) {
			return nil, openErr
		}
		existing, readErr := readDaemonRuntimeRecord(path)
		if readErr == nil && daemonRecordProcessMatches(existing) {
			return nil, &cliError{code: 4, message: fmt.Sprintf("daemon for socket %s is already running", socketPath)}
		}
		if readErr == nil || errors.Is(readErr, os.ErrNotExist) {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			continue
		}
		return nil, readErr
	}
	return nil, &cliError{code: 4, message: "daemon runtime record is busy"}
}

func daemonRecordPath(socketPath string) string { return socketPath + ".daemon.json" }

func readDaemonRuntimeRecord(path string) (daemonRuntimeRecord, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return daemonRuntimeRecord{}, err
	}
	if !securefs.IsPrivateFile(path, info) {
		return daemonRuntimeRecord{}, &cliError{code: 5, message: "daemon runtime record owner or permissions are invalid"}
	}
	file, err := os.Open(path)
	if err != nil {
		return daemonRuntimeRecord{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64<<10))
	decoder.DisallowUnknownFields()
	var record daemonRuntimeRecord
	if err := decoder.Decode(&record); err != nil {
		return daemonRuntimeRecord{}, fmt.Errorf("decode daemon runtime record: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return daemonRuntimeRecord{}, fmt.Errorf("daemon runtime record has trailing data")
	}
	if record.SchemaVersion != daemonRecordSchemaVersion || record.PID <= 0 || record.ProcessID == "" || len(record.InstanceToken) != 64 || record.SocketPath == "" || record.Executable == "" || record.StartedAt.IsZero() {
		return daemonRuntimeRecord{}, fmt.Errorf("daemon runtime record metadata is invalid")
	}
	return record, nil
}

func daemonRecordProcessMatches(record daemonRuntimeRecord) bool {
	identity, err := daemonProcessIdentity(record.PID)
	return err == nil && identity == record.ProcessID
}

func removeDaemonRecordIfOwned(path, token string) {
	record, err := readDaemonRuntimeRecord(path)
	if err == nil && record.InstanceToken == token {
		_ = os.Remove(path)
	}
}

func daemonStatus(socketPath, fallbackLog, fallbackConfig string) (daemonStatusView, daemonRuntimeRecord, error) {
	// running 必须同时满足 record 进程身份未变化和 owning socket 完成 protocol Hello。
	view := daemonStatusView{State: "stopped", SocketPath: socketPath, LogPath: resolveV3LogFilePath(fallbackLog), ConfigPath: strings.TrimSpace(fallbackConfig)}
	record, err := readDaemonRuntimeRecord(daemonRecordPath(socketPath))
	if errors.Is(err, os.ErrNotExist) {
		return view, daemonRuntimeRecord{}, nil
	}
	if err != nil {
		return daemonStatusView{}, daemonRuntimeRecord{}, err
	}
	if !daemonRecordProcessMatches(record) {
		view.State = "stale"
		return view, record, nil
	}
	client, dialErr := v3DialClient(socketPath)
	if dialErr != nil {
		view.State = "starting"
	} else {
		_ = client.Close()
		view.State = "running"
	}
	view.PID = record.PID
	view.LogPath = record.LogPath
	view.ConfigPath = record.ConfigPath
	view.StartedAt = record.StartedAt.Format(time.RFC3339Nano)
	return view, record, nil
}

func newDaemonStartCommand(socket, logFile, configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "start", Short: "Start the current-user daemon service", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !daemonLifecycleSupported() {
				return &cliError{code: 6, message: "daemon service management is supported on Windows, macOS, and Linux"}
			}
			socketPath := resolveV3Socket(*socket)
			status, _, err := daemonStatus(socketPath, *logFile, *configPath)
			if err != nil {
				return err
			}
			if status.State == "running" || status.State == "starting" {
				return &cliError{code: 4, message: fmt.Sprintf("daemon is already %s", status.State)}
			}
			if status.State == "stale" {
				_ = os.Remove(daemonRecordPath(socketPath))
			}
			if err := startDetachedDaemon(socketPath, resolveV3LogFilePath(*logFile), strings.TrimSpace(*configPath)); err != nil {
				return classifyCLIError(err)
			}
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				status, _, err = daemonStatus(socketPath, *logFile, *configPath)
				if err == nil && status.State == "running" {
					if jsonOutput {
						return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Daemon running (pid %d)\n", status.PID)
					return nil
				}
				time.Sleep(25 * time.Millisecond)
			}
			return &cliError{code: 6, message: "daemon did not become ready"}
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newDaemonStopCommand(socket *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "stop", Short: "Stop the current-user daemon service", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			socketPath := resolveV3Socket(*socket)
			status, record, err := daemonStatus(socketPath, "", "")
			if err != nil {
				return err
			}
			if status.State == "stopped" || status.State == "stale" {
				return &cliError{code: 3, message: "daemon is not running"}
			}
			// Signal 只发送给已复验的精确 PID；record 不可信时绝不 fallback 到 pkill/名称匹配。
			if err := stopDaemonProcess(record.PID); err != nil {
				return classifyCLIError(err)
			}
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				if !daemonRecordProcessMatches(record) {
					_ = os.Remove(daemonRecordPath(socketPath))
					if jsonOutput {
						return json.NewEncoder(cmd.OutOrStdout()).Encode(daemonStatusView{State: "stopped", SocketPath: socketPath, LogPath: record.LogPath})
					}
					fmt.Fprintln(cmd.OutOrStdout(), "Daemon stopped")
					return nil
				}
				time.Sleep(25 * time.Millisecond)
			}
			return &cliError{code: 7, message: "timed out waiting for daemon to stop"}
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newDaemonRestartCommand(socket, logFile, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "restart", Short: "Restart the current-user daemon service", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, record, err := daemonStatus(resolveV3Socket(*socket), *logFile, *configPath)
			if err != nil {
				return err
			}
			if status.State == "running" || status.State == "starting" {
				if err := stopDaemonProcess(record.PID); err != nil {
					return classifyCLIError(err)
				}
				deadline := time.Now().Add(5 * time.Second)
				for daemonRecordProcessMatches(record) && time.Now().Before(deadline) {
					time.Sleep(25 * time.Millisecond)
				}
				if daemonRecordProcessMatches(record) {
					return &cliError{code: 7, message: "timed out waiting for daemon to stop"}
				}
				_ = os.Remove(daemonRecordPath(record.SocketPath))
			}
			if err := startDetachedDaemon(resolveV3Socket(*socket), resolveV3LogFilePath(*logFile), strings.TrimSpace(*configPath)); err != nil {
				return classifyCLIError(err)
			}
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				status, _, err = daemonStatus(resolveV3Socket(*socket), *logFile, *configPath)
				if err == nil && status.State == "running" {
					fmt.Fprintf(cmd.OutOrStdout(), "Daemon running (pid %d)\n", status.PID)
					return nil
				}
				time.Sleep(25 * time.Millisecond)
			}
			return &cliError{code: 6, message: "daemon did not become ready after restart"}
		},
	}
}

func newDaemonStatusCommand(socket, logFile, configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "status", Short: "Show current-user daemon service status", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, _, err := daemonStatus(resolveV3Socket(*socket), *logFile, *configPath)
			if err != nil {
				return err
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
			}
			fields := []cliField{{Label: "State", Value: status.State}}
			if status.PID > 0 {
				fields = append(fields, cliField{Label: "PID", Value: fmt.Sprintf("%d", status.PID)})
			}
			fields = append(fields,
				cliField{Label: "Socket", Value: status.SocketPath},
				cliField{Label: "Log", Value: status.LogPath},
			)
			return writeCLIFields(cmd.OutOrStdout(), fields...)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newDaemonLogsCommand(logFile *string) *cobra.Command {
	var follow bool
	var lines int
	command := &cobra.Command{
		Use: "logs", Short: "Read the current-user daemon log", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := resolveV3LogFilePath(*logFile)
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

func writeLogTail(writer io.Writer, path string, lines int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	parts := strings.Split(string(data), "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if lines > 0 && len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	if len(parts) > 0 {
		_, err = fmt.Fprintln(writer, strings.Join(parts, "\n"))
	}
	return err
}

func followLog(ctx context.Context, writer io.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadString('\n')
		if line != "" {
			if _, err := io.WriteString(writer, line); err != nil {
				return err
			}
		}
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func newDaemonDoctorCommand(socket, logFile, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "doctor", Short: "Check daemon runtime paths and ownership", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, _, err := daemonStatus(resolveV3Socket(*socket), *logFile, *configPath)
			if err != nil {
				return err
			}
			if err := writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "State", Value: status.State},
				cliField{Label: "Socket", Value: status.SocketPath},
				cliField{Label: "Record", Value: daemonRecordPath(status.SocketPath)},
				cliField{Label: "Log", Value: status.LogPath},
			); err != nil {
				return err
			}
			if status.State != "running" {
				return &cliError{code: 6, message: "daemon is not running"}
			}
			return nil
		},
	}
}
