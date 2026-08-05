package main

import (
	"os"
	"strings"
)

const daemonHeapProfileDirEnv = "ANYTTY_DAEMON_HEAP_PROFILE_DIR"
const daemonMemstatsDirEnv = "ANYTTY_DAEMON_MEMSTATS_DIR"
const daemonMemstatsStageEnv = "ANYTTY_DIAG_STAGE"
const daemonMemstatsStageFileEnv = "ANYTTY_DIAG_STAGE_FILE"

func readDaemonMemstatsStageFile() string {
	path := strings.TrimSpace(os.Getenv(daemonMemstatsStageFileEnv))
	if path == "" {
		return ""
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func daemonHeapProfileReason(reason string) string {
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
