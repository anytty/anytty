package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

func TestCodexLifecycle(t *testing.T) {
	ctx := Context{TerminalID: "terminal-a", DaemonSocket: "/isolated.sock"}
	store := &Store{}
	for i, step := range []struct{ event, source, want string }{
		{"SessionStart", "startup", "idle"}, {"UserPromptSubmit", "", "working"}, {"PermissionRequest", "", "blocked"}, {"PostToolUse", "", "working"}, {"SessionStart", "compact", "working"}, {"Stop", "", "idle"}, {"UserPromptSubmit", "", "working"}, {"Interrupt", "", "idle"}, {"SessionEnd", "", "exited"},
	} {
		e, err := DecodeCodex([]byte(fmt.Sprintf(`{"session_id":"s","hook_event_name":%q,"source":%q,"cwd":"/project"}`, step.event, step.source)), ctx)
		if err != nil {
			t.Fatal(err)
		}
		e.Epoch = 1
		e.Sequence = uint64(i + 1)
		r, changed, err := store.Apply(*e, time.Now())
		if err != nil || !changed || r.Status != step.want {
			t.Fatalf("%s: %+v, %v", step.event, r, err)
		}
	}
	ignored, err := DecodeCodex([]byte(`{"session_id":"s","hook_event_name":"SubagentStop"}`), ctx)
	if err != nil || ignored != nil {
		t.Fatal("subagent event changed parent")
	}
	ignored, err = DecodeCodex([]byte(`not even JSON`), Context{})
	if err != nil || ignored != nil {
		t.Fatal("outside AnyTTY must no-op")
	}
}

func TestPermissionsOrderingAndProtobufRestart(t *testing.T) {
	s := &Store{}
	e := Event{Agent: "opencode", SessionID: "s", TerminalID: "t", Epoch: 10, Sequence: 1, Status: "working", Kind: "status"}
	apply := func(want string) {
		t.Helper()
		r, changed, err := s.Apply(e, time.Now())
		if err != nil || !changed || r.Status != want {
			t.Fatalf("%+v -> %+v,%v", e, r, err)
		}
		e.Sequence++
	}
	apply("working")
	e.Status = ""
	e.Kind = "permission.asked"
	e.PermissionID = "p1"
	apply("blocked")
	e.PermissionID = "p2"
	apply("blocked")
	data, err := proto.Marshal(s.ProtoSnapshot("daemon-a"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := s.ProtoSnapshot("daemon-a")
	proto.Reset(snapshot)
	if err = proto.Unmarshal(data, snapshot); err != nil {
		t.Fatal(err)
	}
	s = &Store{}
	if err = s.Restore(snapshot); err != nil {
		t.Fatal(err)
	}
	e.Kind = "permission.replied"
	e.PermissionID = "p1"
	apply("blocked")
	e.Status = "idle"
	e.Kind = "status"
	apply("blocked")
	e.Status = ""
	e.Kind = "permission.replied"
	e.PermissionID = "p2"
	apply("idle")
	late := e
	late.Sequence = 1
	late.Status = "working"
	if _, changed, _ := s.Apply(late, time.Now()); changed {
		t.Fatal("stale sequence accepted")
	}
	e.Epoch = 11
	e.Sequence = 1
	e.Status = "working"
	e.Kind = "start"
	apply("working")
	late.Sequence = 100
	if _, changed, _ := s.Apply(late, time.Now()); changed {
		t.Fatal("old epoch accepted")
	}
}

func TestCodexSequenceParallelAndCompact(t *testing.T) {
	dir := t.TempDir()
	seen := sync.Map{}
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := &Event{Agent: "codex", SessionID: "s", Kind: "PreToolUse"}
			if err := StampCodex(context.Background(), dir, e); err != nil {
				t.Error(err)
				return
			}
			if _, loaded := seen.LoadOrStore(e.Sequence, true); loaded {
				t.Errorf("duplicate %d", e.Sequence)
			}
		}()
	}
	wg.Wait()
	e := &Event{Agent: "codex", SessionID: "s", Kind: "metadata"}
	if err := StampCodex(context.Background(), dir, e); err != nil {
		t.Fatal(err)
	}
	if e.Sequence != 21 {
		t.Fatal(e)
	}
	epoch := e.Epoch
	e.Kind = "start"
	if err := StampCodex(context.Background(), dir, e); err != nil {
		t.Fatal(err)
	}
	if e.Sequence != 1 || e.Epoch <= epoch {
		t.Fatal("resume must fence previous epoch")
	}
}

func TestInstallerPreservesOtherHooksAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")
	original := `{"description":"keep me","extra":{"key":true},"hooks":{"Stop":[{"matcher":".*","hooks":[{"type":"command","command":"echo unrelated"}]}]}}`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := InstallCodex(dir, "/tmp/an y'tty"); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(path)
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatal(err)
	}
	var hooks map[string][]json.RawMessage
	_ = json.Unmarshal(root["hooks"], &hooks)
	if len(hooks["Stop"]) != 2 || len(hooks["SessionStart"]) != 1 {
		t.Fatal(string(data))
	}
	if err := UninstallCodex(dir); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	var actual, expected any
	_ = json.Unmarshal(data, &actual)
	_ = json.Unmarshal([]byte(original), &expected)
	a, _ := json.Marshal(actual)
	b, _ := json.Marshal(expected)
	if string(a) != string(b) {
		t.Fatalf("lost user config: %s", data)
	}
	if err := InstallOpenCode(dir, "/tmp/anytty"); err != nil {
		t.Fatal(err)
	}
	if err := UninstallOpenCode(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugins", "anytty-agents.js"), []byte("user plugin"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := InstallOpenCode(dir, "/tmp/anytty"); err == nil {
		t.Fatal("overwrote user file")
	}
	if err := UninstallOpenCode(dir); err == nil {
		t.Fatal("deleted user file")
	}
}
