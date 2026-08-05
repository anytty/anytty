//go:build windows

package core

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
	"unsafe"

	crosspty "github.com/aymanbagabas/go-pty"
	"golang.org/x/sys/windows"
)

var getProcessMemoryInfo = windows.NewLazySystemDLL("kernel32.dll").NewProc("K32GetProcessMemoryInfo")

type ptyProcessPlatform interface {
	Kill() error
	ResourceUsage() (TerminalResourceUsage, bool)
	ProcessExited() error
	OutputDrained() error
	Close() error
}

type windowsPTYProcessPlatform struct {
	process     *os.Process
	terminal    crosspty.ConPty
	job         windows.Handle
	consoleOnce sync.Once
	pipesOnce   sync.Once
	jobOnce     sync.Once
	pipesErr    error
	jobErr      error
}

type processMemoryCounters struct {
	Size                  uint32
	PageFaultCount        uint32
	PeakWorkingSetSize    uintptr
	WorkingSetSize        uintptr
	QuotaPeakPagedPool    uintptr
	QuotaPagedPool        uintptr
	QuotaPeakNonPagedPool uintptr
	QuotaNonPagedPool     uintptr
	PagefileUsage         uintptr
	PeakPagefileUsage     uintptr
}

func newPTYProcessPlatform(process *os.Process, terminal crosspty.Pty) (ptyProcessPlatform, error) {
	if process == nil || process.Pid <= 0 {
		return nil, fmt.Errorf("windows terminal process is unavailable")
	}
	conPTY, ok := terminal.(crosspty.ConPty)
	if !ok {
		return nil, fmt.Errorf("windows terminal is not a ConPTY")
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create terminal job object: %w", err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return nil, fmt.Errorf("configure terminal job object: %w", err)
	}
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil, fmt.Errorf("open terminal process: %w", err)
	}
	assignErr := windows.AssignProcessToJobObject(job, handle)
	_ = windows.CloseHandle(handle)
	if assignErr != nil {
		_ = windows.CloseHandle(job)
		return nil, fmt.Errorf("assign terminal process to job object: %w", assignErr)
	}
	return &windowsPTYProcessPlatform{process: process, terminal: conPTY, job: job}, nil
}

func (platform *windowsPTYProcessPlatform) Kill() error {
	if platform == nil || platform.job == 0 {
		return nil
	}
	// 中文说明：Windows terminal lifecycle 由 Job Object 覆盖根进程及其子进程，不能只杀 cmd.exe 留下孤儿进程。
	if err := windows.TerminateJobObject(platform.job, 1); err != nil && !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return err
	}
	return nil
}

func (platform *windowsPTYProcessPlatform) ResourceUsage() (TerminalResourceUsage, bool) {
	if platform == nil || platform.process == nil || platform.process.Pid <= 0 {
		return TerminalResourceUsage{}, false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ, false, uint32(platform.process.Pid))
	if err != nil {
		return TerminalResourceUsage{}, false
	}
	defer windows.CloseHandle(handle)
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil {
		return TerminalResourceUsage{}, false
	}
	counters := processMemoryCounters{Size: uint32(unsafe.Sizeof(processMemoryCounters{}))}
	result, _, _ := getProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.Size),
	)
	if result == 0 {
		return TerminalResourceUsage{}, false
	}
	sampledAt := time.Now().UTC()
	elapsedNanos := sampledAt.UnixNano() - creation.Nanoseconds()
	cpuNanos := filetimeDurationNanos(kernel) + filetimeDurationNanos(user)
	cpuX100 := 0
	if elapsedNanos > 0 && cpuNanos > 0 {
		cpuX100 = int(float64(cpuNanos) * 10000 / float64(elapsedNanos))
	}
	return TerminalResourceUsage{
		PID:            platform.process.Pid,
		CPUPercentX100: cpuX100,
		MemoryBytes:    uint64(counters.WorkingSetSize),
		SampledAt:      sampledAt,
	}, true
}

func (platform *windowsPTYProcessPlatform) ProcessExited() error {
	if platform == nil {
		return nil
	}
	platform.consoleOnce.Do(func() {
		// 中文说明：先关闭 HPCON，ConPTY 才会关闭写端；readLoop 随后可排空最终输出并收到 EOF。
		windows.ClosePseudoConsole(windows.Handle(platform.terminal.Fd()))
	})
	return nil
}

func (platform *windowsPTYProcessPlatform) OutputDrained() error {
	if platform == nil {
		return nil
	}
	platform.pipesOnce.Do(func() {
		platform.pipesErr = errors.Join(platform.terminal.InputPipe().Close(), platform.terminal.OutputPipe().Close())
	})
	return platform.pipesErr
}

func (platform *windowsPTYProcessPlatform) Close() error {
	if platform == nil {
		return nil
	}
	_ = platform.ProcessExited()
	pipesErr := platform.OutputDrained()
	platform.jobOnce.Do(func() {
		if platform.job != 0 {
			platform.jobErr = windows.CloseHandle(platform.job)
			platform.job = 0
		}
	})
	return errors.Join(pipesErr, platform.jobErr)
}

func filetimeDurationNanos(value windows.Filetime) int64 {
	ticks := uint64(value.HighDateTime)<<32 | uint64(value.LowDateTime)
	return int64(ticks * 100)
}
