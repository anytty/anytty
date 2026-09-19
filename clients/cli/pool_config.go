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

// poolConfigFileName 是 terminal pool 运行时策略的历史配置文件（原 tui-v3.yaml
// 的 pool/daemon 段）。旧 TUI 已删除，但该文件仍是 pool/history 的物理策略输入，
// 保持同名同格式以兼容既有用户配置。
const poolConfigFileName = "tui-v3.yaml"

const maxPoolHistorySizeMB = 8 * 1024 * 1024

const (
	minPoolResourceSampleIntervalMS = 100
	maxPoolResourceSampleIntervalMS = 60_000
	minPoolResourceHistorySamples   = 512
	maxPoolResourceHistorySamples   = 65_536
)

type poolHistoryConfig struct {
	MaxSizeMB        int    `yaml:"max_size_mb"`
	MaxAgeDays       int    `yaml:"max_age_days"`
	Compression      string `yaml:"compression"`
	CompressionLevel string `yaml:"compression_level"`
}

type poolOutputBufferConfig struct {
	CapacityBytes       int64  `yaml:"capacity_bytes"`
	Overflow            string `yaml:"overflow"`
	ResidentBudgetBytes int64  `yaml:"resident_budget_bytes"`
}

type poolResourceSamplingConfig struct {
	IntervalMS int `yaml:"interval_ms"`
	MaxSamples int `yaml:"max_samples"`
}

type poolRuntimeConfig struct {
	History          poolHistoryConfig          `yaml:"history"`
	OutputBuffer     poolOutputBufferConfig     `yaml:"output_buffer"`
	ResourceSampling poolResourceSamplingConfig `yaml:"resource_sampling"`
}

// poolConfigDocument 用指针字段区分"未配置"与显式零值：max_size_mb: 0
// 与 max_age_days: 0 是合法的"关闭限制"配置，不能被默认值覆盖。
// `pool:` 是当前段；`daemon:` 段作为升级兼容继续解析，且 `pool:` 优先。
type poolConfigDocument struct {
	Version int               `yaml:"version"`
	Pool    poolConfigSegment `yaml:"pool"`
	Daemon  poolConfigSegment `yaml:"daemon"`
}

type poolConfigSegment struct {
	History          poolHistorySegment          `yaml:"history"`
	OutputBuffer     poolOutputBufferSegment     `yaml:"output_buffer"`
	ResourceSampling poolResourceSamplingSegment `yaml:"resource_sampling"`
}

type poolHistorySegment struct {
	MaxSizeMB        *int    `yaml:"max_size_mb"`
	MaxAgeDays       *int    `yaml:"max_age_days"`
	Compression      *string `yaml:"compression"`
	CompressionLevel *string `yaml:"compression_level"`
}

type poolOutputBufferSegment struct {
	CapacityBytes       *int64  `yaml:"capacity_bytes"`
	Overflow            *string `yaml:"overflow"`
	ResidentBudgetBytes *int64  `yaml:"resident_budget_bytes"`
}

type poolResourceSamplingSegment struct {
	IntervalMS *int `yaml:"interval_ms"`
	MaxSamples *int `yaml:"max_samples"`
}

func defaultPoolRuntimeConfig() poolRuntimeConfig {
	return poolRuntimeConfig{
		History: poolHistoryConfig{MaxSizeMB: 512, MaxAgeDays: 0, Compression: "zstd", CompressionLevel: "fast"},
		OutputBuffer: poolOutputBufferConfig{
			CapacityBytes: 32 << 20, Overflow: "block", ResidentBudgetBytes: 512 << 20,
		},
		ResourceSampling: poolResourceSamplingConfig{IntervalMS: 500, MaxSamples: 512},
	}
}

func poolConfigDefaultPath() string {
	return filepath.Join(userdirs.ConfigHome(), "anytty", poolConfigFileName)
}

// loadPoolRuntimeConfig 解析 pool 段的物理策略并应用 ANYTTY_* 环境覆盖。
// 默认路径缺失时使用内置默认值；显式路径缺失或校验失败直接报错。
func loadPoolRuntimeConfig(path string) (poolRuntimeConfig, error) {
	config := defaultPoolRuntimeConfig()
	explicit := strings.TrimSpace(path) != ""
	if !explicit {
		path = poolConfigDefaultPath()
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) || explicit {
				return poolRuntimeConfig{}, err
			}
		} else {
			var document poolConfigDocument
			if err := yaml.Unmarshal(data, &document); err != nil {
				return poolRuntimeConfig{}, fmt.Errorf("parse pool config %q: %w", path, err)
			}
			if document.Version != 0 && document.Version != 1 {
				return poolRuntimeConfig{}, fmt.Errorf("pool config version must be 1, got %d", document.Version)
			}
			config = mergePoolRuntimeConfig(config, document.Daemon)
			config = mergePoolRuntimeConfig(config, document.Pool)
		}
	}
	if err := applyPoolConfigEnv(&config, os.LookupEnv); err != nil {
		return poolRuntimeConfig{}, err
	}
	if err := validatePoolRuntimeConfig(config); err != nil {
		return poolRuntimeConfig{}, err
	}
	return config, nil
}

// mergePoolRuntimeConfig 只覆盖配置文档中显式出现的字段。
func mergePoolRuntimeConfig(base poolRuntimeConfig, value poolConfigSegment) poolRuntimeConfig {
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

func applyPoolConfigEnv(config *poolRuntimeConfig, lookup func(string) (string, bool)) error {
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

func validatePoolRuntimeConfig(config poolRuntimeConfig) error {
	if config.OutputBuffer.CapacityBytes < 64<<10 || config.OutputBuffer.CapacityBytes > 256<<20 {
		return fmt.Errorf("pool.output_buffer.capacity_bytes must be between 65536 and 268435456")
	}
	if !poolConfigOneOf(config.OutputBuffer.Overflow, "drop", "block") {
		return fmt.Errorf("pool.output_buffer.overflow must be drop or block, got %q", config.OutputBuffer.Overflow)
	}
	if config.OutputBuffer.ResidentBudgetBytes < 64<<10 || config.OutputBuffer.ResidentBudgetBytes > 2<<30 {
		return fmt.Errorf("pool.output_buffer.resident_budget_bytes must be between 65536 and 2147483648")
	}
	if config.ResourceSampling.IntervalMS < minPoolResourceSampleIntervalMS || config.ResourceSampling.IntervalMS > maxPoolResourceSampleIntervalMS {
		return fmt.Errorf("pool.resource_sampling.interval_ms must be between %d and %d", minPoolResourceSampleIntervalMS, maxPoolResourceSampleIntervalMS)
	}
	if config.ResourceSampling.MaxSamples < minPoolResourceHistorySamples || config.ResourceSampling.MaxSamples > maxPoolResourceHistorySamples {
		return fmt.Errorf("pool.resource_sampling.max_samples must be between %d and %d", minPoolResourceHistorySamples, maxPoolResourceHistorySamples)
	}
	if config.History.MaxSizeMB < 0 || config.History.MaxSizeMB > maxPoolHistorySizeMB {
		return fmt.Errorf("pool.history.max_size_mb must be between 0 and %d", maxPoolHistorySizeMB)
	}
	if config.History.MaxAgeDays < 0 || config.History.MaxAgeDays > 36500 {
		return fmt.Errorf("pool.history.max_age_days must be between 0 and 36500")
	}
	if !poolConfigOneOf(config.History.Compression, "zstd", "s2", "none") {
		return fmt.Errorf("pool.history.compression must be zstd, s2 or none, got %q", config.History.Compression)
	}
	if !poolConfigOneOf(config.History.CompressionLevel, "fast", "balanced", "best") {
		return fmt.Errorf("pool.history.compression_level must be fast, balanced or best, got %q", config.History.CompressionLevel)
	}
	return nil
}

func poolConfigOneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
