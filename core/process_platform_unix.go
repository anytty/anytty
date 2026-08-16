//go:build !windows

package core

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	crosspty "github.com/aymanbagabas/go-pty"
)

type ptyProcessPlatform interface {
	Kill() error
	ResourceUsage() (TerminalResourceUsage, bool)
	ProcessExited() error
	OutputDrained() error
	Close() error
}

func foregroundProcessSnapshot(rootPIDs []int) map[int]string {
	if len(rootPIDs) == 0 {
		return nil
	}
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,tpgid=,command=").Output()
	if err != nil {
		return nil
	}
	return parseForegroundProcesses(rootPIDs, out)
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
