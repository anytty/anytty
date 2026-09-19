//go:build windows

package core

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
	"unsafe"

	crosspty "github.com/anytty/anytty/internal/pty"
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

func foregroundProcessSnapshot([]int) map[int]foregroundProcessInfo { return nil }

type windowsPTYProcessPlatform struct {
	process     *os.Process
	terminal    crosspty.ConPty
	job         windows.Handle
	jobMu       sync.Mutex
	cpuSampleAt time.Time
	cpuNanos    uint64
	consoleOnce sync.Once
	pipesOnce   sync.Once
	jobOnce     sync.Once
	consoleErr  error
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

// x/sys/windows exposes QueryInformationJobObject but not these result layouts.
type jobObjectBasicAccountingInformation struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

type jobObjectBasicProcessIDList struct {
	NumberOfAssignedProcesses uint32
	NumberOfProcessIDsInList  uint32
	ProcessIDs                [1]uintptr
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
	if platform == nil {
		return nil
	}
	platform.jobMu.Lock()
	defer platform.jobMu.Unlock()
	if platform.job == 0 {
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
	platform.jobMu.Lock()
	defer platform.jobMu.Unlock()
	if platform.job == 0 {
		return TerminalResourceUsage{}, false
	}
	processIDs, err := windowsJobProcessIDs(platform.job)
	if err != nil {
		return TerminalResourceUsage{}, false
	}
	memoryBytes, ok := windowsProcessTreeWorkingSet(processIDs)
	if !ok {
		return TerminalResourceUsage{}, false
	}
	sampledAt := time.Now().UTC()
	cpuNanos, err := windowsJobCPUTimeNanos(platform.job)
	if err != nil {
		return TerminalResourceUsage{}, false
	}
	return TerminalResourceUsage{
		PID:            platform.process.Pid,
		CPUPercentX100: platform.cpuPercentX100(sampledAt, cpuNanos),
		MemoryBytes:    memoryBytes,
		SampledAt:      sampledAt,
	}, true
}

func windowsJobCPUTimeNanos(job windows.Handle) (uint64, error) {
	accounting := jobObjectBasicAccountingInformation{}
	if err := windows.QueryInformationJobObject(
		job,
		windows.JobObjectBasicAccountingInformation,
		uintptr(unsafe.Pointer(&accounting)),
		uint32(unsafe.Sizeof(accounting)),
		nil,
	); err != nil {
		return 0, err
	}
	if accounting.TotalUserTime < 0 || accounting.TotalKernelTime < 0 {
		return 0, fmt.Errorf("job object returned a negative CPU time")
	}
	ticks := uint64(accounting.TotalUserTime) + uint64(accounting.TotalKernelTime)
	if ticks < uint64(accounting.TotalUserTime) || ticks > ^uint64(0)/100 {
		return 0, fmt.Errorf("job object CPU time overflow")
	}
	return ticks * 100, nil
}

func windowsJobProcessIDs(job windows.Handle) ([]uint32, error) {
	const (
		initialCapacity = 16
		maxCapacity     = 1 << 20
	)
	capacity := uint32(initialCapacity)
	headerBytes := unsafe.Offsetof(jobObjectBasicProcessIDList{}.ProcessIDs)
	pointerBytes := unsafe.Sizeof(uintptr(0))
	for capacity <= maxCapacity {
		bufferBytes := headerBytes + uintptr(capacity)*pointerBytes
		buffer := make([]uintptr, (bufferBytes+pointerBytes-1)/pointerBytes)
		list := (*jobObjectBasicProcessIDList)(unsafe.Pointer(&buffer[0]))
		err := windows.QueryInformationJobObject(
			job,
			windows.JobObjectBasicProcessIdList,
			uintptr(unsafe.Pointer(list)),
			uint32(bufferBytes),
			nil,
		)
		if err == nil {
			count := list.NumberOfProcessIDsInList
			if count > capacity {
				return nil, fmt.Errorf("job object returned %d process IDs into a %d-entry buffer", count, capacity)
			}
			rawIDs := unsafe.Slice(&list.ProcessIDs[0], int(count))
			processIDs := make([]uint32, 0, count)
			for _, rawID := range rawIDs {
				if rawID > 0 && uint64(rawID) <= uint64(^uint32(0)) {
					processIDs = append(processIDs, uint32(rawID))
				}
			}
			return processIDs, nil
		}
		if !errors.Is(err, windows.ERROR_MORE_DATA) {
			return nil, err
		}
		nextCapacity := list.NumberOfAssignedProcesses
		if nextCapacity <= capacity {
			nextCapacity = capacity * 2
		}
		capacity = nextCapacity
	}
	return nil, fmt.Errorf("job object process list exceeds %d entries", maxCapacity)
}

func windowsProcessTreeWorkingSet(processIDs []uint32) (uint64, bool) {
	var total uint64
	observed := false
	for _, processID := range processIDs {
		workingSet, ok := windowsProcessWorkingSet(processID)
		if !ok {
			continue
		}
		observed = true
		if ^uint64(0)-total < workingSet {
			total = ^uint64(0)
			continue
		}
		total += workingSet
	}
	return total, observed
}

func windowsProcessWorkingSet(processID uint32) (uint64, bool) {
	if processID == 0 {
		return 0, false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ, false, processID)
	if err != nil {
		return 0, false
	}
	defer windows.CloseHandle(handle)
	counters := processMemoryCounters{Size: uint32(unsafe.Sizeof(processMemoryCounters{}))}
	result, _, _ := getProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.Size),
	)
	if result == 0 {
		return 0, false
	}
	return uint64(counters.WorkingSetSize), true
}

func (platform *windowsPTYProcessPlatform) cpuPercentX100(sampledAt time.Time, cpuNanos uint64) int {
	percentX100 := 0
	if !platform.cpuSampleAt.IsZero() && sampledAt.After(platform.cpuSampleAt) && cpuNanos >= platform.cpuNanos {
		elapsedNanos := sampledAt.Sub(platform.cpuSampleAt).Nanoseconds()
		if elapsedNanos > 0 {
			percentX100 = int(float64(cpuNanos-platform.cpuNanos) * 10000 / float64(elapsedNanos))
		}
	}
	platform.cpuSampleAt = sampledAt
	platform.cpuNanos = cpuNanos
	return percentX100
}

func (platform *windowsPTYProcessPlatform) ProcessExited() error {
	if platform == nil {
		return nil
	}
	platform.consoleOnce.Do(func() {
		// 中文说明：先关闭 HPCON，ConPTY 才会关闭写端；readLoop 随后可排空最终输出并收到 EOF。
		platform.consoleErr = platform.terminal.ClosePseudoConsole()
	})
	return platform.consoleErr
}

func (platform *windowsPTYProcessPlatform) OutputDrained() error {
	if platform == nil {
		return nil
	}
	platform.pipesOnce.Do(func() {
		platform.pipesErr = platform.terminal.ClosePipes()
	})
	return platform.pipesErr
}

func (platform *windowsPTYProcessPlatform) Close() error {
	if platform == nil {
		return nil
	}
	consoleErr := platform.ProcessExited()
	pipesErr := platform.OutputDrained()
	platform.jobOnce.Do(func() {
		platform.jobMu.Lock()
		defer platform.jobMu.Unlock()
		if platform.job != 0 {
			platform.jobErr = windows.CloseHandle(platform.job)
			platform.job = 0
		}
	})
	return errors.Join(consoleErr, pipesErr, platform.jobErr)
}
