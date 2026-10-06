package endpoint

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestOfflineNoticeIsNotRepeated pins the dedupe: a paired-but-unreachable
// endpoint is retried on every backoff (offline -> connecting -> offline), and
// the user must see one warning, not one per retry.
func TestOfflineNoticeIsNotRepeated(t *testing.T) {
	var mu sync.Mutex
	var notices []string
	m := NewManager(Options{
		DialTimeout:  50 * time.Millisecond,
		CallTimeout:  50 * time.Millisecond,
		BackoffMin:   10 * time.Millisecond,
		BackoffMax:   20 * time.Millisecond,
		RegistryPath: t.TempDir() + "/none.yaml",
		Dial: func(context.Context, Config) (sessionConn, error) {
			return nil, errors.New("no eligible route for the current platform")
		},
		OnNotice: func(level, message string) {
			mu.Lock()
			notices = append(notices, level+": "+message)
			mu.Unlock()
		},
	})
	t.Cleanup(func() { _ = m.Close() })
	if err := m.Register(Config{Name: "dev", Kind: KindDaemon, Socket: t.TempDir() + "/dev.sock"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(notices) == 0 {
		t.Fatal("no offline notice was emitted")
	}
	if len(notices) != 1 {
		t.Fatalf("offline notice repeated %d times: %v", len(notices), notices)
	}
	if !strings.Contains(notices[0], "offline") || !strings.Contains(notices[0], "no eligible route") {
		t.Fatalf("notice = %q", notices[0])
	}
}
