// Package sessions 追踪 access owner 建立的 grant 会话，供撤销与过期时实时踢线。
// 它只持有 transport 生命周期，不解释 access wire。
package sessions

import (
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/shared/transport"
)

// Registry 是按 GrantID 索引的活动会话登记表。
// Track 返回 release；CloseGrant 关闭该 grant 的全部活动会话并在之后拒绝新登记。
type Registry struct {
	mu      sync.Mutex
	byGrant map[string]map[*sessionEntry]struct{}
	revoked map[string]struct{}
}

type sessionEntry struct {
	connection transport.Transport
	timer      *time.Timer
}

// NewRegistry 创建空登记表。
func NewRegistry() *Registry {
	return &Registry{byGrant: make(map[string]map[*sessionEntry]struct{}), revoked: make(map[string]struct{})}
}

// Track 登记一条会话并返回释放函数。expiresAt 非零时到期自动关闭。
// 该 grant 已被 CloseGrant 撤销时，连接会被立即关闭。
func (registry *Registry) Track(grantID string, connection transport.Transport, expiresAt time.Time) func() {
	if registry == nil || connection == nil {
		return func() {}
	}
	grantID = strings.TrimSpace(grantID)
	if grantID == "" {
		return func() {}
	}
	entry := &sessionEntry{connection: connection}
	registry.mu.Lock()
	if _, revoked := registry.revoked[grantID]; revoked {
		registry.mu.Unlock()
		_ = connection.Close()
		return func() {}
	}
	group := registry.byGrant[grantID]
	if group == nil {
		group = make(map[*sessionEntry]struct{})
		registry.byGrant[grantID] = group
	}
	group[entry] = struct{}{}
	if !expiresAt.IsZero() {
		delay := time.Until(expiresAt)
		if delay < 0 {
			delay = 0
		}
		entry.timer = time.AfterFunc(delay, func() { registry.closeEntry(grantID, entry) })
	}
	registry.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() { registry.releaseEntry(grantID, entry) })
	}
}

// CloseGrant 关闭一个 grant 的全部活动会话，并阻止后续 Track 重新接入。
// 返回被关闭的会话数。
func (registry *Registry) CloseGrant(grantID string) int {
	if registry == nil {
		return 0
	}
	grantID = strings.TrimSpace(grantID)
	if grantID == "" {
		return 0
	}
	registry.mu.Lock()
	registry.revoked[grantID] = struct{}{}
	group := registry.byGrant[grantID]
	delete(registry.byGrant, grantID)
	entries := make([]*sessionEntry, 0, len(group))
	for entry := range group {
		entries = append(entries, entry)
	}
	registry.mu.Unlock()
	for _, entry := range entries {
		if entry.timer != nil {
			entry.timer.Stop()
		}
		_ = entry.connection.Close()
	}
	return len(entries)
}

// ActiveGrant 报告某个 grant 当前登记的会话数；主要供测试与诊断使用。
func (registry *Registry) ActiveGrant(grantID string) int {
	if registry == nil {
		return 0
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return len(registry.byGrant[strings.TrimSpace(grantID)])
}

// CloseAll 关闭全部活动会话并清空登记；用于进程退出。
func (registry *Registry) CloseAll() {
	if registry == nil {
		return
	}
	registry.mu.Lock()
	entries := make([]*sessionEntry, 0)
	for grantID, group := range registry.byGrant {
		delete(registry.byGrant, grantID)
		for entry := range group {
			entries = append(entries, entry)
		}
	}
	registry.mu.Unlock()
	for _, entry := range entries {
		if entry.timer != nil {
			entry.timer.Stop()
		}
		_ = entry.connection.Close()
	}
}

func (registry *Registry) releaseEntry(grantID string, entry *sessionEntry) {
	registry.mu.Lock()
	if entry.timer != nil {
		entry.timer.Stop()
	}
	if group := registry.byGrant[grantID]; group != nil {
		delete(group, entry)
		if len(group) == 0 {
			delete(registry.byGrant, grantID)
		}
	}
	registry.mu.Unlock()
}

func (registry *Registry) closeEntry(grantID string, entry *sessionEntry) {
	registry.mu.Lock()
	if group := registry.byGrant[grantID]; group != nil {
		delete(group, entry)
		if len(group) == 0 {
			delete(registry.byGrant, grantID)
		}
	}
	registry.mu.Unlock()
	_ = entry.connection.Close()
}
