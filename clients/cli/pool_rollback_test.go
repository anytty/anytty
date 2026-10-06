package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRollbackStartedPoolIgnoresOtherInstance 固定并发 auto-start 的回滚边界：
// 只有本次启动的 PID 才能被回滚，否则输掉 record 竞争的一方会停掉赢家的 pool。
func TestRollbackStartedPoolIgnoresOtherInstance(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "anytty.sock")
	identity, err := poolProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatalf("process identity: %v", err)
	}
	record := poolRuntimeRecord{
		SchemaVersion: poolRecordSchemaVersion,
		PID:           os.Getpid(),
		ProcessID:     identity,
		InstanceToken: strings.Repeat("a", 64),
		Executable:    "/tmp/anytty-rollback-test",
		SocketPath:    socketPath,
		StartedAt:     time.Now().UTC(),
	}
	if err := writePoolRuntimeRecord(poolRecordPath(socketPath), record); err != nil {
		t.Fatalf("write record: %v", err)
	}

	rollbackStartedPool(socketPath, "", "", os.Getpid()+1)
	if _, err := os.Stat(poolRecordPath(socketPath)); err != nil {
		t.Fatalf("record must survive a foreign rollback: %v", err)
	}
}
