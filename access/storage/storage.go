// Package storage 是 access-local 的通用 opaque KV，实现客户端 storage.* 命令。
//
// 它从 daemon/core 迁移而来（Phase 2）：同一进程内共享一份真值，按
// app/scope/owner/key 分区，值是不透明的 []byte，version 单调递增用于 CAS。
// 它不解释 value 语义，也不做鉴权：调用方（access/server 路由层）已完成
// 本地信任边界或 remoteauth。
package storage

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// Scope 是 storage key 的可见性分区。
type Scope string

const (
	// ScopePublic 对所有 owner 可见。
	ScopePublic Scope = "public"
	// ScopePrivate 仅对 owner 可见。
	ScopePrivate Scope = "private"
)

// 存储变更操作类型。
const (
	OpPut    = "put"
	OpDelete = "delete"
)

var (
	// ErrEntryNotFound 表示 key 不存在。
	ErrEntryNotFound = errors.New("storage entry was not found")
	// ErrInvalidKey 表示 app_id / key 缺失。
	ErrInvalidKey = errors.New("storage key is invalid")
	// ErrVersionConflict 表示 CAS 期望版本不匹配。
	ErrVersionConflict = errors.New("storage version conflict")
)

// Entry 是一次 storage 读取结果。
type Entry struct {
	AppID     string
	Scope     Scope
	OwnerID   string
	Key       string
	Value     []byte
	Version   uint64
	UpdatedAt time.Time
}

// Clone 返回 value 的深拷贝。
func (entry Entry) Clone() Entry {
	entry.Value = append([]byte(nil), entry.Value...)
	return entry
}

// PutRequest 是一次 CAS put 输入。
type PutRequest struct {
	AppID           string
	Scope           Scope
	OwnerID         string
	Key             string
	Value           []byte
	CheckVersion    bool
	ExpectedVersion uint64
}

// DeleteRequest 是一次 CAS delete 输入。
type DeleteRequest struct {
	AppID           string
	Scope           Scope
	OwnerID         string
	Key             string
	CheckVersion    bool
	ExpectedVersion uint64
}

// DeleteResult 是 delete 的确定性结果。
type DeleteResult struct {
	AppID   string
	Scope   Scope
	OwnerID string
	Key     string
	Deleted bool
	Version uint64
}

// Change 是一次 storage 变更广播。
type Change struct {
	AppID   string
	Scope   Scope
	OwnerID string
	Key     string
	Version uint64
	Op      string
}

type storageKey struct {
	appID   string
	scope   Scope
	ownerID string
	key     string
}

// Store 是一个进程内 storage 真值 owner。所有方法并发安全。
type Store struct {
	mu          sync.RWMutex
	entries     map[storageKey]Entry
	subscribers map[uint64]chan Change
	nextSub     uint64
}

// New 创建空 store。
func New() *Store {
	return &Store{
		entries:     make(map[storageKey]Entry),
		subscribers: make(map[uint64]chan Change),
	}
}

// Get 读取一个 entry；不存在时返回 ErrEntryNotFound。
func (store *Store) Get(appID string, scope Scope, ownerID string, key string) (Entry, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	entry, ok := store.entries[makeKey(appID, scope, ownerID, key)]
	if !ok {
		return Entry{}, ErrEntryNotFound
	}
	return entry.Clone(), nil
}

// Put 执行一次 CAS put 并广播变更。
func (store *Store) Put(request PutRequest) (Entry, error) {
	key := makeKey(request.AppID, request.Scope, request.OwnerID, request.Key)
	if key.appID == "" || key.key == "" {
		return Entry{}, ErrInvalidKey
	}
	now := time.Now().UTC()
	store.mu.Lock()
	current, exists := store.entries[key]
	if request.CheckVersion {
		currentVersion := uint64(0)
		if exists {
			currentVersion = current.Version
		}
		if currentVersion != request.ExpectedVersion {
			store.mu.Unlock()
			return Entry{}, ErrVersionConflict
		}
	}
	entry := Entry{
		AppID:     key.appID,
		Scope:     key.scope,
		OwnerID:   key.ownerID,
		Key:       key.key,
		Value:     append([]byte(nil), request.Value...),
		Version:   current.Version + 1,
		UpdatedAt: now,
	}
	if !exists {
		entry.Version = 1
	}
	store.entries[key] = entry.Clone()
	change := Change{AppID: key.appID, Scope: key.scope, OwnerID: key.ownerID, Key: key.key, Version: entry.Version, Op: OpPut}
	store.mu.Unlock()
	store.publish(change)
	return entry, nil
}

// Delete 执行一次 CAS delete 并广播变更；不存在时按幂等成功处理。
func (store *Store) Delete(request DeleteRequest) (DeleteResult, error) {
	key := makeKey(request.AppID, request.Scope, request.OwnerID, request.Key)
	if key.appID == "" || key.key == "" {
		return DeleteResult{}, ErrInvalidKey
	}
	store.mu.Lock()
	current, exists := store.entries[key]
	currentVersion := uint64(0)
	if exists {
		currentVersion = current.Version
	}
	if request.CheckVersion && currentVersion != request.ExpectedVersion {
		store.mu.Unlock()
		return DeleteResult{}, ErrVersionConflict
	}
	if exists {
		delete(store.entries, key)
		currentVersion++
	}
	result := DeleteResult{
		AppID:   key.appID,
		Scope:   key.scope,
		OwnerID: key.ownerID,
		Key:     key.key,
		Deleted: exists,
		Version: currentVersion,
	}
	store.mu.Unlock()
	if exists {
		store.publish(Change{AppID: key.appID, Scope: key.scope, OwnerID: key.ownerID, Key: key.key, Version: currentVersion, Op: OpDelete})
	}
	return result, nil
}

// List 返回按 key 排序的稳定窗口；它不排序 value，也不解释 key 语义。
func (store *Store) List(appID string, scope Scope, ownerID string, prefix string) []Entry {
	key := makeKey(appID, scope, ownerID, "")
	store.mu.RLock()
	defer store.mu.RUnlock()
	out := make([]Entry, 0)
	for candidate, entry := range store.entries {
		if candidate.appID != key.appID || candidate.scope != key.scope || candidate.ownerID != key.ownerID {
			continue
		}
		if prefix != "" && !strings.HasPrefix(candidate.key, prefix) {
			continue
		}
		out = append(out, entry.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Subscribe 返回变更广播；ctx 取消时自动退订。
// channel 不关闭：consumer 必须同时 select ctx（避免与 publish 竞争），
// 慢消费者会被跳过（事件是通知，不是 durable log）。
func (store *Store) Subscribe(ctx context.Context) <-chan Change {
	if ctx == nil {
		ctx = context.Background()
	}
	out := make(chan Change, 64)
	store.mu.Lock()
	store.nextSub++
	id := store.nextSub
	store.subscribers[id] = out
	store.mu.Unlock()
	go func() {
		<-ctx.Done()
		store.mu.Lock()
		delete(store.subscribers, id)
		store.mu.Unlock()
	}()
	return out
}

func (store *Store) publish(change Change) {
	store.mu.RLock()
	channels := make([]chan Change, 0, len(store.subscribers))
	for _, channel := range store.subscribers {
		channels = append(channels, channel)
	}
	store.mu.RUnlock()
	for _, channel := range channels {
		select {
		case channel <- change:
		default:
		}
	}
}

func makeKey(appID string, scope Scope, ownerID string, key string) storageKey {
	if scope == "" {
		scope = ScopePublic
	}
	return storageKey{
		appID:   strings.TrimSpace(appID),
		scope:   scope,
		ownerID: strings.TrimSpace(ownerID),
		key:     strings.TrimSpace(key),
	}
}
