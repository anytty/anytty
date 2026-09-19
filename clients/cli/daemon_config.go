package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anytty/anytty/shared/userdirs"
	"gopkg.in/yaml.v3"
)

// daemonConfigFileName 是 daemon 运行时策略的历史配置文件（原 tui-v3.yaml
// 的 daemon 段）。旧 TUI 已删除，但该文件仍是 daemon/history 的物理策略输入，
// 保持同名同格式以兼容既有用户配置。
const daemonConfigFileName = "tui-v3.yaml"

const maxDaemonHistorySizeMB = 8 * 1024 * 1024

const (
	minDaemonResourceSampleIntervalMS = 100
	maxDaemonResourceSampleIntervalMS = 60_000
	minDaemonResourceHistorySamples   = 512
	maxDaemonResourceHistorySamples   = 65_536
)

type daemonHistoryConfig struct {
	MaxSizeMB        int    `yaml:"max_size_mb"`
	MaxAgeDays       int    `yaml:"max_age_days"`
	Compression      string `yaml:"compression"`
	CompressionLevel string `yaml:"compression_level"`
}

type daemonOutputBufferConfig struct {
	CapacityBytes       int64  `yaml:"capacity_bytes"`
	Overflow            string `yaml:"overflow"`
	ResidentBudgetBytes int64  `yaml:"resident_budget_bytes"`
}

type daemonResourceSamplingConfig struct {
	IntervalMS int `yaml:"interval_ms"`
	MaxSamples int `yaml:"max_samples"`
}

type daemonRuntimeConfig struct {
	History          daemonHistoryConfig          `yaml:"history"`
	OutputBuffer     daemonOutputBufferConfig     `yaml:"output_buffer"`
	ResourceSampling daemonResourceSamplingConfig `yaml:"resource_sampling"`
}

// daemonConfigDocument 用指针字段区分"未配置"与显式零值：max_size_mb: 0
// 与 max_age_days: 0 是合法的"关闭限制"配置，不能被默认值覆盖。
type daemonConfigDocument struct {
	Version int                 `yaml:"version"`
	Daemon  daemonConfigSegment `yaml:"daemon"`
}

type daemonConfigSegment struct {
	History          daemonHistorySegment          `yaml:"history"`
	OutputBuffer     daemonOutputBufferSegment     `yaml:"output_buffer"`
	ResourceSampling daemonResourceSamplingSegment `yaml:"resource_sampling"`
}

type daemonHistorySegment struct {
	MaxSizeMB        *int    `yaml:"max_size_mb"`
	MaxAgeDays       *int    `yaml:"max_age_days"`
	Compression      *string `yaml:"compression"`
	CompressionLevel *string `yaml:"compression_level"`
}

type daemonOutputBufferSegment struct {
	CapacityBytes       *int64  `yaml:"capacity_bytes"`
	Overflow            *string `yaml:"overflow"`
	ResidentBudgetBytes *int64  `yaml:"resident_budget_bytes"`
}

type daemonResourceSamplingSegment struct {
	IntervalMS *int `yaml:"interval_ms"`
	MaxSamples *int `yaml:"max_samples"`
}

func defaultDaemonRuntimeConfig() daemonRuntimeConfig {
	return daemonRuntimeConfig{
		History: daemonHistoryConfig{MaxSizeMB: 512, MaxAgeDays: 0, Compression: "zstd", CompressionLevel: "fast"},
		OutputBuffer: daemonOutputBufferConfig{
			CapacityBytes: 32 << 20, Overflow: "block", ResidentBudgetBytes: 512 << 20,
		},
		ResourceSampling: daemonResourceSamplingConfig{IntervalMS: 500, MaxSamples: 512},
	}
}

func daemonConfigDefaultPath() string {
	return filepath.Join(userdirs.ConfigHome(), "anytty", daemonConfigFileName)
}

// loadDaemonRuntimeConfig 解析 daemon 段的物理策略并应用 ANYTTY_* 环境覆盖。
// 默认路径缺失时使用内置默认值；显式路径缺失或校验失败直接报错。
func loadDaemonRuntimeConfig(path string) (daemonRuntimeConfig, error) {
	config := defaultDaemonRuntimeConfig()
	explicit := strings.TrimSpace(path) != ""
	if !explicit {
		path = daemonConfigDefaultPath()
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) || explicit {
				return daemonRuntimeConfig{}, err
			}
		} else {
			var document daemonConfigDocument
			if err := yaml.Unmarshal(data, &document); err != nil {
				return daemonRuntimeConfig{}, fmt.Errorf("parse daemon config %q: %w", path, err)
			}
			if document.Version != 0 && document.Version != 1 {
				return daemonRuntimeConfig{}, fmt.Errorf("daemon config version must be 1, got %d", document.Version)
			}
			config = mergeDaemonRuntimeConfig(config, document.Daemon)
		}
	}
	if err := applyDaemonConfigEnv(&config, os.LookupEnv); err != nil {
		return daemonRuntimeConfig{}, err
	}
	if err := validateDaemonRuntimeConfig(config); err != nil {
		return daemonRuntimeConfig{}, err
	}
	return config, nil
}

// mergeDaemonRuntimeConfig 只覆盖配置文档中显式出现的字段。
func mergeDaemonRuntimeConfig(base daemonRuntimeConfig, value daemonConfigSegment) daemonRuntimeConfig {
	if value.History.MaxSizeMB != nil {
		base.History.MaxSizeMB = *value.History.MaxSizeMB
	}
	if value.History.MaxAgeDays != nil {
		base.History.MaxAgeDays = *value.History.MaxAgeDays
	}
	if value.History.Compression != nil && strings.TrimSpace(*value.History.Compression) != "" {
		base.History.Compression = *value.History.Compression
	}
	if value.History.CompressionLevel != nil && strings.TrimSpace(*value.History.CompressionLevel) != "" {
		base.History.CompressionLevel = *value.History.CompressionLevel
	}
	if value.OutputBuffer.CapacityBytes != nil {
		base.OutputBuffer.CapacityBytes = *value.OutputBuffer.CapacityBytes
	}
	if value.OutputBuffer.Overflow != nil && strings.TrimSpace(*value.OutputBuffer.Overflow) != "" {
		base.OutputBuffer.Overflow = *value.OutputBuffer.Overflow
	}
	if value.OutputBuffer.ResidentBudgetBytes != nil {
		base.OutputBuffer.ResidentBudgetBytes = *value.OutputBuffer.ResidentBudgetBytes
	}
	if value.ResourceSampling.IntervalMS != nil {
		base.ResourceSampling.IntervalMS = *value.ResourceSampling.IntervalMS
	}
	if value.ResourceSampling.MaxSamples != nil {
		base.ResourceSampling.MaxSamples = *value.ResourceSampling.MaxSamples
	}
	return base
}

func applyDaemonConfigEnv(config *daemonRuntimeConfig, lookup func(string) (string, bool)) error {
	lookupString := func(name string) string {
		if lookup == nil {
			return ""
		}
		value, _ := lookup(name)
		return strings.TrimSpace(value)
	}
	setInt := func(name string, target *int) error {
		value := lookupString(name)
		if value == "" {
			return nil
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		*target = parsed
		return nil
	}
	setInt64 := func(name string, target *int64) error {
		value := lookupString(name)
		if value == "" {
			return nil
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		*target = parsed
		return nil
	}
	if err := setInt64("ANYTTY_OUTPUT_BUFFER_CAPACITY_BYTES", &config.OutputBuffer.CapacityBytes); err != nil {
		return err
	}
	if value := lookupString("ANYTTY_OUTPUT_BUFFER_OVERFLOW"); value != "" {
		config.OutputBuffer.Overflow = value
	}
	if err := setInt64("ANYTTY_OUTPUT_RESIDENT_BUDGET_BYTES", &config.OutputBuffer.ResidentBudgetBytes); err != nil {
		return err
	}
	if err := setInt("ANYTTY_HISTORY_MAX_SIZE_MB", &config.History.MaxSizeMB); err != nil {
		return err
	}
	if err := setInt("ANYTTY_HISTORY_MAX_AGE_DAYS", &config.History.MaxAgeDays); err != nil {
		return err
	}
	if value := lookupString("ANYTTY_HISTORY_COMPRESSION"); value != "" {
		config.History.Compression = value
	}
	if value := lookupString("ANYTTY_HISTORY_COMPRESSION_LEVEL"); value != "" {
		config.History.CompressionLevel = value
	}
	if err := setInt("ANYTTY_RESOURCE_SAMPLING_INTERVAL_MS", &config.ResourceSampling.IntervalMS); err != nil {
		return err
	}
	if err := setInt("ANYTTY_RESOURCE_SAMPLING_MAX_SAMPLES", &config.ResourceSampling.MaxSamples); err != nil {
		return err
	}
	return nil
}

func validateDaemonRuntimeConfig(config daemonRuntimeConfig) error {
	if config.OutputBuffer.CapacityBytes < 64<<10 || config.OutputBuffer.CapacityBytes > 256<<20 {
		return fmt.Errorf("daemon.output_buffer.capacity_bytes must be between 65536 and 268435456")
	}
	if !daemonConfigOneOf(config.OutputBuffer.Overflow, "drop", "block") {
		return fmt.Errorf("daemon.output_buffer.overflow must be drop or block, got %q", config.OutputBuffer.Overflow)
	}
	if config.OutputBuffer.ResidentBudgetBytes < 64<<10 || config.OutputBuffer.ResidentBudgetBytes > 2<<30 {
		return fmt.Errorf("daemon.output_buffer.resident_budget_bytes must be between 65536 and 2147483648")
	}
	if config.ResourceSampling.IntervalMS < minDaemonResourceSampleIntervalMS || config.ResourceSampling.IntervalMS > maxDaemonResourceSampleIntervalMS {
		return fmt.Errorf("daemon.resource_sampling.interval_ms must be between %d and %d", minDaemonResourceSampleIntervalMS, maxDaemonResourceSampleIntervalMS)
	}
	if config.ResourceSampling.MaxSamples < minDaemonResourceHistorySamples || config.ResourceSampling.MaxSamples > maxDaemonResourceHistorySamples {
		return fmt.Errorf("daemon.resource_sampling.max_samples must be between %d and %d", minDaemonResourceHistorySamples, maxDaemonResourceHistorySamples)
	}
	if config.History.MaxSizeMB < 0 || config.History.MaxSizeMB > maxDaemonHistorySizeMB {
		return fmt.Errorf("daemon.history.max_size_mb must be between 0 and %d", maxDaemonHistorySizeMB)
	}
	if config.History.MaxAgeDays < 0 || config.History.MaxAgeDays > 36500 {
		return fmt.Errorf("daemon.history.max_age_days must be between 0 and 36500")
	}
	if !daemonConfigOneOf(config.History.Compression, "zstd", "s2", "none") {
		return fmt.Errorf("daemon.history.compression must be zstd, s2 or none, got %q", config.History.Compression)
	}
	if !daemonConfigOneOf(config.History.CompressionLevel, "fast", "balanced", "best") {
		return fmt.Errorf("daemon.history.compression_level must be fast, balanced or best, got %q", config.History.CompressionLevel)
	}
	return nil
}

func daemonConfigOneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
