//go:build !windows

package core

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	crosspty "github.com/anytty/anytty/internal/pty"
)

type ptyProcessPlatform interface {
	Kill() error
	ResourceUsage() (TerminalResourceUsage, bool)
	ProcessExited() error
	OutputDrained() error
	Close() error
}

func foregroundProcessSnapshot(rootPIDs []int) map[int]foregroundProcessInfo {
	if len(rootPIDs) == 0 {
		return nil
	}
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,tpgid=,command=").Output()
	if err != nil {
		return nil
	}
	snapshots := parseForegroundProcessSnapshots(rootPIDs, out)
	foregroundPIDs := make([]int, 0, len(snapshots))
	for _, snapshot := range snapshots {
		foregroundPIDs = append(foregroundPIDs, snapshot.PID)
	}
	workingDirectories := foregroundWorkingDirectories(foregroundPIDs)
	for rootPID, snapshot := range snapshots {
		snapshot.CWD = workingDirectories[snapshot.PID]
		snapshots[rootPID] = snapshot
	}
	return snapshots
}

func foregroundWorkingDirectories(pids []int) map[int]string {
	result := make(map[int]string, len(pids))
	switch runtime.GOOS {
	case "linux":
		for _, pid := range pids {
			path, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/cwd")
			if err == nil {
				if normalized, ok := normalizeForegroundWorkingDirectory(path); ok {
					result[pid] = normalized
				}
			}
		}
	case "darwin":
		arguments := make([]string, 0, len(pids))
		for _, pid := range pids {
			if pid > 0 {
				arguments = append(arguments, strconv.Itoa(pid))
			}
		}
		if len(arguments) == 0 {
			return result
		}
		out, err := exec.Command(
			"lsof",
			"-a",
			"-p", strings.Join(arguments, ","),
			"-d", "cwd",
			"-Fpn",
		).Output()
		if err == nil {
			return parseForegroundWorkingDirectories(out)
		}
	}
	return result
}

func parseForegroundWorkingDirectories(output []byte) map[int]string {
	result := make(map[int]string)
	currentPID := 0
	for _, line := range strings.Split(string(output), "\n") {
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			pid, err := strconv.Atoi(line[1:])
			if err != nil || pid <= 0 {
				currentPID = 0
				continue
			}
			currentPID = pid
		case 'n':
			if currentPID == 0 {
				continue
			}
			if normalized, ok := normalizeForegroundWorkingDirectory(line[1:]); ok {
				result[currentPID] = normalized
			}
		}
	}
	return result
}

func normalizeForegroundWorkingDirectory(value string) (string, bool) {
	if value == "" || len(value) > 4096 ||
		strings.IndexFunc(value, func(r rune) bool { return r < 0x20 }) >= 0 ||
		!filepath.IsAbs(value) {
		return "", false
	}
	return filepath.Clean(value), true
}

type unixPTYProcessPlatform struct {
	process   *os.Process
	master    *os.File
	closeOnce sync.Once
	closeErr  error
}

func newPTYProcessPlatform(process *os.Process, terminal crosspty.Pty) (ptyProcessPlatform, error) {
	unixTerminal, ok := terminal.(crosspty.UnixPty)
	if !ok || unixTerminal.Master() == nil || unixTerminal.Slave() == nil {
		return nil, errors.New("unix terminal file descriptors are unavailable")
	}
	// The child inherited its own slave descriptor during Start. Keeping the
	// parent's copy open prevents the master read loop from observing EOF after
	// the child exits, which leaves completed terminals stuck in running state.
	if err := unixTerminal.Slave().Close(); err != nil {
		return nil, err
	}
	return &unixPTYProcessPlatform{process: process, master: unixTerminal.Master()}, nil
}

func (platform *unixPTYProcessPlatform) Kill() error {
	if platform == nil || platform.process == nil {
		return nil
	}
	// 中文说明：Unix PTY command 由跨平台 PTY owner 建立独立 session；终止必须覆盖整个进程组。
	if err := syscall.Kill(-platform.process.Pid, syscall.SIGHUP); err != nil {
		if signalErr := platform.process.Signal(syscall.SIGHUP); signalErr != nil && !errors.Is(signalErr, os.ErrProcessDone) {
			return errors.Join(err, signalErr)
		}
	}
	return nil
}

func (platform *unixPTYProcessPlatform) ResourceUsage() (TerminalResourceUsage, bool) {
	if platform == nil || platform.process == nil || platform.process.Pid <= 0 {
		return TerminalResourceUsage{}, false
	}
	pid := platform.process.Pid
	// 中文说明：terminal 的资源范围是 PTY 根进程及其完整后代树；一次进程表快照
	// 同时聚合 CPU 与 RSS，后台作业即使切换了进程组也不会从统计中消失。
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,%cpu=,rss=").Output()
	if err != nil {
		return TerminalResourceUsage{}, false
	}
	return parseProcessTreeResourceUsage(pid, out, time.Now().UTC())
}

func (*unixPTYProcessPlatform) ProcessExited() error { return nil }

func (platform *unixPTYProcessPlatform) OutputDrained() error { return platform.closeTerminal() }

func (platform *unixPTYProcessPlatform) Close() error { return platform.closeTerminal() }

func (platform *unixPTYProcessPlatform) closeTerminal() error {
	if platform == nil {
		return nil
	}
	platform.closeOnce.Do(func() {
		if platform.master != nil {
			platform.closeErr = platform.master.Close()
		}
	})
	return platform.closeErr
}
