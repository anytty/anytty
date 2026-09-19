// Package transfer 实现 access 文件传输的断点续传记录、窗口策略与进度合并。
//
// 它是 §11 的 access-side 基础：transfer 记录跨会话/进程持久化，续传 token
// 是不透明 bearer secret，并绑定目标路径、大小与 owner；窗口/分片按链路自适应；
// 进度按时间窗合并，避免高频小帧。文件内容始终流式处理，不整文件驻留内存。
package transfer

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/anytty/anytty/shared/filepublish"
)

// TokenBytes 是续传 bearer token 的长度。
const TokenBytes = 32

// Direction 是 transfer 的方向。
type Direction string

const (
	// DirectionUpload 表示客户端→主机。
	DirectionUpload Direction = "upload"
	// DirectionDownload 表示主机→客户端。
	DirectionDownload Direction = "download"
)

var (
	// ErrRecordInvalid 表示记录缺少 ID/token/路径等必要字段。
	ErrRecordInvalid = errors.New("transfer record is invalid")
	// ErrStoreCorrupt 表示持久化文件不可解析；调用方必须 fail closed，不能静默丢续传记录。
	ErrStoreCorrupt = errors.New("transfer store is corrupt")
)

// Record 是一条可跨会话/进程恢复的传输记录。
// Token 是不透明 bearer secret；Path/Size/OwnerID 绑定恢复目标。
type Record struct {
	ID               string
	Token            []byte
	Direction        Direction
	Path             string
	OwnerID          string
	Size             int64
	Offset           int64
	TargetSize       int64
	TargetModifiedAt time.Time
	Overwrite        bool
	HashState        []byte
	TempPath         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ExpiresAt        time.Time
}

// Clone 深拷贝 token/hash state，避免调用方共享底层数组。
func (record Record) Clone() Record {
	record.Token = append([]byte(nil), record.Token...)
	record.HashState = append([]byte(nil), record.HashState...)
	return record
}

// MatchesTarget 校验续传目标仍与记录一致：大小与 mtime 任一变化都拒绝恢复。
// expectedModifiedAt 为零表示调用方不检查 mtime。
func (record Record) MatchesTarget(size int64, modifiedAt time.Time) bool {
	if record.TargetSize != size {
		return false
	}
	if record.TargetModifiedAt.IsZero() || modifiedAt.IsZero() {
		return true
	}
	return record.TargetModifiedAt.Equal(modifiedAt)
}

// Expired 报告记录是否已过期。
func (record Record) Expired(now time.Time) bool {
	return !record.ExpiresAt.IsZero() && !now.Before(record.ExpiresAt)
}

// NewToken 生成一个不透明、不可猜测的 bearer token。
func NewToken() ([]byte, error) {
	token := make([]byte, TokenBytes)
	if _, err := rand.Read(token); err != nil {
		return nil, fmt.Errorf("transfer: allocate resume token: %w", err)
	}
	return token, nil
}

// Store 是 access-side 持久化 transfer 记录表。所有方法并发安全。
type Store struct {
	mu      sync.Mutex
	path    string
	records map[string]Record
	byToken map[string]string
}

type persistedStore struct {
	Version int      `json:"version"`
	Records []Record `json:"records"`
}

const storeVersion = 1

// Open 打开（或创建）dir 下的持久化 transfer 记录表。
// 目录不存在时创建为 0700；文件为 0600。
func Open(dir string) (*Store, error) {
	trimmed := filepath.Clean(dir)
	if trimmed == "" || trimmed == "." {
		return nil, fmt.Errorf("%w: store dir is required", ErrRecordInvalid)
	}
	if err := os.MkdirAll(trimmed, 0o700); err != nil {
		return nil, fmt.Errorf("transfer: create store dir: %w", err)
	}
	store := &Store{
		path:    filepath.Join(trimmed, "transfers.json"),
		records: make(map[string]Record),
		byToken: make(map[string]string),
	}
	payload, err := os.ReadFile(store.path)
	switch {
	case err == nil:
		var persisted persistedStore
		if err := json.Unmarshal(payload, &persisted); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStoreCorrupt, err)
		}
		if persisted.Version != storeVersion {
			return nil, fmt.Errorf("%w: unsupported version %d", ErrStoreCorrupt, persisted.Version)
		}
		for _, record := range persisted.Records {
			if record.ID == "" || len(record.Token) == 0 {
				continue
			}
			store.records[record.ID] = record.Clone()
			store.byToken[string(record.Token)] = record.ID
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return nil, fmt.Errorf("transfer: read store: %w", err)
	}
	return store, nil
}

// Put 插入或更新记录并同步落盘。
func (store *Store) Put(record Record) error {
	if store == nil {
		return fmt.Errorf("%w: store is nil", ErrRecordInvalid)
	}
	if record.ID == "" || len(record.Token) < TokenBytes || record.Path == "" {
		return fmt.Errorf("%w: id, token and path are required", ErrRecordInvalid)
	}
	if record.Direction != DirectionUpload && record.Direction != DirectionDownload {
		return fmt.Errorf("%w: unsupported direction %q", ErrRecordInvalid, record.Direction)
	}
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	store.mu.Lock()
	defer store.mu.Unlock()
	store.records[record.ID] = record.Clone()
	store.byToken[string(record.Token)] = record.ID
	return store.persistLocked()
}

// Lookup 按 bearer token 查找记录。
func (store *Store) Lookup(token []byte) (Record, bool) {
	if store == nil || len(token) == 0 {
		return Record{}, false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	id, ok := store.byToken[string(token)]
	if !ok {
		return Record{}, false
	}
	record, ok := store.records[id]
	if !ok {
		return Record{}, false
	}
	return record.Clone(), true
}

// Get 按 ID 查找记录。
func (store *Store) Get(id string) (Record, bool) {
	if store == nil {
		return Record{}, false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	record, ok := store.records[id]
	if !ok {
		return Record{}, false
	}
	return record.Clone(), true
}

// Delete 删除记录并落盘；不存在时幂等成功。
func (store *Store) Delete(id string) error {
	if store == nil {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	record, ok := store.records[id]
	if !ok {
		return nil
	}
	delete(store.records, id)
	delete(store.byToken, string(record.Token))
	return store.persistLocked()
}

// DeleteByToken 按 bearer token 删除记录；不存在时幂等成功。
func (store *Store) DeleteByToken(token []byte) error {
	if store == nil || len(token) == 0 {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	id, ok := store.byToken[string(token)]
	if !ok {
		return nil
	}
	record := store.records[id]
	delete(store.records, id)
	delete(store.byToken, string(record.Token))
	return store.persistLocked()
}

// Prune 删除过期记录并落盘，返回删除数量。
func (store *Store) Prune(now time.Time) int {
	if store == nil {
		return 0
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	removed := 0
	for id, record := range store.records {
		if !record.Expired(now) {
			continue
		}
		delete(store.records, id)
		delete(store.byToken, string(record.Token))
		removed++
	}
	if removed > 0 {
		_ = store.persistLocked()
	}
	return removed
}

// Path 返回持久化文件路径（诊断用）。
func (store *Store) Path() string {
	if store == nil {
		return ""
	}
	return store.path
}

// All 返回全部记录的深拷贝快照；调用方负责过期清理与临时文件回收。
func (store *Store) All() []Record {
	if store == nil {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	out := make([]Record, 0, len(store.records))
	for _, record := range store.records {
		out = append(out, record.Clone())
	}
	return out
}

// Len 返回当前记录数；主要供测试与诊断。
func (store *Store) Len() int {
	if store == nil {
		return 0
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.records)
}

func (store *Store) persistLocked() error {
	persisted := persistedStore{Version: storeVersion}
	for _, record := range store.records {
		persisted.Records = append(persisted.Records, record.Clone())
	}
	payload, err := json.Marshal(persisted)
	if err != nil {
		return fmt.Errorf("transfer: encode store: %w", err)
	}
	temporary := store.path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("transfer: create store temp: %w", err)
	}
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		_ = os.Remove(temporary)
		return fmt.Errorf("transfer: write store: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(temporary)
		return fmt.Errorf("transfer: sync store: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("transfer: close store: %w", err)
	}
	if err := filepublish.Rename(temporary, store.path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("transfer: publish store: %w", err)
	}
	if err := filepublish.SyncDirectory(filepath.Dir(store.path)); err != nil {
		return fmt.Errorf("transfer: sync store dir: %w", err)
	}
	return nil
}
