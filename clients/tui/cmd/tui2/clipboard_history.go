package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const defaultClipboardHistoryMax = 100

// clipboardHistoryEntry is deliberately host-owned. Layout programs receive
// rows for rendering, but clipboard.paste addresses an entry by id so the
// program never supplies arbitrary bytes to a PTY.
type clipboardHistoryEntry struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	CreatedAt int64  `json:"created_at_ms"`
}

type clipboardHistory struct {
	mu      sync.Mutex
	path    string
	max     int
	entries []clipboardHistoryEntry
}

func newClipboardHistory(path string, max int) *clipboardHistory {
	if max <= 0 {
		max = defaultClipboardHistoryMax
	}
	h := &clipboardHistory{path: path, max: max}
	h.load()
	return h
}

func defaultClipboardHistoryPath() string {
	stateHome := strings.TrimSpace(os.Getenv("XDG_STATE_HOME"))
	if stateHome == "" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			stateHome = filepath.Join(home, ".local", "state")
		}
	}
	if stateHome == "" {
		stateHome = os.TempDir()
	}
	return filepath.Join(stateHome, "anytty", "tui2", "clipboard-history.json")
}

func (h *clipboardHistory) load() {
	if h == nil || h.path == "" {
		return
	}
	payload, err := os.ReadFile(h.path)
	if err != nil {
		return
	}
	var entries []clipboardHistoryEntry
	if json.Unmarshal(payload, &entries) != nil {
		return
	}
	for _, entry := range entries {
		if entry.ID != "" && entry.Text != "" {
			h.entries = append(h.entries, entry)
		}
		if len(h.entries) >= h.max {
			break
		}
	}
}

func (h *clipboardHistory) saveLocked() error {
	if h == nil || h.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(h.entries, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(h.path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(h.path), ".clipboard-history-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, h.path)
}

func newClipboardID() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err == nil {
		return "clip-" + hex.EncodeToString(raw)
	}
	return fmt.Sprintf("clip-%d", time.Now().UnixNano())
}

func (h *clipboardHistory) Add(text string) (clipboardHistoryEntry, error) {
	text = strings.ReplaceAll(text, "\x00", "")
	if text == "" {
		return clipboardHistoryEntry{}, nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.entries {
		if h.entries[i].Text == text {
			entry := h.entries[i]
			entry.CreatedAt = time.Now().UnixMilli()
			copy(h.entries[1:i+1], h.entries[0:i])
			h.entries[0] = entry
			return entry, h.saveLocked()
		}
	}
	entry := clipboardHistoryEntry{ID: newClipboardID(), Text: text, CreatedAt: time.Now().UnixMilli()}
	h.entries = append([]clipboardHistoryEntry{entry}, h.entries...)
	if len(h.entries) > h.max {
		h.entries = h.entries[:h.max]
	}
	return entry, h.saveLocked()
}

func (h *clipboardHistory) List() []clipboardHistoryEntry {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]clipboardHistoryEntry(nil), h.entries...)
}

func (h *clipboardHistory) Get(id string) (clipboardHistoryEntry, bool) {
	if h == nil || id == "" {
		return clipboardHistoryEntry{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, entry := range h.entries {
		if entry.ID == id {
			return entry, true
		}
	}
	return clipboardHistoryEntry{}, false
}

func (h *clipboardHistory) Delete(id string) error {
	if h == nil || id == "" {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, entry := range h.entries {
		if entry.ID == id {
			h.entries = append(h.entries[:i], h.entries[i+1:]...)
			return h.saveLocked()
		}
	}
	return nil
}

func (h *clipboardHistory) Rows() []string {
	entries := h.List()
	rows := make([]string, 0, len(entries))
	for _, entry := range entries {
		data, _ := json.Marshal(entry)
		rows = append(rows, string(data))
	}
	return rows
}

// readSystemClipboard uses the platform's native text clipboard command. The
// host can replace it through Options.ClipboardRead in tests or embedders.
func readSystemClipboard() (string, error) {
	commands := make([][]string, 0, 3)
	switch runtime.GOOS {
	case "darwin":
		commands = append(commands, []string{"pbpaste"})
	case "windows":
		commands = append(commands, []string{"powershell", "-NoProfile", "-Command", "Get-Clipboard"})
	default:
		commands = append(commands,
			[]string{"wl-paste", "--no-newline"},
			[]string{"xclip", "-selection", "clipboard", "-out"},
			[]string{"xsel", "--clipboard", "--output"},
		)
	}
	var lastErr error
	for _, argv := range commands {
		if _, err := exec.LookPath(argv[0]); err != nil {
			lastErr = err
			continue
		}
		out, err := exec.Command(argv[0], argv[1:]...).Output()
		if err == nil {
			return string(out), nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("system clipboard is unavailable")
	}
	return "", lastErr
}
