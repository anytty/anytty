// Package agents implements the Agent workbench plugin's hook adapters and state.
// Agent JSON is accepted only at the external hook boundary; transport uses protobuf.
package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const PluginID = "org.anytty.agents"

type Context struct {
	TerminalID   string
	DaemonSocket string
}

func (c Context) Available() bool { return c.TerminalID != "" && c.DaemonSocket != "" }

type Event struct {
	Agent              string   `json:"agent"`
	SessionID          string   `json:"session_id"`
	TerminalID         string   `json:"terminal_id"`
	Project            string   `json:"project"`
	Title              string   `json:"title"`
	Kind               string   `json:"kind"`
	Status             string   `json:"status"`
	PermissionID       string   `json:"permission_id,omitempty"`
	Epoch              uint64   `json:"epoch"`
	Sequence           uint64   `json:"sequence"`
	FullState          bool     `json:"full_state,omitempty"`
	PendingPermissions []string `json:"pending_permissions,omitempty"`
}

// DecodeCodex deliberately never reads prompts, tool input, or transcripts.
func DecodeCodex(data []byte, ctx Context) (*Event, error) {
	if !ctx.Available() {
		return nil, nil
	}
	var raw struct {
		SessionID string `json:"session_id"`
		CWD       string `json:"cwd"`
		Name      string `json:"hook_event_name"`
		Source    string `json:"source"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("codex hook: %w", err)
	}
	if raw.SessionID == "" {
		return nil, errors.New("codex hook missing session_id")
	}
	e := &Event{Agent: "codex", SessionID: raw.SessionID, TerminalID: ctx.TerminalID, Project: raw.CWD, Kind: raw.Name}
	switch raw.Name {
	case "SessionStart":
		if raw.Source == "compact" {
			e.Kind = "metadata"
		} else {
			e.Kind = "start"
			e.Status = "idle"
		}
	case "UserPromptSubmit", "PreToolUse", "PostToolUse":
		e.Status = "working"
	case "PermissionRequest":
		e.Status = "blocked"
	case "Stop", "Interrupt":
		e.Status = "idle"
	case "SessionEnd":
		e.Status = "exited"
	default:
		return nil, nil // SubagentStop does not finish its parent.
	}
	return e, nil
}

// DecodeOpenCode accepts the reduced envelope emitted by our official plugin
// callback, rather than arbitrary terminal output or guessed Agent activity.
func DecodeOpenCode(data []byte, ctx Context) (*Event, error) {
	if !ctx.Available() {
		return nil, nil
	}
	var e Event
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("opencode hook: %w", err)
	}
	if e.Agent != "opencode" || e.SessionID == "" || e.Epoch == 0 || e.Sequence == 0 {
		return nil, errors.New("invalid opencode hook identity or sequence")
	}
	switch e.Kind {
	case "start", "metadata", "status", "permission.asked", "permission.replied":
	default:
		return nil, errors.New("invalid opencode hook event")
	}
	if !validStatus(e.Status) {
		return nil, errors.New("invalid opencode hook status")
	}
	if e.FullState && e.Status == "" {
		return nil, errors.New("complete state requires base status")
	}
	if strings.HasPrefix(e.Kind, "permission.") && e.PermissionID == "" {
		return nil, errors.New("permission event missing ID")
	}
	e.TerminalID = ctx.TerminalID // A hook cannot select another terminal through stdin.
	return &e, nil
}

func validStatus(s string) bool {
	switch s {
	case "", "unknown", "idle", "working", "blocked", "error", "exited":
		return true
	}
	return false
}
