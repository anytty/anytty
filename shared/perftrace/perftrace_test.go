package perftrace

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnableFromEnvWritesSnapshot(t *testing.T) {
	Disable()
	path := filepath.Join(t.TempDir(), "perf", "trace.json")
	t.Setenv(envPath, path)
	t.Setenv(envIntervalMs, "10000")
	stop, gotPath, ok := EnableFromEnv(context.Background())
	if !ok || gotPath != path {
		t.Fatalf("expected perftrace enabled at %q, got enabled=%v path=%q", path, ok, gotPath)
	}
	finish := Measure("unit.measure")
	finish(42)
	Count("unit.count", 7)
	stop()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty perftrace snapshot")
	}
	if Current() != nil {
		t.Fatal("stop should disable active recorder")
	}
}

func TestEnableFromEnvJSONLAppendsProcessRecords(t *testing.T) {
	Disable()
	path := filepath.Join(t.TempDir(), "perf", "trace.jsonl")
	t.Setenv(envPath, path)
	t.Setenv(envIntervalMs, "10000")

	stopDaemon, _, ok := EnableFromEnvWithProcess(context.Background(), "core-v2-daemon")
	if !ok {
		t.Fatal("expected daemon perftrace enabled")
	}
	Count("daemon.event", 1)
	stopDaemon()

	stopTUI, _, ok := EnableFromEnvWithProcess(context.Background(), "tui-v3")
	if !ok {
		t.Fatal("expected tui perftrace enabled")
	}
	Count("tui.event", 1)
	stopTUI()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read jsonl trace: %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("expected 2 jsonl records, got %d: %s", len(lines), data)
	}
	var daemonRecord TraceRecord
	if err := json.Unmarshal(lines[0], &daemonRecord); err != nil {
		t.Fatalf("decode daemon record: %v", err)
	}
	if daemonRecord.Format != traceRecordFormat || daemonRecord.Process != "core-v2-daemon" || daemonRecord.Sequence != 1 {
		t.Fatalf("unexpected daemon record: %+v", daemonRecord)
	}
	if _, ok := daemonRecord.Snapshot.Event("daemon.event"); !ok {
		t.Fatal("daemon record missing daemon event")
	}
	var tuiRecord TraceRecord
	if err := json.Unmarshal(lines[1], &tuiRecord); err != nil {
		t.Fatalf("decode tui record: %v", err)
	}
	if tuiRecord.Format != traceRecordFormat || tuiRecord.Process != "tui-v3" || tuiRecord.Sequence != 1 {
		t.Fatalf("unexpected tui record: %+v", tuiRecord)
	}
	if _, ok := tuiRecord.Snapshot.Event("tui.event"); !ok {
		t.Fatal("tui record missing tui event")
	}
}
