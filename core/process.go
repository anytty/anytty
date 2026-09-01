package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/shared/perftrace"
	crosspty "github.com/aymanbagabas/go-pty"
)

type ProcessFactory interface {
	Spawn(context.Context, ProcessSpec) (TerminalProcess, error)
}

type ProcessFactoryFunc func(context.Context, ProcessSpec) (TerminalProcess, error)

func (fn ProcessFactoryFunc) Spawn(ctx context.Context, spec ProcessSpec) (TerminalProcess, error) {
	return fn(ctx, spec)
}

type ProcessSpec struct {
	TerminalID         string
	Command            []string
	Size               Size
	Dir                string
	Env                []string
	ScrollbackSize     int
	ScrollbackMaxBytes int64
	ScrollbackMaxAge   time.Duration
}

func cloneProcessSpec(spec ProcessSpec) ProcessSpec {
	spec.Command = append([]string(nil), spec.Command...)
	spec.Env = append([]string(nil), spec.Env...)
	return spec
}

type TerminalProcess interface {
	Input([]byte) error
	Resize(Size) error
	Output() <-chan []byte
	CancelOutput()
	Kill() error
	Wait() <-chan ProcessExit
	Close() error
}

type terminalProcessResourceSampler interface {
	ResourceUsage() (TerminalResourceUsage, bool)
}

// ptyProcessFactory 是 core-v2 真实 terminal process 边界；外部客户端仍只通过 protocol/socket 访问。
type ptyProcessFactory struct{}

func newPTYProcessFactory() ProcessFactory {
	return ptyProcessFactory{}
}

func (ptyProcessFactory) Spawn(ctx context.Context, spec ProcessSpec) (TerminalProcess, error) {
	finishTotal := perftrace.Measure("core.process.spawn.total")
	defer finishTotal(0)
	if len(spec.Command) == 0 {
		return nil, ErrInvalidCommand
	}
	size := spec.Size
	if !size.Valid() {
		size = Size{Cols: 80, Rows: 24}
	}
	terminal, err := crosspty.New()
	if err != nil {
		return nil, fmt.Errorf("create terminal pty: %w", err)
	}
	if err := terminal.Resize(int(size.Cols), int(size.Rows)); err != nil {
		_ = terminal.Close()
		return nil, fmt.Errorf("set initial terminal size: %w", err)
	}
	cmd := terminal.CommandContext(ctx, spec.Command[0], spec.Command[1:]...)
	cmd.Dir = spec.Dir
	cmd.Env = ptyProcessEnv(spec.TerminalID, spec.Env)
	finishStart := perftrace.Measure("core.process.pty_start")
	err = cmd.Start()
	finishStart(0)
	if err != nil {
		_ = terminal.Close()
		return nil, err
	}
	platform, err := newPTYProcessPlatform(cmd.Process, terminal)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = terminal.Close()
		_ = cmd.Wait()
		return nil, fmt.Errorf("establish terminal process lifecycle: %w", err)
	}
	process := &ptyProcess{
		terminal:     terminal,
		cmd:          cmd,
		platform:     platform,
		outputCh:     make(chan []byte),
		outputCancel: make(chan struct{}),
		waitCh:       make(chan ProcessExit, 1),
		readDone:     make(chan struct{}),
	}
	go process.readLoop()
	go process.waitLoop()
	return process, nil
}

type ptyProcess struct {
	mu            sync.Mutex
	terminal      crosspty.Pty
	cmd           *crosspty.Cmd
	platform      ptyProcessPlatform
	outputCh      chan []byte
	outputCancel  chan struct{}
	waitCh        chan ProcessExit
	readDone      chan struct{}
	closeOnce     sync.Once
	outputOnce    sync.Once
	waitOnce      sync.Once
	killRequested atomic.Bool
}

const ptyReadBufferBytes = 64 * 1024

func (process *ptyProcess) Input(data []byte) error {
	process.mu.Lock()
	terminal := process.terminal
	process.mu.Unlock()
	if terminal == nil {
		return io.ErrClosedPipe
	}
	_, err := terminal.Write(data)
	return err
}

func (process *ptyProcess) Resize(size Size) error {
	if !size.Valid() {
		return ErrInvalidServerSize
	}
	process.mu.Lock()
	terminal := process.terminal
	process.mu.Unlock()
	if terminal == nil {
		return io.ErrClosedPipe
	}
	return terminal.Resize(int(size.Cols), int(size.Rows))
}

func (process *ptyProcess) Output() <-chan []byte {
	return process.outputCh
}

func (process *ptyProcess) CancelOutput() {
	process.outputOnce.Do(func() { close(process.outputCancel) })
}

func (process *ptyProcess) ResourceUsage() (TerminalResourceUsage, bool) {
	process.mu.Lock()
	platform := process.platform
	process.mu.Unlock()
	if platform == nil {
		return TerminalResourceUsage{}, false
	}
	return platform.ResourceUsage()
}

func (process *ptyProcess) Kill() error {
	process.mu.Lock()
	platform := process.platform
	process.killRequested.Store(true)
	process.mu.Unlock()
	if platform == nil {
		return nil
	}
	return platform.Kill()
}

func (process *ptyProcess) Wait() <-chan ProcessExit {
	return process.waitCh
}

func (process *ptyProcess) Close() error {
	var err error
	process.closeOnce.Do(func() {
		process.CancelOutput()
		killErr := process.Kill()
		process.mu.Lock()
		platform := process.platform
		process.terminal = nil
		process.platform = nil
		process.mu.Unlock()
		if platform != nil {
			err = errors.Join(err, platform.Close())
		}
		err = errors.Join(killErr, err)
	})
	return err
}

func (process *ptyProcess) readLoop() {
	defer close(process.outputCh)
	defer close(process.readDone)
	buf := make([]byte, ptyReadBufferBytes)
	for {
		process.mu.Lock()
		terminal := process.terminal
		process.mu.Unlock()
		if terminal == nil {
			return
		}
		n, err := terminal.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			select {
			case process.outputCh <- chunk:
			case <-process.outputCancel:
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (process *ptyProcess) waitLoop() {
	process.waitOnce.Do(func() {
		err := process.cmd.Wait()
		process.mu.Lock()
		platform := process.platform
		process.mu.Unlock()
		if platform != nil {
			_ = platform.ProcessExited()
		}
		<-process.readDone
		if platform != nil {
			_ = platform.OutputDrained()
		}
		process.mu.Lock()
		process.terminal = nil
		process.mu.Unlock()
		process.waitCh <- ProcessExit{
			Code: processExitCode(err, process.killRequested.Load()),
			Err:  err,
		}
		close(process.waitCh)
	})
}

func processExitCode(err error, killed bool) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if killed {
			return -1
		}
		return exitErr.ExitCode()
	}
	return -1
}

func parseProcessResourceUsage(pid int, output []byte, sampledAt time.Time) (TerminalResourceUsage, bool) {
	if pid <= 0 {
		return TerminalResourceUsage{}, false
	}
	fields := strings.Fields(string(output))
	if len(fields) < 2 {
		return TerminalResourceUsage{}, false
	}
	cpu, err := strconv.ParseFloat(strings.TrimSuffix(fields[0], "%"), 64)
	if err != nil {
		return TerminalResourceUsage{}, false
	}
	rssKB, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return TerminalResourceUsage{}, false
	}
	if cpu < 0 {
		cpu = 0
	}
	return TerminalResourceUsage{
		PID:            pid,
		CPUPercentX100: int(cpu*100 + 0.5),
		MemoryBytes:    rssKB * 1024,
		SampledAt:      sampledAt,
	}, true
}

type processResourceRow struct {
	pid   int
	ppid  int
	cpu   float64
	rssKB uint64
}

func parseForegroundProcess(rootPID int, output []byte) (string, bool) {
	processes := parseForegroundProcesses([]int{rootPID}, output)
	process, ok := processes[rootPID]
	return process, ok
}

func parseForegroundProcesses(rootPIDs []int, output []byte) map[int]string {
	snapshots := parseForegroundProcessSnapshots(rootPIDs, output)
	result := make(map[int]string, len(snapshots))
	for rootPID, snapshot := range snapshots {
		result[rootPID] = snapshot.Process
	}
	return result
}

type foregroundProcessInfo struct {
	PID     int
	Process string
	CWD     string
}

func parseForegroundProcessSnapshots(rootPIDs []int, output []byte) map[int]foregroundProcessInfo {
	type row struct {
		pid     int
		tpgid   int
		command string
	}
	rows := make(map[int]row)
	rootSet := make(map[int]struct{}, len(rootPIDs))
	for _, pid := range rootPIDs {
		if pid > 0 {
			rootSet[pid] = struct{}{}
		}
	}
	rootForegroundGroups := make(map[int]int, len(rootSet))
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		pid, pidErr := strconv.Atoi(fields[0])
		tpgid, groupErr := strconv.Atoi(fields[2])
		if pidErr != nil || groupErr != nil || pid <= 0 {
			continue
		}
		current := row{pid: pid, tpgid: tpgid, command: strings.Join(fields[3:], " ")}
		rows[pid] = current
		if _, wanted := rootSet[pid]; wanted {
			rootForegroundGroups[pid] = tpgid
		}
	}
	result := make(map[int]foregroundProcessInfo, len(rootForegroundGroups))
	for rootPID, group := range rootForegroundGroups {
		if group <= 0 {
			continue
		}
		foreground, ok := rows[group]
		if !ok {
			continue
		}
		if process, ok := normalizeForegroundProcessName(foreground.command); ok {
			result[rootPID] = foregroundProcessInfo{
				PID:     foreground.pid,
				Process: process,
			}
		}
	}
	return result
}

func normalizeForegroundProcessName(command string) (string, bool) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" || strings.IndexFunc(trimmed, func(r rune) bool { return r < 0x20 }) >= 0 {
		return "", false
	}
	lower := strings.ToLower(trimmed)
	for _, known := range []struct {
		name    string
		markers []string
	}{
		{name: "claude", markers: []string{"claude-code", "/claude", "\\claude"}},
		{name: "gemini", markers: []string{"gemini-cli", "/gemini", "\\gemini"}},
		{name: "codex", markers: []string{"/codex", "\\codex", " codex"}},
		{name: "copilot", markers: []string{"github-copilot", "copilot-cli", "/copilot", "\\copilot"}},
		{name: "cursor", markers: []string{"cursor-agent", "/cursor", "\\cursor"}},
		{name: "opencode", markers: []string{"/opencode", "\\opencode"}},
		{name: "qwen", markers: []string{"qwen-code", "/qwen", "\\qwen"}},
		{name: "cline", markers: []string{"/cline", "\\cline"}},
		{name: "zed", markers: []string{"/zed", "\\zed", " zed"}},
		{name: "warp", markers: []string{"warp-cli", "/warp", "\\warp"}},
	} {
		for _, marker := range known.markers {
			if strings.Contains(lower, marker) {
				return known.name, true
			}
		}
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "", false
	}
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(fields[0])), ".exe")
	if name == "" || len(name) > 128 {
		return "", false
	}
	return name, true
}

// parseProcessTreeResourceUsage aggregates the terminal root process and every
// descendant that still belongs to its process tree at sample time.
func parseProcessTreeResourceUsage(rootPID int, output []byte, sampledAt time.Time) (TerminalResourceUsage, bool) {
	if rootPID <= 0 {
		return TerminalResourceUsage{}, false
	}
	rows := make(map[int]processResourceRow)
	children := make(map[int][]int)
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		pid, pidErr := strconv.Atoi(fields[0])
		ppid, ppidErr := strconv.Atoi(fields[1])
		cpu, cpuErr := strconv.ParseFloat(strings.TrimSuffix(fields[2], "%"), 64)
		rssKB, rssErr := strconv.ParseUint(fields[3], 10, 64)
		if pidErr != nil || ppidErr != nil || cpuErr != nil || rssErr != nil || pid <= 0 {
			continue
		}
		if cpu < 0 {
			cpu = 0
		}
		rows[pid] = processResourceRow{pid: pid, ppid: ppid, cpu: cpu, rssKB: rssKB}
		children[ppid] = append(children[ppid], pid)
	}
	if _, ok := rows[rootPID]; !ok {
		return TerminalResourceUsage{}, false
	}
	stack := []int{rootPID}
	visited := make(map[int]struct{})
	var cpu float64
	var rssKB uint64
	for len(stack) > 0 {
		last := len(stack) - 1
		pid := stack[last]
		stack = stack[:last]
		if _, seen := visited[pid]; seen {
			continue
		}
		row, ok := rows[pid]
		if !ok {
			continue
		}
		visited[pid] = struct{}{}
		cpu += row.cpu
		rssKB += row.rssKB
		stack = append(stack, children[pid]...)
	}
	return TerminalResourceUsage{
		PID:            rootPID,
		CPUPercentX100: int(cpu*100 + 0.5),
		MemoryBytes:    rssKB * 1024,
		SampledAt:      sampledAt,
	}, true
}

func ptyProcessEnv(id string, extra []string) []string {
	env := os.Environ()
	env = append(env,
		"TERM=xterm-256color",
		"ANYTTY=1",
		"ANYTTY_TERMINAL_ID="+id,
	)
	return append(env, extra...)
}

type ProcessExit struct {
	Code int
	Err  error
}
