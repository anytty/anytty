package agents

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed opencode.mjs
var openCodeSource []byte

const managedMarker = "anytty-managed-agent-hook-v1"

var codexEvents = []string{"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse", "PermissionRequest", "Stop", "Interrupt", "SessionEnd"}

// InstallCodex preserves unrelated source fields, matcher groups and handlers.
// It does not alter the Codex trust database: users review new hooks via /hooks.
func InstallCodex(configDir, executable string) error { return editCodex(configDir, executable, true) }
func UninstallCodex(configDir string) error           { return editCodex(configDir, "", false) }

func editCodex(dir, executable string, install bool) error {
	if dir == "" {
		return errors.New("explicit Codex config directory is required")
	}
	if install && !filepath.IsAbs(executable) {
		return errors.New("hook executable must be an absolute path")
	}
	path := filepath.Join(dir, "hooks.json")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if os.IsNotExist(err) && !install {
		return nil
	}
	root := map[string]json.RawMessage{}
	if len(data) > 0 {
		if err = json.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("read hooks.json: %w", err)
		}
	}
	if root == nil {
		return errors.New("hooks.json must be an object")
	}
	hooks := map[string][]map[string]json.RawMessage{}
	if raw, ok := root["hooks"]; ok {
		if err = json.Unmarshal(raw, &hooks); err != nil {
			return fmt.Errorf("read hooks: %w", err)
		}
		if hooks == nil {
			return errors.New("hooks must be an object")
		}
	}
	for event, groups := range hooks {
		kept := make([]map[string]json.RawMessage, 0, len(groups))
		for _, group := range groups {
			var handlers []map[string]json.RawMessage
			if err = json.Unmarshal(group["hooks"], &handlers); err != nil {
				return fmt.Errorf("read %s hook handlers: %w", event, err)
			}
			clean := make([]map[string]json.RawMessage, 0, len(handlers))
			for _, handler := range handlers {
				var status string
				_ = json.Unmarshal(handler["statusMessage"], &status)
				if status != managedMarker {
					clean = append(clean, handler)
				}
			}
			if len(clean) > 0 {
				group["hooks"], _ = json.Marshal(clean)
				kept = append(kept, group)
			}
		}
		if len(kept) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = kept
		}
	}
	if install {
		command := "'" + strings.ReplaceAll(executable, "'", "'\"'\"'") + "' plugin agents hook codex"
		for _, event := range codexEvents {
			handler, _ := json.Marshal([]map[string]any{{"type": "command", "command": command, "timeout": 3, "statusMessage": managedMarker}})
			hooks[event] = append(hooks[event], map[string]json.RawMessage{"hooks": handler})
		}
	}
	root["hooks"], _ = json.Marshal(hooks)
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(out, '\n'), 0600)
}

func InstallOpenCode(configDir, executable string) error {
	if configDir == "" || !filepath.IsAbs(executable) {
		return errors.New("explicit OpenCode config directory and absolute executable required")
	}
	path := filepath.Join(configDir, "plugins", "anytty-agents.js")
	old, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && !bytes.HasPrefix(old, []byte("// "+managedMarker+"\n")) {
		return errors.New("refusing to replace unmanaged OpenCode plugin")
	}
	encoded, _ := json.Marshal(executable)
	source := bytes.Replace(openCodeSource, []byte("\"__ANYTTY_EXECUTABLE__\""), encoded, 1)
	return atomicWrite(path, source, 0600)
}

func UninstallOpenCode(configDir string) error {
	if configDir == "" {
		return errors.New("explicit OpenCode config directory required")
	}
	path := filepath.Join(configDir, "plugins", "anytty-agents.js")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !bytes.HasPrefix(data, []byte("// "+managedMarker+"\n")) {
		return errors.New("refusing to remove unmanaged OpenCode plugin")
	}
	return os.Remove(path)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("configuration target is not a regular file")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".anytty-hooks-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}
