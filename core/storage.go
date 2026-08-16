package core

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type StorageScope string

const (
	StorageScopePublic  StorageScope = "public"
	StorageScopePrivate StorageScope = "private"
)

const (
	StorageOpPut    = "put"
	StorageOpDelete = "delete"
)

type StorageEntry struct {
	AppID     string
	Scope     StorageScope
	OwnerID   string
	Key       string
	Value     []byte
	Version   uint64
	UpdatedAt time.Time
}

func (entry StorageEntry) Clone() StorageEntry {
	entry.Value = append([]byte(nil), entry.Value...)
	return entry
}

type StoragePutRequest struct {
	AppID           string
	Scope           StorageScope
	OwnerID         string
	Key             string
	Value           []byte
	CheckVersion    bool
	ExpectedVersion uint64
}

type StorageDeleteRequest struct {
	AppID           string
	Scope           StorageScope
	OwnerID         string
	Key             string
	CheckVersion    bool
	ExpectedVersion uint64
}

type StorageDeleteResult struct {
	AppID   string
	Scope   StorageScope
	OwnerID string
	Key     string
	Deleted bool
	Version uint64
}

type StorageChanged struct {
	AppID   string
	Scope   StorageScope
	OwnerID string
	Key     string
	Version uint64
	Op      string
}

type storageStore struct {
	mu      sync.RWMutex
	entries map[storageKey]StorageEntry
}

type storageKey struct {
	appID   string
	scope   StorageScope
	ownerID string
	key     string
}

func newStorageStore() *storageStore {
	return &storageStore{entries: make(map[storageKey]StorageEntry)}
}

func (store *storageStore) get(appID string, scope StorageScope, ownerID string, key string) (StorageEntry, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	entry, ok := store.entries[makeStorageKey(appID, scope, ownerID, key)]
	if !ok {
		return StorageEntry{}, ErrStorageEntryNotFound
	}
	return entry.Clone(), nil
}

func (store *storageStore) put(request StoragePutRequest) (StorageEntry, error) {
	key := makeStorageKey(request.AppID, request.Scope, request.OwnerID, request.Key)
	if key.appID == "" || key.key == "" {
		return StorageEntry{}, ErrInvalidStorageKey
	}
	now := time.Now().UTC()
	store.mu.Lock()
	defer store.mu.Unlock()
	current, exists := store.entries[key]
	if request.CheckVersion {
		currentVersion := uint64(0)
		if exists {
			currentVersion = current.Version
		}
		if currentVersion != request.ExpectedVersion {
			return StorageEntry{}, ErrStorageVersionConflict
		}
	}
	entry := StorageEntry{
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
	return entry, nil
}

func (store *storageStore) delete(request StorageDeleteRequest) (StorageDeleteResult, error) {
	key := makeStorageKey(request.AppID, request.Scope, request.OwnerID, request.Key)
	if key.appID == "" || key.key == "" {
		return StorageDeleteResult{}, ErrInvalidStorageKey
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	current, exists := store.entries[key]
	currentVersion := uint64(0)
	if exists {
		currentVersion = current.Version
	}
	if request.CheckVersion && currentVersion != request.ExpectedVersion {
		return StorageDeleteResult{}, ErrStorageVersionConflict
	}
	if exists {
		delete(store.entries, key)
		currentVersion++
	}
	return StorageDeleteResult{
		AppID:   key.appID,
		Scope:   key.scope,
		OwnerID: key.ownerID,
		Key:     key.key,
		Deleted: exists,
		Version: currentVersion,
	}, nil
}

func (store *storageStore) list(appID string, scope StorageScope, ownerID string, prefix string) []StorageEntry {
	key := makeStorageKey(appID, scope, ownerID, "")
	store.mu.RLock()
	defer store.mu.RUnlock()
	out := make([]StorageEntry, 0)
	for candidate, entry := range store.entries {
		if candidate.appID != key.appID || candidate.scope != key.scope || candidate.ownerID != key.ownerID {
			continue
		}
		if prefix != "" && !strings.HasPrefix(candidate.key, prefix) {
			continue
		}
		out = append(out, entry.Clone())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out
}

func makeStorageKey(appID string, scope StorageScope, ownerID string, key string) storageKey {
	if scope == "" {
		scope = StorageScopePublic
	}
	return storageKey{
		appID:   strings.TrimSpace(appID),
		scope:   scope,
		ownerID: strings.TrimSpace(ownerID),
		key:     strings.TrimSpace(key),
	}
}
