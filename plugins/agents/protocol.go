package agents

import (
	"errors"
	"time"

	"github.com/anytty/anytty/proto/apipb"
)

func (e Event) Proto(daemonID string, now time.Time) *apipb.PluginAgentReport {
	return &apipb.PluginAgentReport{AgentId: e.Agent + ":" + e.SessionID, Provider: e.Agent, SessionId: e.SessionID, Sequence: e.Sequence, State: e.Status, Title: e.Title, Cwd: e.Project, Terminal: &apipb.PluginTerminalRef{DaemonId: daemonID, TerminalId: e.TerminalID}, ObservedUnixMillis: now.UnixMilli(), Event: e.Kind, SourceEpoch: e.Epoch, PermissionId: e.PermissionID, FullState: e.FullState, PendingPermissionIds: append([]string(nil), e.PendingPermissions...), BaseState: e.Status}
}

func EventFromProto(r *apipb.PluginAgentReport) (Event, error) {
	if r == nil || r.Terminal == nil {
		return Event{}, errors.New("agent report requires terminal")
	}
	return Event{Agent: r.Provider, SessionID: r.SessionId, TerminalID: r.Terminal.TerminalId, Project: r.Cwd, Title: r.Title, Kind: r.Event, Status: r.State, Epoch: r.SourceEpoch, Sequence: r.Sequence, PermissionID: r.PermissionId, FullState: r.FullState, PendingPermissions: append([]string(nil), r.PendingPermissionIds...)}, nil
}

func (s *Store) ProtoSnapshot(daemonID string) *apipb.PluginAgentSnapshot {
	records, version := s.Snapshot()
	out := &apipb.PluginAgentSnapshot{Revision: version}
	for _, record := range records {
		r := record.Event.Proto(daemonID, record.UpdatedAt)
		r.BaseState = record.BaseStatus
		r.Stale = record.Stale
		r.PendingPermissionIds = append([]string(nil), record.PendingPermissions...)
		out.Agents = append(out.Agents, r)
	}
	return out
}

func (s *Store) Restore(snapshot *apipb.PluginAgentSnapshot) error {
	if snapshot == nil {
		return errors.New("agent snapshot required")
	}
	states := map[string]*sessionState{}
	for _, report := range snapshot.Agents {
		e, err := EventFromProto(report)
		if err != nil {
			return err
		}
		if e.SessionID == "" || e.TerminalID == "" || e.Epoch == 0 || e.Sequence == 0 || !validStatus(e.Status) {
			return errors.New("invalid saved agent report")
		}
		base := report.BaseState
		if base == "" {
			base = report.State
		}
		permissions := map[string]bool{}
		for _, id := range report.PendingPermissionIds {
			permissions[id] = true
		}
		states[e.Agent+"\x00"+e.SessionID] = &sessionState{record: Record{Event: e, UpdatedAt: time.UnixMilli(report.ObservedUnixMillis), Version: snapshot.Revision, BaseStatus: base, PendingPermissions: append([]string(nil), report.PendingPermissionIds...), Stale: report.Stale}, baseStatus: base, permissions: permissions}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = states
	s.version = snapshot.Revision
	return nil
}
