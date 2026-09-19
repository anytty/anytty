package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anytty/anytty/access/accessrun"
	"github.com/anytty/anytty/shared/securefs"
)

// accessRunArgs 构造 `anytty access run` 子进程参数。
// Direct listener 只通过环境变量配置时也要透传，保证 pool --route 语义不丢。
func accessRunArgs(socketPath, accessLogPath string) []string {
	args := []string{"--socket", socketPath, "--log-file", accessLogPath}
	if route := strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_LISTEN")); route != "" {
		args = append(args, "--route", route)
	}
	return append(args, "access", "run")
}

// accessLogPath 返回 access 进程日志路径；它始终与 pool 日志分开。
func accessLogPath() string {
	if explicit := strings.TrimSpace(os.Getenv("ANYTTY_ACCESS_LOG_FILE")); explicit != "" {
		return explicit
	}
	return accessrun.DefaultLogFile()
}

// patchPoolRecordAccess 把 access 子进程身份写入 pool runtime record，
// 让 status/stop 能管理两个进程。record 仍由 pool owner 创建，这里只做原子补写。
func patchPoolRecordAccess(socketPath string, accessPID int, accessProcessID, accessLogPath string) error {
	path := effectivePoolRecordPath(socketPath)
	record, err := readPoolRuntimeRecord(path)
	if err != nil {
		return err
	}
	record.AccessPID = accessPID
	record.AccessProcessID = accessProcessID
	record.AccessLogPath = accessLogPath
	return writePoolRuntimeRecord(path, record)
}

// clearPoolRecordAccess 清除已停止 access 的身份字段。
func clearPoolRecordAccess(socketPath string) error {
	path := effectivePoolRecordPath(socketPath)
	record, err := readPoolRuntimeRecord(path)
	if err != nil {
		return err
	}
	if record.AccessPID == 0 {
		return nil
	}
	record.AccessPID = 0
	record.AccessProcessID = ""
	record.AccessLogPath = ""
	return writePoolRuntimeRecord(path, record)
}

func writePoolRuntimeRecord(path string, record poolRuntimeRecord) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := securefs.SecureDirectory(dir); err != nil {
		return fmt.Errorf("secure pool runtime directory: %w", err)
	}
	temporary, err := os.CreateTemp(dir, ".pool-record-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	cleanup := func() { _ = os.Remove(temporaryPath) }
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		cleanup()
		return err
	}
	encoder := json.NewEncoder(temporary)
	if err := encoder.Encode(record); err != nil {
		_ = temporary.Close()
		cleanup()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		cleanup()
		return err
	}
	if err := temporary.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		cleanup()
		return err
	}
	return securefs.SecureFile(path)
}

// poolRecordAccessMatches 复验 access 子进程身份没有变化。
func poolRecordAccessMatches(record poolRuntimeRecord) bool {
	if record.AccessPID <= 0 || record.AccessProcessID == "" {
		return false
	}
	identity, err := poolProcessIdentity(record.AccessPID)
	return err == nil && identity == record.AccessProcessID
}

// waitForProviderSocket 等待 pool provider socket 完成 Hello。
// access 是 canonical 入口，必须先确认 provider 就绪，否则首批终端命令会 race。
func waitForProviderSocket(socketPath string, timeout time.Duration) error {
	provider := accessrun.ProviderSocketPath(socketPath)
	return waitForSocket(provider, timeout, func() error {
		return probeProviderSocket(provider)
	})
}

// accessProcessState 返回 access 进程的健康状态：running/starting/stale/stopped。
func accessProcessState(record poolRuntimeRecord) string {
	if record.AccessPID <= 0 {
		return "stopped"
	}
	if !poolRecordAccessMatches(record) {
		return "stale"
	}
	if err := probeV3Socket(accessHealthSocket(record.SocketPath)); err != nil {
		return "starting"
	}
	return "running"
}

// accessHealthSocket 返回用于探测 access 进程的 socket：
// Phase 3 flip 后 access 持有 canonical 客户端入口。
func accessHealthSocket(socketPath string) string {
	return accessrun.DefaultAccessSocket(socketPath)
}

// poolHealthSocket 返回用于探测 pool 终端进程的 socket：
// Phase 3 flip 后 pool 只在 <canonical>.provider 上服务 access。
func poolHealthSocket(socketPath string) string {
	return accessrun.ProviderSocketPath(socketPath)
}

// stopManagedAccess 停止 record 中登记的 access 进程；身份不匹配时只清理字段。
func stopManagedAccess(record poolRuntimeRecord) error {
	if record.AccessPID <= 0 {
		return nil
	}
	if poolRecordAccessMatches(record) {
		if err := stopPoolProcess(record.AccessPID); err != nil {
			return err
		}
		deadline := time.Now().Add(5 * time.Second)
		for poolRecordAccessMatches(record) && time.Now().Before(deadline) {
			time.Sleep(25 * time.Millisecond)
		}
		if poolRecordAccessMatches(record) {
			return errors.New("timed out waiting for access process to stop")
		}
	}
	return clearPoolRecordAccess(record.SocketPath)
}
