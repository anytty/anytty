package core

import (
	"errors"
	"fmt"
	"time"
)

const (
	DefaultTerminalOutputBufferCapacityBytes int64 = 32 << 20
	MinTerminalOutputBufferCapacityBytes     int64 = 64 << 10
	MaxTerminalOutputBufferCapacityBytes     int64 = 256 << 20
	DefaultTerminalOutputResidentBudgetBytes int64 = 512 << 20
	MinTerminalOutputResidentBudgetBytes     int64 = 64 << 10
	MaxTerminalOutputResidentBudgetBytes     int64 = 2 << 30
)

type TerminalOutputOverflowPolicy string

const (
	TerminalOutputOverflowDrop  TerminalOutputOverflowPolicy = "drop"
	TerminalOutputOverflowBlock TerminalOutputOverflowPolicy = "block"
)

// TerminalOutputBufferConfig controls the one shared PTY payload buffer owned by
// each terminal process generation. Live and history retain independent cursors.
type TerminalOutputBufferConfig struct {
	CapacityBytes int64
	Overflow      TerminalOutputOverflowPolicy
}

func DefaultTerminalOutputBufferConfig() TerminalOutputBufferConfig {
	return TerminalOutputBufferConfig{
		CapacityBytes: DefaultTerminalOutputBufferCapacityBytes,
		Overflow:      TerminalOutputOverflowBlock,
	}
}

func (cfg TerminalOutputBufferConfig) Validate() error {
	if cfg.CapacityBytes < MinTerminalOutputBufferCapacityBytes || cfg.CapacityBytes > MaxTerminalOutputBufferCapacityBytes {
		return fmt.Errorf("terminal output buffer capacity must be between %d and %d bytes", MinTerminalOutputBufferCapacityBytes, MaxTerminalOutputBufferCapacityBytes)
	}
	if cfg.Overflow != TerminalOutputOverflowDrop && cfg.Overflow != TerminalOutputOverflowBlock {
		return fmt.Errorf("terminal output buffer overflow must be drop or block, got %q", cfg.Overflow)
	}
	return nil
}

func (cfg TerminalOutputBufferConfig) normalized() TerminalOutputBufferConfig {
	defaults := DefaultTerminalOutputBufferConfig()
	if cfg.CapacityBytes == 0 {
		cfg.CapacityBytes = defaults.CapacityBytes
	}
	if cfg.Overflow == "" {
		cfg.Overflow = defaults.Overflow
	}
	if cfg.Validate() != nil {
		return defaults
	}
	return cfg
}

type TerminalCreateOptions struct {
	Dir                string
	Env                []string
	ScrollbackSize     int
	ScrollbackMaxBytes int64
	ScrollbackMaxAge   time.Duration
}

func cloneTerminalCreateOptions(options TerminalCreateOptions) TerminalCreateOptions {
	options.Env = append([]string(nil), options.Env...)
	return options
}

type Size struct {
	Cols uint16
	Rows uint16
}

func (size Size) Valid() bool {
	return size.Cols > 0 && size.Rows > 0
}

type TerminalState string

const (
	TerminalStateCreated TerminalState = "created"
	TerminalStateRunning TerminalState = "running"
	TerminalStateExited  TerminalState = "exited"
	TerminalStateRemoved TerminalState = "removed"
)

type TerminalInfo struct {
	ID                string
	Name              string
	Command           []string
	Tags              map[string]string
	Size              Size
	State             TerminalState
	CWD               string
	LiveCWD           string
	CreatedAt         time.Time
	ExitCode          *int
	ExitedAt          time.Time
	Resources         TerminalResourceUsage
	ResourceHistory   []TerminalResourceUsage
	ForegroundProcess string
	LastOutputAt      time.Time
}

func (info TerminalInfo) Clone() TerminalInfo {
	info.Command = append([]string(nil), info.Command...)
	info.Tags = cloneStringMap(info.Tags)
	info.ResourceHistory = append([]TerminalResourceUsage(nil), info.ResourceHistory...)
	if info.ExitCode != nil {
		code := *info.ExitCode
		info.ExitCode = &code
	}
	return info
}

// TerminalResourceUsage 是 core-v2 对 terminal 进程资源的采样结果；真值来自当前
// TerminalProcess 的 OS 进程采样，只作为 Terminal Manager 诊断投影，不参与 terminal 生命周期判断。
type TerminalResourceUsage struct {
	PID            int
	CPUPercentX100 int
	MemoryBytes    uint64
	SampledAt      time.Time
}

type TerminalRecord struct {
	ID      string
	Name    string
	Command []string
	Tags    map[string]string
	Size    Size
	Options TerminalCreateOptions
}

// HistoryBacklogStatus exposes the history cursor's view of the shared terminal
// output buffer. ResidentBytes counts shared payload once, not once per consumer.
type HistoryBacklogStatus struct {
	TerminalID             string
	HistoryEnabled         bool
	OutputBufferPolicy     TerminalOutputOverflowPolicy
	BufferCapacityBytes    int64
	ResidentBytes          int64
	AggregateResidentBytes int64
	AggregateBudgetBytes   int64
	DroppedBytes           uint64
	GapCount               uint64
	OutputBufferWaitNanos  int64
	Unavailable            bool
	UnavailableReason      string
	Closed                 bool
}

var (
	ErrServerClosed              = errors.New("core-v2 server closed")
	ErrInvalidTerminalID         = errors.New("invalid terminal id")
	ErrInvalidCommand            = errors.New("invalid command")
	ErrDuplicateTerminal         = errors.New("duplicate terminal")
	ErrTerminalNotFound          = errors.New("terminal not found")
	ErrTerminalExited            = errors.New("terminal exited")
	ErrHistoryNotRebuilt         = errors.New("history not rebuilt")
	ErrHistoryDisabled           = errors.New("history disabled")
	ErrTerminalOutputSyncLost    = errors.New("terminal output sync lost")
	ErrTerminalOutputUnavailable = errors.New("terminal output consumer unavailable")
	ErrInvalidServerSize         = errors.New("invalid server size")
	ErrNilListenerFactory        = errors.New("nil listener factory")
	ErrInvalidStorageKey         = errors.New("invalid storage key")
	ErrStorageEntryNotFound      = errors.New("storage entry not found")
	ErrStorageVersionConflict    = errors.New("storage version conflict")
)

type TerminalOutputError struct {
	TerminalID   string
	Consumer     string
	Epoch        uint64
	DroppedBytes uint64
	Cause        error
}

func (err *TerminalOutputError) Error() string {
	if err == nil {
		return ErrTerminalOutputUnavailable.Error()
	}
	if err.DroppedBytes > 0 {
		return fmt.Sprintf("%s output sync lost at epoch %d after dropping %d bytes", err.Consumer, err.Epoch, err.DroppedBytes)
	}
	if err.Cause != nil {
		return fmt.Sprintf("%s output unavailable: %v", err.Consumer, err.Cause)
	}
	return fmt.Sprintf("%s output unavailable", err.Consumer)
}

func (err *TerminalOutputError) Unwrap() error {
	if err != nil && err.Cause != nil {
		return err.Cause
	}
	return ErrTerminalOutputUnavailable
}

func (err *TerminalOutputError) Is(target error) bool {
	if err == nil {
		return false
	}
	if target == ErrTerminalOutputUnavailable {
		return true
	}
	return err.DroppedBytes > 0 && target == ErrTerminalOutputSyncLost
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
