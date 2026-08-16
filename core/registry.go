package core

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type terminalRegistry struct {
	mu        sync.RWMutex
	terminals map[string]TerminalInfo
}

func newTerminalRegistry() *terminalRegistry {
	return &terminalRegistry{terminals: make(map[string]TerminalInfo)}
}

func (registry *terminalRegistry) register(record TerminalRecord, defaultSize Size) (TerminalInfo, error) {
	id := strings.TrimSpace(record.ID)
	if id == "" {
		return TerminalInfo{}, ErrInvalidTerminalID
	}
	if len(record.Command) == 0 {
		return TerminalInfo{}, ErrInvalidCommand
	}
	size := record.Size
	if !size.Valid() {
		size = defaultSize
	}
	if !size.Valid() {
		return TerminalInfo{}, ErrInvalidServerSize
	}
	name := strings.TrimSpace(record.Name)
	if name == "" {
		name = id
	}
	info := TerminalInfo{
		ID:        id,
		Name:      name,
		Command:   append([]string(nil), record.Command...),
		Tags:      cloneStringMap(record.Tags),
		Size:      size,
		State:     TerminalStateRunning,
		CWD:       record.Options.Dir,
		LiveCWD:   record.Options.Dir,
		CreatedAt: time.Now().UTC(),
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.terminals[id]; exists {
		return TerminalInfo{}, ErrDuplicateTerminal
	}
	if registry.nameExistsLocked("", name) {
		return TerminalInfo{}, ErrDuplicateTerminal
	}
	registry.terminals[id] = info.Clone()
	return info, nil
}

func (registry *terminalRegistry) get(id string) (TerminalInfo, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	info, ok := registry.terminals[id]
	if !ok {
		return TerminalInfo{}, ErrTerminalNotFound
	}
	return info.Clone(), nil
}

func (registry *terminalRegistry) list() []TerminalInfo {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	out := make([]TerminalInfo, 0, len(registry.terminals))
	for _, info := range registry.terminals {
		out = append(out, info.Clone())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (registry *terminalRegistry) remove(id string) (TerminalInfo, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	info, ok := registry.terminals[id]
	if !ok {
		return TerminalInfo{}, ErrTerminalNotFound
	}
	delete(registry.terminals, id)
	info.State = TerminalStateRemoved
	return info.Clone(), nil
}

func (registry *terminalRegistry) replace(info TerminalInfo) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, ok := registry.terminals[info.ID]; !ok {
		return ErrTerminalNotFound
	}
	if registry.nameExistsLocked(info.ID, info.Name) {
		return ErrDuplicateTerminal
	}
	registry.terminals[info.ID] = info.Clone()
	return nil
}

func (registry *terminalRegistry) setMetadata(id string, name string, tags map[string]string) (TerminalInfo, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	info, ok := registry.terminals[id]
	if !ok {
		return TerminalInfo{}, ErrTerminalNotFound
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = id
	}
	// 中文说明：ME009 起 terminal name 是 daemon-local create/binding key；
	// registry 是同一 daemon 内 terminal identity 的 truth source，metadata
	// rename 也必须在这里拒绝重名，不能只在 TUI picker 层拦截。
	if registry.nameExistsLocked(id, name) {
		return TerminalInfo{}, ErrDuplicateTerminal
	}
	info.Name = name
	info.Tags = cloneStringMap(tags)
	registry.terminals[id] = info.Clone()
	return info.Clone(), nil
}

func (registry *terminalRegistry) nameExistsLocked(excludeID string, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for id, info := range registry.terminals {
		if id == excludeID {
			continue
		}
		if strings.TrimSpace(info.Name) == name {
			return true
		}
	}
	return false
}

func (registry *terminalRegistry) clear() {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.terminals = make(map[string]TerminalInfo)
}
