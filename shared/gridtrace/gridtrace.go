package gridtrace

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const EnvPath = "ANYTTY_GRID_HISTORY_TRACE"

var (
	mu          sync.Mutex
	eventCounts = map[string]int{}
)

func Enabled() bool {
	return tracePath() != ""
}

func Log(event string, kv ...any) {
	write(event, 0, kv...)
}

func LogLimited(event string, limit int, kv ...any) {
	write(event, limit, kv...)
}

func Short(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	if limit <= 3 {
		return s[:limit]
	}
	return s[:limit-3] + "..."
}

func write(event string, limit int, kv ...any) {
	path := tracePath()
	if path == "" || strings.TrimSpace(event) == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if limit > 0 {
		eventCounts[event]++
		if eventCounts[event] > limit {
			return
		}
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0o700)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	var b strings.Builder
	b.WriteString("ts=")
	b.WriteString(strconv.Quote(time.Now().UTC().Format(time.RFC3339Nano)))
	b.WriteString(" event=")
	b.WriteString(strconv.Quote(event))
	for i := 0; i < len(kv); i += 2 {
		key := fmt.Sprint(kv[i])
		if key == "" {
			continue
		}
		value := "<missing>"
		if i+1 < len(kv) {
			value = fmt.Sprint(kv[i+1])
		}
		b.WriteByte(' ')
		b.WriteString(sanitizeKey(key))
		b.WriteByte('=')
		b.WriteString(strconv.Quote(Short(value, 512)))
	}
	b.WriteByte('\n')
	_, _ = f.WriteString(b.String())
}

func tracePath() string {
	v := strings.TrimSpace(os.Getenv(EnvPath))
	if v == "" || strings.EqualFold(v, "0") || strings.EqualFold(v, "false") || strings.EqualFold(v, "off") || strings.EqualFold(v, "no") {
		return ""
	}
	switch strings.ToLower(v) {
	case "1", "true", "on", "yes", "debug":
		return filepath.Join(os.TempDir(), "anytty-grid-history-trace.log")
	default:
		return v
	}
}

func sanitizeKey(key string) string {
	var b strings.Builder
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	if b.Len() == 0 {
		return "key"
	}
	return b.String()
}
