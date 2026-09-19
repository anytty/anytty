package cli

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

const poolRecordSchemaVersion = 1

type poolRuntimeRecord struct {
	SchemaVersion int       `json:"schema_version"`
	PID           int       `json:"pid"`
	ProcessID     string    `json:"process_identity"`
	InstanceToken string    `json:"instance_token"`
	Executable    string    `json:"executable"`
	SocketPath    string    `json:"socket_path"`
	LogPath       string    `json:"log_path"`
	ConfigPath    string    `json:"config_path,omitempty"`
	DirectListen  string    `json:"direct_listen,omitempty"`
	StartedAt     time.Time `json:"started_at"`

	// Access 字段登记同一 lifecycle 管理的 access 子进程；旧 record 缺省为空。
	AccessPID           int    `json:"access_pid,omitempty"`
	AccessProcessID     string `json:"access_process_identity,omitempty"`
	AccessLogPath       string `json:"access_log_path,omitempty"`
	AccessInstanceToken string `json:"access_instance_token,omitempty"`
}

type poolStatusView struct {
	State         string `json:"state"`
	PID           int    `json:"pid,omitempty"`
	SocketPath    string `json:"socket_path"`
	LogPath       string `json:"log_path"`
	ConfigPath    string `json:"config_path,omitempty"`
	DirectListen  string `json:"direct_listen,omitempty"`
	StartedAt     string `json:"started_at,omitempty"`
	AccessState   string `json:"access_state,omitempty"`
	AccessPID     int    `json:"access_pid,omitempty"`
	AccessLogPath string `json:"access_log_path,omitempty"`
}

func addPoolLifecycleCommands(command *cobra.Command, socket, logFile, configPath *string, run func(*cobra.Command, []string) error) {
	command.AddCommand(&cobra.Command{Use: "run", Short: "Run the current-user terminal pool in the foreground", Args: cobra.NoArgs, RunE: run})
	command.AddCommand(newPoolStartCommand(socket, logFile, configPath))
	command.AddCommand(newPoolStopCommand(socket))
	command.AddCommand(newPoolRestartCommand(socket, logFile, configPath))
	command.AddCommand(newPoolStatusCommand(socket, logFile, configPath))
	command.AddCommand(newPoolLogsCommand(logFile))
	command.AddCommand(newPoolDoctorCommand(socket, logFile, configPath))
}

func openPrivatePoolLog(path string) (*os.File, error) {
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

func acquirePoolRuntimeRecord(socketPath, logPath, configPath string) (func(), error) {
	// record 由 pool 进程自己原子占有，是 stop/restart 的唯一 PID truth；CLI 不扫描进程名。
	if !poolLifecycleSupported() {
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
	processID, err := poolProcessIdentity(os.Getpid())
	if err != nil {
		return nil, fmt.Errorf("resolve pool process identity: %w", err)
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	record := poolRuntimeRecord{
		SchemaVersion: poolRecordSchemaVersion, PID: os.Getpid(), ProcessID: processID,
		InstanceToken: hex.EncodeToString(tokenBytes), Executable: executable,
		SocketPath: socketPath, LogPath: logPath, ConfigPath: strings.TrimSpace(configPath),
		DirectListen: strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_LISTEN")),
		StartedAt:    time.Now().UTC(),
	}
	path := poolRecordPath(socketPath)
	if legacyRecord, legacyErr := readPoolRuntimeRecord(legacyPoolRecordPath(socketPath)); legacyErr == nil {
		if poolRecordProcessMatches(legacyRecord) {
			return nil, &cliError{code: 4, message: fmt.Sprintf("terminal pool for socket %s is already running", socketPath)}
		}
		_ = os.Remove(legacyPoolRecordPath(socketPath))
	} else if !errors.Is(legacyErr, os.ErrNotExist) {
		return nil, legacyErr
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := securefs.SecureDirectory(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("secure pool runtime directory: %w", err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		file, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if openErr == nil {
			if secureErr := securefs.SecureFile(path); secureErr != nil {
				_ = file.Close()
				_ = os.Remove(path)
				return nil, fmt.Errorf("secure pool runtime record: %w", secureErr)
			}
			encoder := json.NewEncoder(file)
			encodeErr := encoder.Encode(record)
			closeErr := file.Close()
			if encodeErr != nil || closeErr != nil {
				_ = os.Remove(path)
				return nil, errors.Join(encodeErr, closeErr)
			}
			return func() { removePoolRecordIfOwned(path, record.InstanceToken) }, nil
		}
		if !errors.Is(openErr, os.ErrExist) {
			return nil, openErr
		}
		existing, readErr := readPoolRuntimeRecord(path)
		if readErr == nil && poolRecordProcessMatches(existing) {
			return nil, &cliError{code: 4, message: fmt.Sprintf("terminal pool for socket %s is already running", socketPath)}
		}
		if readErr == nil || errors.Is(readErr, os.ErrNotExist) {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			continue
		}
		return nil, readErr
	}
	return nil, &cliError{code: 4, message: "terminal pool runtime record is busy"}
}

// poolRecordPath 是新 pool runtime record 的路径。
func poolRecordPath(socketPath string) string { return socketPath + ".pool.json" }

// legacyPoolRecordPath 是升级前 daemon runtime record 的路径；只读回退，不再新写。
func legacyPoolRecordPath(socketPath string) string { return socketPath + ".daemon.json" }

// effectivePoolRecordPath 返回当前生效的 record：新的 pool record 优先，缺失时
// 回退旧 daemon record，让升级窗口里的 stop/status/pair 仍能命中真实进程。
func effectivePoolRecordPath(socketPath string) string {
	if _, err := os.Lstat(poolRecordPath(socketPath)); err == nil {
		return poolRecordPath(socketPath)
	}
	if _, err := os.Lstat(legacyPoolRecordPath(socketPath)); err == nil {
		return legacyPoolRecordPath(socketPath)
	}
	return poolRecordPath(socketPath)
}

// readPoolRuntimeRecordForSocket 优先读取新的 pool record，缺失时回退旧
// daemon record；旧 record 缺省时返回 os.ErrNotExist。
func readPoolRuntimeRecordForSocket(socketPath string) (poolRuntimeRecord, string, error) {
	path := poolRecordPath(socketPath)
	record, err := readPoolRuntimeRecord(path)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return record, path, err
	}
	legacy := legacyPoolRecordPath(socketPath)
	record, err = readPoolRuntimeRecord(legacy)
	if err != nil {
		return poolRuntimeRecord{}, legacy, err
	}
	return record, legacy, nil
}

// removePoolRecord removes both the current and the legacy record for a socket.
func removePoolRecord(socketPath string) {
	_ = os.Remove(poolRecordPath(socketPath))
	_ = os.Remove(legacyPoolRecordPath(socketPath))
}

func readPoolRuntimeRecord(path string) (poolRuntimeRecord, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return poolRuntimeRecord{}, err
	}
	if !securefs.IsPrivateFile(path, info) {
		return poolRuntimeRecord{}, &cliError{code: 5, message: "pool runtime record owner or permissions are invalid"}
	}
	file, err := os.Open(path)
	if err != nil {
		return poolRuntimeRecord{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64<<10))
	decoder.DisallowUnknownFields()
	var record poolRuntimeRecord
	if err := decoder.Decode(&record); err != nil {
		return poolRuntimeRecord{}, fmt.Errorf("decode pool runtime record: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return poolRuntimeRecord{}, fmt.Errorf("pool runtime record has trailing data")
	}
	if record.SchemaVersion != poolRecordSchemaVersion || record.PID <= 0 || record.ProcessID == "" || len(record.InstanceToken) != 64 || record.SocketPath == "" || record.Executable == "" || record.StartedAt.IsZero() {
		return poolRuntimeRecord{}, fmt.Errorf("pool runtime record metadata is invalid")
	}
	if record.AccessPID != 0 && (record.AccessProcessID == "" || record.AccessPID <= 0) {
		return poolRuntimeRecord{}, fmt.Errorf("pool runtime record access metadata is invalid")
	}
	return record, nil
}

func poolRecordProcessMatches(record poolRuntimeRecord) bool {
	identity, err := poolProcessIdentity(record.PID)
	return err == nil && identity == record.ProcessID
}

func removePoolRecordIfOwned(path, token string) {
	record, err := readPoolRuntimeRecord(path)
	if err == nil && record.InstanceToken == token {
		_ = os.Remove(path)
	}
}

func poolStatus(socketPath, fallbackLog, fallbackConfig string) (poolStatusView, poolRuntimeRecord, error) {
	// running 必须同时满足 record 进程身份未变化和 owning socket 完成 protocol Hello。
	view := poolStatusView{State: "stopped", SocketPath: socketPath, LogPath: resolveV3LogFilePath(fallbackLog), ConfigPath: strings.TrimSpace(fallbackConfig)}
	record, _, err := readPoolRuntimeRecordForSocket(socketPath)
	if errors.Is(err, os.ErrNotExist) {
		return view, poolRuntimeRecord{}, nil
	}
	if err != nil {
		return poolStatusView{}, poolRuntimeRecord{}, err
	}
	if !poolRecordProcessMatches(record) {
		view.State = "stale"
		return view, record, nil
	}
	if dialErr := probeProviderSocket(poolHealthSocket(socketPath)); dialErr != nil {
		view.State = "starting"
	} else {
		view.State = "running"
	}
	view.PID = record.PID
	view.LogPath = record.LogPath
	view.ConfigPath = record.ConfigPath
	view.DirectListen = record.DirectListen
	view.StartedAt = record.StartedAt.Format(time.RFC3339Nano)
	view.AccessState = accessProcessState(record)
	if record.AccessPID > 0 && view.AccessState != "stale" && view.AccessState != "stopped" {
		view.AccessPID = record.AccessPID
	}
	view.AccessLogPath = record.AccessLogPath
	return view, record, nil
}

func newPoolStartCommand(socket, logFile, configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "start", Short: "Start the current-user terminal pool service", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !poolLifecycleSupported() {
				return &cliError{code: 6, message: "terminal pool service management is supported on Windows, macOS, and Linux"}
			}
			socketPath := resolveV3Socket(*socket)
			status, _, err := poolStatus(socketPath, *logFile, *configPath)
			if err != nil {
				return err
			}
			if status.State == "running" || status.State == "starting" {
				return &cliError{code: 4, message: fmt.Sprintf("terminal pool is already %s", status.State)}
			}
			if status.State == "stale" {
				removePoolRecord(socketPath)
			}
			if err := startDetachedPool(socketPath, resolveV3LogFilePath(*logFile), strings.TrimSpace(*configPath)); err != nil {
				return classifyCLIError(err)
			}
			deadline := time.Now().Add(5 * time.Second)
			poolReady := false
			for time.Now().Before(deadline) {
				status, _, err = poolStatus(socketPath, *logFile, *configPath)
				if err == nil && status.State == "running" {
					poolReady = true
					break
				}
				time.Sleep(25 * time.Millisecond)
			}
			if !poolReady {
				return &cliError{code: 6, message: "terminal pool did not become ready"}
			}
			// pool 先起，access 后起；access 失败必须回滚 pool，避免半栈。
			if err := startManagedAccess(socketPath); err != nil {
				if status.PID > 0 {
					_ = stopPoolProcess(status.PID)
				}
				return classifyCLIError(err)
			}
			status, _, err = poolStatus(socketPath, *logFile, *configPath)
			if err != nil {
				return err
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Terminal pool running (pid %d), access running (pid %d)\n", status.PID, status.AccessPID)
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newPoolStopCommand(socket *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "stop", Short: "Stop the current-user terminal pool service", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			socketPath := resolveV3Socket(*socket)
			status, record, err := poolStatus(socketPath, "", "")
			if err != nil {
				return err
			}
			if status.State == "stopped" || status.State == "stale" {
				return &cliError{code: 3, message: "terminal pool is not running"}
			}
			// access 先停；它只监听内部/客户端入口，不持有终端 truth。
			if err := stopManagedAccess(record); err != nil {
				return classifyCLIError(err)
			}
			// Signal 只发送给已复验的精确 PID；record 不可信时绝不 fallback 到 pkill/名称匹配。
			if err := stopPoolProcess(record.PID); err != nil {
				return classifyCLIError(err)
			}
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				if !poolRecordProcessMatches(record) {
					removePoolRecord(socketPath)
					if jsonOutput {
						return json.NewEncoder(cmd.OutOrStdout()).Encode(poolStatusView{State: "stopped", SocketPath: socketPath, LogPath: record.LogPath})
					}
					fmt.Fprintln(cmd.OutOrStdout(), "Terminal pool stopped")
					return nil
				}
				time.Sleep(25 * time.Millisecond)
			}
			return &cliError{code: 7, message: "timed out waiting for terminal pool to stop"}
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newPoolRestartCommand(socket, logFile, configPath *string) *cobra.Command {
	var keepAccess bool
	command := &cobra.Command{
		Use: "restart", Short: "Restart the current-user terminal pool service", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			socketPath := resolveV3Socket(*socket)
			status, record, err := poolStatus(socketPath, *logFile, *configPath)
			if err != nil {
				return err
			}
			// access 是独立进程：默认随 pool 重启；--keep-access 保留它，实现"只升级 pool"。
			accessPreserved := false
			if keepAccess && poolRecordAccessMatches(record) {
				accessPreserved = true
			} else if err := stopManagedAccess(record); err != nil {
				return classifyCLIError(err)
			}
			if status.State == "running" || status.State == "starting" {
				if err := stopPoolProcess(record.PID); err != nil {
					return classifyCLIError(err)
				}
				deadline := time.Now().Add(5 * time.Second)
				for poolRecordProcessMatches(record) && time.Now().Before(deadline) {
					time.Sleep(25 * time.Millisecond)
				}
				if poolRecordProcessMatches(record) {
					return &cliError{code: 7, message: "timed out waiting for terminal pool to stop"}
				}
				removePoolRecord(record.SocketPath)
			}
			if strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_LISTEN")) == "" && record.DirectListen != "" {
				if err := os.Setenv("ANYTTY_DIRECT_LISTEN", record.DirectListen); err != nil {
					return err
				}
			}
			if err := startDetachedPool(socketPath, resolveV3LogFilePath(*logFile), strings.TrimSpace(*configPath)); err != nil {
				return classifyCLIError(err)
			}
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				status, _, err = poolStatus(socketPath, *logFile, *configPath)
				if err == nil && status.State == "running" {
					break
				}
				time.Sleep(25 * time.Millisecond)
			}
			if err != nil || status.State != "running" {
				return &cliError{code: 6, message: "terminal pool did not become ready after restart"}
			}
			accessReady := false
			if accessPreserved {
				if patchErr := patchPoolRecordAccess(socketPath, record.AccessPID, record.AccessProcessID, record.AccessLogPath); patchErr == nil {
					if view, _, readErr := currentAccessStatus(socketPath); readErr == nil && view.State == "running" {
						accessReady = true
					}
				}
			}
			if !accessReady {
				if accessPreserved {
					// Keep-access 期望保留的进程已不可用：先清理旧身份再拉起新的。
					_, staleRecord, _ := currentAccessStatus(socketPath)
					_ = stopManagedAccess(staleRecord)
				}
				if err := startManagedAccess(socketPath); err != nil {
					if status.PID > 0 {
						_ = stopPoolProcess(status.PID)
					}
					return classifyCLIError(err)
				}
			}
			view, _, err := currentAccessStatus(socketPath)
			if err != nil {
				return err
			}
			if view.State != "running" {
				return &cliError{code: 6, message: "access is not running after restart"}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Terminal pool running (pid %d), access running (pid %d)\n", status.PID, view.PID)
			return nil
		},
	}
	command.Flags().BoolVar(&keepAccess, "keep-access", false, "restart the terminal pool only; keep the running access process and terminals")
	return command
}

func newPoolStatusCommand(socket, logFile, configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "status", Short: "Show current-user terminal pool service status", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, _, err := poolStatus(resolveV3Socket(*socket), *logFile, *configPath)
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
			if status.AccessState != "" {
				fields = append(fields, cliField{Label: "Access", Value: status.AccessState})
			}
			if status.AccessPID > 0 {
				fields = append(fields, cliField{Label: "Access PID", Value: fmt.Sprintf("%d", status.AccessPID)})
			}
			if status.AccessLogPath != "" {
				fields = append(fields, cliField{Label: "Access log", Value: status.AccessLogPath})
			}
			return writeCLIFields(cmd.OutOrStdout(), fields...)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newPoolLogsCommand(logFile *string) *cobra.Command {
	var follow bool
	var lines int
	command := &cobra.Command{
		Use: "logs", Short: "Read the current-user terminal pool log", Args: cobra.NoArgs,
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

func newPoolDoctorCommand(socket, logFile, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "doctor", Short: "Check terminal pool runtime paths and ownership", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, _, err := poolStatus(resolveV3Socket(*socket), *logFile, *configPath)
			if err != nil {
				return err
			}
			if err := writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "State", Value: status.State},
				cliField{Label: "Socket", Value: status.SocketPath},
				cliField{Label: "Record", Value: effectivePoolRecordPath(status.SocketPath)},
				cliField{Label: "Log", Value: status.LogPath},
				cliField{Label: "Access", Value: status.AccessState},
				cliField{Label: "Access log", Value: status.AccessLogPath},
			); err != nil {
				return err
			}
			if status.State != "running" {
				return &cliError{code: 6, message: "terminal pool is not running"}
			}
			return nil
		},
	}
}

// startManagedAccess 启动被 pool lifecycle 管理的 access 子进程，
// 复验 PID 身份后写入 record，并等待 protocol socket 就绪。
func startManagedAccess(socketPath string) error {
	logPath := accessLogPath()
	pid, err := startDetachedAccess(socketPath, logPath)
	if err != nil {
		return err
	}
	identity, err := poolProcessIdentity(pid)
	if err != nil {
		_ = stopPoolProcess(pid)
		return err
	}
	if err := patchPoolRecordAccess(socketPath, pid, identity, logPath); err != nil {
		_ = stopPoolProcess(pid)
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		record, _, readErr := readPoolRuntimeRecordForSocket(socketPath)
		if readErr == nil && accessProcessState(record) == "running" {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	record, _, _ := readPoolRuntimeRecordForSocket(socketPath)
	_ = stopManagedAccess(record)
	return &cliError{code: 6, message: "access did not become ready"}
}
