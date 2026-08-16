package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

const (
	ClipboardStorageAppID       = WorkbenchStorageAppID
	ClipboardStorageScopePublic = WorkbenchStorageScopePublic
	ClipboardStorageKeyRoot     = "clipboard/history"
	ClipboardStorageSchema      = "anytty.tui.v3.clipboard-history"
	ClipboardStorageSchemaV2    = 2

	DefaultClipboardHistoryMaxItems = 200
	MaxClipboardHistoryBytes        = 1 << 20
	MaxClipboardHistoryEntryBytes   = 256 << 10
	clipboardEntrySummaryRunes      = 72
)

var ErrInvalidClipboardSnapshot = errors.New("invalid clipboard snapshot")

// ClipboardStore 保存 reducer-owned clipboard 历史。
// 它只保留当前客户端的复制记录，不直接负责系统 clipboard IO。
type ClipboardStore struct {
	Entries            []ClipboardEntry
	LastSavedVersion   uint64
	LastAppliedVersion uint64
	LastEventVersion   uint64
	BaseVersion        uint64
	ConflictVersion    uint64
	Conflict           bool
	Dirty              bool
	DirtyMergeable     bool
}

type ClipboardEntry struct {
	ID      string
	Title   string
	Text    string
	Preview string
}

type ClipboardStorageRef struct {
	AppID   string
	Scope   string
	OwnerID string
	Key     string
	Version uint64
}

type ClipboardStorageSnapshot struct {
	Schema        string           `json:"schema"`
	SchemaVersion int              `json:"schemaVersion"`
	Entries       []ClipboardEntry `json:"entries"`
}

func DefaultClipboardStorageRef(ownerID string) ClipboardStorageRef {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		ownerID = DefaultWorkspaceID
	}
	return ClipboardStorageRef{
		AppID:   ClipboardStorageAppID,
		Scope:   ClipboardStorageScopePublic,
		OwnerID: ownerID,
		Key:     ClipboardStorageKeyRoot,
	}
}

func (ref ClipboardStorageRef) WithVersion(version uint64) ClipboardStorageRef {
	ref.Version = version
	return ref
}

func (ref ClipboardStorageRef) KeyPrefix() string {
	key := strings.TrimSpace(ref.Key)
	if key == "" {
		return "clipboard/"
	}
	if strings.HasSuffix(key, "/") {
		return key
	}
	if slash := strings.LastIndex(key, "/"); slash >= 0 {
		return key[:slash+1]
	}
	return key
}

func (store ClipboardStore) WithCopiedText(text string) ClipboardStore {
	store, _ = store.WithCopiedTextLimit(text, DefaultClipboardHistoryMaxItems)
	return store
}

func (store ClipboardStore) WithCopiedTextLimit(text string, maxItems int) (ClipboardStore, bool) {
	entry, ok := NewClipboardEntry(text)
	if !ok {
		return store, false
	}
	entries := make([]ClipboardEntry, 0, len(store.Entries)+1)
	entries = append(entries, entry)
	for _, existing := range store.Entries {
		if existing.ID == entry.ID && existing.Text == entry.Text {
			continue
		}
		entries = append(entries, existing)
	}
	store.Entries = limitClipboardEntries(entries, maxItems, MaxClipboardHistoryBytes)
	store = store.markDirty(true)
	return store, true
}

func (store ClipboardStore) ReplaceEntryText(entryID string, text string) ClipboardStore {
	entryID = strings.TrimSpace(entryID)
	if entryID == "" {
		return store
	}
	nextEntry, ok := NewClipboardEntry(text)
	if !ok {
		return store
	}
	next := make([]ClipboardEntry, 0, len(store.Entries))
	replaced := false
	for _, entry := range store.Entries {
		if entry.ID == entryID && !replaced {
			next = append(next, nextEntry)
			replaced = true
			continue
		}
		if entry.ID == nextEntry.ID && entry.Text == nextEntry.Text {
			continue
		}
		next = append(next, entry)
	}
	store.Entries = next
	if replaced {
		store = store.markDirty(false)
	}
	return store
}

func (store ClipboardStore) DeleteEntry(entryID string, text string) ClipboardStore {
	entryID = strings.TrimSpace(entryID)
	if entryID == "" {
		return store
	}
	next := make([]ClipboardEntry, 0, len(store.Entries))
	deleted := false
	for _, entry := range store.Entries {
		if entry.ID == entryID && entry.Text == text {
			deleted = true
			continue
		}
		next = append(next, entry)
	}
	store.Entries = next
	if deleted {
		store = store.markDirty(false)
	}
	return store
}

func (store ClipboardStore) markDirty(mergeable bool) ClipboardStore {
	if !store.Dirty {
		store.Dirty = true
		store.DirtyMergeable = mergeable
		return store
	}
	store.DirtyMergeable = store.DirtyMergeable && mergeable
	return store
}

func (store ClipboardStore) MarkSaved(ref ClipboardStorageRef, version uint64) ClipboardStore {
	store.LastSavedVersion = version
	store.BaseVersion = version
	store.Conflict = false
	store.ConflictVersion = 0
	store.Dirty = false
	store.DirtyMergeable = false
	_ = ref
	return store
}

func (store ClipboardStore) MarkApplied(version uint64) ClipboardStore {
	store.LastAppliedVersion = version
	store.BaseVersion = version
	store.Conflict = false
	store.ConflictVersion = 0
	store.Dirty = false
	store.DirtyMergeable = false
	return store
}

func (store ClipboardStore) MarkMerged(version uint64) ClipboardStore {
	store.LastAppliedVersion = version
	store.BaseVersion = version
	store.Conflict = false
	store.ConflictVersion = 0
	store.Dirty = true
	return store
}

func (store ClipboardStore) MarkEvent(version uint64) ClipboardStore {
	store.LastEventVersion = version
	return store
}

func (store ClipboardStore) MarkConflict(version uint64) ClipboardStore {
	store.Conflict = true
	store.ConflictVersion = version
	return store
}

func (store ClipboardStore) ShouldIgnoreEvent(eventVersion uint64) bool {
	return eventVersion != 0 && eventVersion == store.LastSavedVersion
}

func (store ClipboardStore) SaveVersion() uint64 {
	return store.BaseVersion
}

func (store ClipboardStore) HasPendingLocalChanges() bool {
	return store.Dirty
}

func (store ClipboardStore) PendingChangesAreMergeable() bool {
	return store.Dirty && store.DirtyMergeable
}

func NewClipboardEntry(text string) (ClipboardEntry, bool) {
	text = normalizeClipboardText(text)
	if strings.TrimSpace(text) == "" || len(text) > MaxClipboardHistoryEntryBytes {
		return ClipboardEntry{}, false
	}
	return ClipboardEntry{
		ID:      clipboardEntryID(text),
		Title:   clipboardEntryTitle(text),
		Text:    text,
		Preview: clipboardEntryPreview(text),
	}, true
}

func SnapshotClipboardForStorage(store ClipboardStore) ClipboardStorageSnapshot {
	snapshot := ClipboardStorageSnapshot{
		Schema:        ClipboardStorageSchema,
		SchemaVersion: ClipboardStorageSchemaV2,
		Entries:       make([]ClipboardEntry, 0, len(store.Entries)),
	}
	seen := map[string]struct{}{}
	for _, entry := range store.Entries {
		normalized, ok := normalizeClipboardEntry(entry)
		if !ok {
			continue
		}
		key := normalized.ID + "\x00" + normalized.Text
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		snapshot.Entries = append(snapshot.Entries, normalized)
	}
	return snapshot
}

func (snapshot ClipboardStorageSnapshot) ToClipboardStore() (ClipboardStore, error) {
	if err := snapshot.Validate(); err != nil {
		return ClipboardStore{}, err
	}
	store := ClipboardStore{}
	for _, entry := range snapshot.Entries {
		normalized, ok := normalizeClipboardEntry(entry)
		if !ok {
			continue
		}
		store.Entries = append(store.Entries, normalized)
	}
	store.Entries = limitClipboardEntries(store.Entries, DefaultClipboardHistoryMaxItems, MaxClipboardHistoryBytes)
	return store, nil
}

func (store ClipboardStore) MergeLoadedEntries(loaded ClipboardStore) ClipboardStore {
	entries := make([]ClipboardEntry, 0, len(store.Entries)+len(loaded.Entries))
	seen := map[string]struct{}{}
	appendUnique := func(entry ClipboardEntry) {
		normalized, ok := normalizeClipboardEntry(entry)
		if !ok {
			return
		}
		key := normalized.ID + "\x00" + normalized.Text
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		entries = append(entries, normalized)
	}
	for _, entry := range store.Entries {
		appendUnique(entry)
	}
	for _, entry := range loaded.Entries {
		appendUnique(entry)
	}
	store.Entries = limitClipboardEntries(entries, DefaultClipboardHistoryMaxItems, MaxClipboardHistoryBytes)
	return store
}

func (snapshot ClipboardStorageSnapshot) Validate() error {
	if snapshot.Schema != ClipboardStorageSchema || snapshot.SchemaVersion != ClipboardStorageSchemaV2 {
		return ErrInvalidClipboardSnapshot
	}
	for _, entry := range snapshot.Entries {
		if _, ok := normalizeClipboardEntry(entry); !ok {
			return ErrInvalidClipboardSnapshot
		}
	}
	return nil
}

func EncodeClipboardStorageSnapshotValue(snapshot ClipboardStorageSnapshot) ([]byte, error) {
	if err := snapshot.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(snapshot)
}

func DecodeClipboardStorageSnapshot(data []byte) (ClipboardStorageSnapshot, error) {
	var snapshot ClipboardStorageSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return ClipboardStorageSnapshot{}, err
	}
	if err := snapshot.Validate(); err != nil {
		return ClipboardStorageSnapshot{}, err
	}
	return snapshot, nil
}

func normalizeClipboardEntry(entry ClipboardEntry) (ClipboardEntry, bool) {
	normalized, ok := NewClipboardEntry(entry.Text)
	if !ok {
		return ClipboardEntry{}, false
	}
	return normalized, true
}

func normalizeClipboardText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

func clipboardEntryID(text string) string {
	sum := sha256.Sum256([]byte(text))
	return "clip:" + hex.EncodeToString(sum[:16])
}

func clipboardEntryTitle(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "clipboard entry"
	}
	first := text
	if index := strings.Index(first, "\n"); index >= 0 {
		first = first[:index]
	}
	first = strings.TrimSpace(first)
	if first == "" {
		return "clipboard entry"
	}
	return clipboardSummary(first)
}

func clipboardEntryPreview(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= 1 {
		return clipboardSummary(text)
	}
	first := strings.TrimSpace(lines[0])
	if first == "" {
		return "..."
	}
	return clipboardSummary(first + " ...")
}

func clipboardSummary(text string) string {
	runes := []rune(text)
	if len(runes) <= clipboardEntrySummaryRunes {
		return text
	}
	return string(runes[:clipboardEntrySummaryRunes-3]) + "..."
}

func limitClipboardEntries(entries []ClipboardEntry, maxItems int, maxBytes int) []ClipboardEntry {
	if maxItems <= 0 {
		maxItems = DefaultClipboardHistoryMaxItems
	}
	limited := make([]ClipboardEntry, 0, maxItems)
	usedBytes := 0
	for _, entry := range entries {
		if len(limited) >= maxItems {
			break
		}
		entryBytes := len(entry.Text)
		if entryBytes > MaxClipboardHistoryEntryBytes || usedBytes+entryBytes > maxBytes {
			continue
		}
		limited = append(limited, entry)
		usedBytes += entryBytes
	}
	return limited
}
