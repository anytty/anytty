package agents

import (
	"errors"
	"sort"
	"sync"
	"time"
)

type Record struct {
	Event
	UpdatedAt          time.Time
	Version            uint64
	BaseStatus         string
	PendingPermissions []string
	Stale              bool
}

type sessionState struct {
	record      Record
	baseStatus  string
	permissions map[string]bool
}

type Store struct {
	mu       sync.Mutex
	sessions map[string]*sessionState
	version  uint64
}

// Apply rejects delayed reports from a superseded source incarnation and
// duplicates. Epochs are ordered, persistent source counters (not random IDs).
func (s *Store) Apply(e Event, now time.Time) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.SessionID == "" || e.TerminalID == "" || e.Epoch == 0 || e.Sequence == 0 || !validStatus(e.Status) || (e.Agent != "codex" && e.Agent != "opencode") {
		return Record{}, false, errors.New("invalid agent report")
	}
	if (e.Kind == "permission.asked" || e.Kind == "permission.replied") && e.PermissionID == "" {
		return Record{}, false, errors.New("permission event missing ID")
	}
	if s.sessions == nil {
		s.sessions = map[string]*sessionState{}
	}
	key := e.Agent + "\x00" + e.SessionID
	old := s.sessions[key]
	if old != nil && (e.Epoch < old.record.Epoch || e.Epoch == old.record.Epoch && e.Sequence <= old.record.Sequence) {
		return old.record, false, nil
	}
	if old == nil || e.Epoch > old.record.Epoch {
		old = &sessionState{baseStatus: "unknown", permissions: map[string]bool{}}
		s.sessions[key] = old
	}
	if e.FullState {
		clear(old.permissions)
		for _, id := range e.PendingPermissions {
			if id != "" {
				old.permissions[id] = true
			}
		}
	}
	if e.Kind == "permission.asked" {
		old.permissions[e.PermissionID] = true
	}
	if e.Kind == "permission.replied" {
		delete(old.permissions, e.PermissionID)
	}
	if e.Status != "" {
		old.baseStatus = e.Status
	}
	if e.Status == "exited" {
		clear(old.permissions)
	}
	if e.Project == "" {
		e.Project = old.record.Project
	}
	if e.Title == "" {
		e.Title = old.record.Title
	}
	e.Status = old.baseStatus
	if len(old.permissions) > 0 {
		e.Status = "blocked"
	}
	s.version++
	pending := make([]string, 0, len(old.permissions))
	for id := range old.permissions {
		pending = append(pending, id)
	}
	sort.Strings(pending)
	old.record = Record{Event: e, UpdatedAt: now, Version: s.version, BaseStatus: old.baseStatus, PendingPermissions: pending}
	return old.record, true, nil
}

func (s *Store) Snapshot() ([]Record, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]Record, 0, len(s.sessions))
	for _, state := range s.sessions {
		rows = append(rows, state.record)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Agent != rows[j].Agent {
			return rows[i].Agent < rows[j].Agent
		}
		return rows[i].SessionID < rows[j].SessionID
	})
	return rows, s.version
}

// A restored live status is an observation from an earlier service lifetime,
// not proof that the Agent is still running. A fresh accepted report clears it.
func (s *Store) MarkRestoredStale() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	for _, state := range s.sessions {
		if state.record.Status != "exited" && !state.record.Stale {
			state.record.Stale = true
			changed = true
		}
	}
	if changed {
		s.version++
	}
	return changed
}
