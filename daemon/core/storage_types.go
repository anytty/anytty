package core

import "time"

// Storage* 类型保留为公共 Proto mapping 的稳定投影。
// Phase 2/3 后 storage.* 真值在 access（access/storage）；daemon 不再持有 store，
// 也不提供 bearer fallback。

// StorageScope 是 storage key 的可见性分区。
type StorageScope string

const (
	// StorageScopePublic 对所有 owner 可见。
	StorageScopePublic StorageScope = "public"
	// StorageScopePrivate 仅对 owner 可见。
	StorageScopePrivate StorageScope = "private"
)

// 存储变更操作类型。
const (
	StorageOpPut    = "put"
	StorageOpDelete = "delete"
)

// StorageEntry 是一次 storage 读取结果投影。
type StorageEntry struct {
	AppID     string
	Scope     StorageScope
	OwnerID   string
	Key       string
	Value     []byte
	Version   uint64
	UpdatedAt time.Time
}

// Clone 返回 value 的深拷贝。
func (entry StorageEntry) Clone() StorageEntry {
	entry.Value = append([]byte(nil), entry.Value...)
	return entry
}

// StoragePutRequest 是一次 CAS put 输入。
type StoragePutRequest struct {
	AppID           string
	Scope           StorageScope
	OwnerID         string
	Key             string
	Value           []byte
	CheckVersion    bool
	ExpectedVersion uint64
}

// StorageDeleteRequest 是一次 CAS delete 输入。
type StorageDeleteRequest struct {
	AppID           string
	Scope           StorageScope
	OwnerID         string
	Key             string
	CheckVersion    bool
	ExpectedVersion uint64
}

// StorageDeleteResult 是 delete 的确定结果。
type StorageDeleteResult struct {
	AppID   string
	Scope   StorageScope
	OwnerID string
	Key     string
	Deleted bool
	Version uint64
}

// StorageChanged 是一次 storage 变更广播投影。
type StorageChanged struct {
	AppID   string
	Scope   StorageScope
	OwnerID string
	Key     string
	Version uint64
	Op      string
}
