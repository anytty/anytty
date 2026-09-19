//go:build windows

package core

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

const windowsJobQueryHelperEnv = "ANYTTY_WINDOWS_JOB_QUERY_HELPER"

func TestWindowsCPUPercentX100UsesIntervalDelta(t *testing.T) {
	base := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	platform := &windowsPTYProcessPlatform{}

	if got := platform.cpuPercentX100(base, uint64(time.Second)); got != 0 {
		t.Fatalf("first CPU sample must establish a baseline, got %d", got)
	}
	if got := platform.cpuPercentX100(base.Add(500*time.Millisecond), uint64(1250*time.Millisecond)); got != 5000 {
		t.Fatalf("250ms CPU over 500ms should be 50%%, got %d", got)
	}
	if got := platform.cpuPercentX100(base.Add(time.Second), uint64(2250*time.Millisecond)); got != 20000 {
		t.Fatalf("1s CPU over 500ms should preserve multi-core 200%% usage, got %d", got)
	}
}

func TestWindowsCPUPercentX100ResetsAfterCounterRegression(t *testing.T) {
	base := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	platform := &windowsPTYProcessPlatform{}
	platform.cpuPercentX100(base, uint64(time.Second))

	if got := platform.cpuPercentX100(base.Add(time.Second), uint64(500*time.Millisecond)); got != 0 {
		t.Fatalf("regressed CPU counter must reset the baseline, got %d", got)
	}
	if got := platform.cpuPercentX100(base.Add(2*time.Second), uint64(time.Second)); got != 5000 {
		t.Fatalf("CPU sampling should recover after reset, got %d", got)
	}
}

func TestWindowsProcessWorkingSetReadsCurrentProcess(t *testing.T) {
	workingSet, ok := windowsProcessWorkingSet(uint32(os.Getpid()))
	if !ok || workingSet == 0 {
		t.Fatalf("expected a non-zero working set for the test process, got %d ok=%v", workingSet, ok)
	}
}

func TestWindowsJobQueriesAssignedProcess(t *testing.T) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	commands := make([]*exec.Cmd, 0, 2)
	t.Cleanup(func() {
		_ = windows.TerminateJobObject(job, 1)
		for _, command := range commands {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
		_ = windows.CloseHandle(job)
	})
	for range 2 {
		command := exec.Command(os.Args[0], "-test.run=^TestWindowsJobQueryHelperProcess$")
		command.Env = append(os.Environ(), windowsJobQueryHelperEnv+"=1")
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, command)
		handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(command.Process.Pid))
		if err != nil {
			t.Fatal(err)
		}
		assignErr := windows.AssignProcessToJobObject(job, handle)
		_ = windows.CloseHandle(handle)
		if assignErr != nil {
			t.Fatal(assignErr)
		}
	}
	time.Sleep(100 * time.Millisecond)

	processIDs, err := windowsJobProcessIDs(job)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range commands {
		found := false
		for _, processID := range processIDs {
			if processID == uint32(command.Process.Pid) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("assigned process %d missing from job process list %v", command.Process.Pid, processIDs)
		}
	}
	if _, err := windowsJobCPUTimeNanos(job); err != nil {
		t.Fatalf("query job CPU time: %v", err)
	}
	if workingSet, ok := windowsProcessTreeWorkingSet(processIDs); !ok || workingSet == 0 {
		t.Fatalf("expected a non-zero job working set, got %d ok=%v", workingSet, ok)
	}
	platform := &windowsPTYProcessPlatform{process: commands[0].Process, job: job}
	first, ok := platform.ResourceUsage()
	if !ok || first.MemoryBytes < 32<<20 {
		t.Fatalf("expected aggregate memory from both job members, got %#v ok=%v", first, ok)
	}
	time.Sleep(250 * time.Millisecond)
	second, ok := platform.ResourceUsage()
	if !ok || second.CPUPercentX100 <= 0 {
		t.Fatalf("expected non-zero aggregate interval CPU, got %#v ok=%v", second, ok)
	}
}

func TestWindowsJobQueryHelperProcess(t *testing.T) {
	if os.Getenv(windowsJobQueryHelperEnv) != "1" {
		return
	}
	memory := make([]byte, 16<<20)
	for index := 0; index < len(memory); index += 4096 {
		memory[index] = 1
	}
	deadline := time.Now().Add(30 * time.Second)
	var value uint64
	for time.Now().Before(deadline) {
		value = value*1664525 + 1013904223
	}
	runtime.KeepAlive(memory)
	runtime.KeepAlive(value)
}
