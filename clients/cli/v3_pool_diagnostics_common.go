package cli

import (
	"os"
	"strings"
)

const poolHeapProfileDirEnv = "ANYTTY_POOL_HEAP_PROFILE_DIR"
const poolMemstatsDirEnv = "ANYTTY_POOL_MEMSTATS_DIR"
const poolMemstatsStageEnv = "ANYTTY_DIAG_STAGE"
const poolMemstatsStageFileEnv = "ANYTTY_DIAG_STAGE_FILE"

// Legacy daemon env names stay readable for upgrades; new ANYTTY_POOL_* wins.
const legacyPoolHeapProfileDirEnv = "ANYTTY_DAEMON_HEAP_PROFILE_DIR"
const legacyPoolMemstatsDirEnv = "ANYTTY_DAEMON_MEMSTATS_DIR"

// poolEnv prefers the current ANYTTY_POOL_* name, then the legacy ANYTTY_DAEMON_*.
func poolEnv(primary, legacy string) string {
	if value := strings.TrimSpace(os.Getenv(primary)); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(legacy))
}

func readPoolMemstatsStageFile() string {
	path := strings.TrimSpace(os.Getenv(poolMemstatsStageFileEnv))
	if path == "" {
		return ""
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func poolHeapProfileReason(reason string) string {
	reason = strings.TrimSpace(strings.ToLower(reason))
	if reason == "" {
		return "sample"
	}
	var builder strings.Builder
	for _, r := range reason {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "sample"
	}
	return builder.String()
}
