// Command tui2-scrollcheck probes the exact terminal history path used by
// tui2. It is intentionally independent of the TUI renderer: one invocation
// attaches to a daemon terminal, runs the same +/-1 scroll operations as the
// shell program, and prints one JSON record per step.
//
// Example:
//
//	go run ./clients/tui/cmd/tui2-scrollcheck -endpoint hs -terminal autopush
//
// A successful first scroll proves that the remote service implements the
// current history-window contract. An unsupported/missing-result error points
// at an old daemon or a daemon with history disabled, before mouse routing is
// involved.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anytty/anytty/clients/tui/endpoint"
	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/runtime"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, string(os.PathListSeparator)) }

func (s *stringList) Set(value string) error {
	for _, item := range strings.Split(value, string(os.PathListSeparator)) {
		if item = strings.TrimSpace(item); item != "" {
			*s = append(*s, item)
		}
	}
	return nil
}

type record struct {
	Time          string   `json:"time"`
	Phase         string   `json:"phase"`
	Endpoint      string   `json:"endpoint,omitempty"`
	Terminal      string   `json:"terminal,omitempty"`
	Kind          string   `json:"kind,omitempty"`
	ConnectMode   string   `json:"connect_mode,omitempty"`
	Health        string   `json:"health,omitempty"`
	Persistent    bool     `json:"persistent,omitempty"`
	Delta         int      `json:"delta,omitempty"`
	Offset        int      `json:"offset,omitempty"`
	HistoryActive bool     `json:"history_active,omitempty"`
	Rows          int      `json:"rows,omitempty"`
	Sample        []string `json:"sample,omitempty"`
	DurationMS    int64    `json:"duration_ms,omitempty"`
	Error         string   `json:"error,omitempty"`
	Diagnosis     string   `json:"diagnosis,omitempty"`
}

func emit(v record) {
	v.Time = time.Now().UTC().Format(time.RFC3339Nano)
	_ = json.NewEncoder(os.Stdout).Encode(v)
}

func classify(err error) string {
	if err == nil {
		return "ok"
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "unsupported"), strings.Contains(text, "missing window result"):
		return "remote_history_api_missing_or_old_service"
	case strings.Contains(text, "history disabled"), strings.Contains(text, "not available"):
		return "remote_history_disabled"
	case strings.Contains(text, "connection changed"), strings.Contains(text, "offline"):
		return "remote_connection_changed_or_offline"
	default:
		return "scroll_path_error"
	}
}

func classifyAttach(err string) string {
	text := strings.ToLower(err)
	switch {
	case strings.Contains(text, "operation not permitted"):
		return "transport_blocked_by_environment"
	case strings.Contains(text, "no eligible route"), strings.Contains(text, "no available credential"):
		return "remote_route_or_credential_unavailable"
	case strings.Contains(text, "offline"):
		return "endpoint_offline"
	default:
		return "attach_failed"
	}
}

func resolveConfig(paths []string, name string) (endpoint.Config, bool, error) {
	cfg, ok, err := endpoint.SharedConfigForEndpointIn(paths, name)
	if err != nil || ok {
		return cfg, ok, err
	}
	// The interactive TUI accepts the human label (for example "hs") while
	// the protocol uses the stable device id. Keep the probe convenient by
	// resolving that label before reporting endpoint_not_found.
	configs, warnings := endpoint.LoadSharedEndpointConfigs(paths...)
	for _, candidate := range configs {
		if strings.EqualFold(strings.TrimSpace(candidate.Label), strings.TrimSpace(name)) {
			return candidate, true, nil
		}
	}
	if len(warnings) > 0 && err == nil {
		err = errors.New(strings.Join(warnings, "; "))
	}
	return endpoint.Config{}, false, err
}

func main() {
	var paths stringList
	endpointName := flag.String("endpoint", "", "daemon endpoint name from endpoints.yaml")
	terminalID := flag.String("terminal", "", "daemon terminal id, for example autopush")
	cols := flag.Int("cols", 80, "probe width")
	rows := flag.Int("rows", 24, "probe height")
	flag.Var(&paths, "registry", "endpoints.yaml path; may be repeated or path-list separated")
	flag.Parse()
	if strings.TrimSpace(*endpointName) == "" || strings.TrimSpace(*terminalID) == "" {
		fmt.Fprintln(os.Stderr, "usage: tui2-scrollcheck -endpoint NAME -terminal ID [-registry endpoints.yaml]")
		os.Exit(2)
	}

	cfg, ok, err := resolveConfig(paths, *endpointName)
	if err != nil {
		emit(record{Phase: "config", Endpoint: *endpointName, Error: err.Error(), Diagnosis: "endpoint_registry_error"})
		os.Exit(1)
	}
	if !ok {
		emit(record{Phase: "config", Endpoint: *endpointName, Error: "endpoint not found in registry", Diagnosis: "endpoint_not_found"})
		os.Exit(1)
	}
	emit(record{Phase: "config", Endpoint: cfg.Name, Kind: cfg.KindName(), ConnectMode: cfg.ConnectModeName()})

	mgr := endpoint.NewManager(endpoint.Options{RegistryPaths: paths})
	defer mgr.Close()
	if err := mgr.Register(cfg); err != nil {
		emit(record{Phase: "register", Endpoint: cfg.Name, Error: err.Error(), Diagnosis: "endpoint_register_error"})
		os.Exit(1)
	}

	handler := runtime.NewTerminalHandler(runtime.TerminalOptions{
		Cols: *cols, Rows: *rows, OwnerID: "tui2-scrollcheck",
		NewPTY: func(config pty.Config) pty.PTY {
			config.ViewID = "tui2-scrollcheck"
			config.Fit = false
			return mgr.NewRemotePTY(config)
		},
	})
	defer handler.Close()

	fit := false
	attach := runtime.Request{
		Method: runtime.Method{Name: "terminal.attach"},
		Params: &pb.MethodParams{Endpoint: cfg.Name, Id: *terminalID, Fit: &fit},
	}
	start := time.Now()
	outcome, pending := handler.Handle(attach)
	if !outcome.OK || pending {
		emit(record{Phase: "attach", Endpoint: cfg.Name, Terminal: *terminalID, Health: mgr.Health(cfg.Name), Error: outcome.Error, Diagnosis: classifyAttach(outcome.Error), DurationMS: time.Since(start).Milliseconds()})
		os.Exit(1)
	}
	term, ok := handler.TerminalAt(cfg.Name, *terminalID)
	if !ok {
		emit(record{Phase: "attach", Endpoint: cfg.Name, Terminal: *terminalID, Error: "terminal not registered", Diagnosis: "attach_missing"})
		os.Exit(1)
	}
	emit(record{Phase: "attach", Endpoint: cfg.Name, Terminal: *terminalID, Health: mgr.Health(cfg.Name), Persistent: term.HasPersistentHistory()})

	// This is deliberately the same gesture the shell receives from its wheel
	// handler: one row older, then two rows newer. The third down step checks
	// that a live bottom remains a stable no-op.
	for _, delta := range []int{1, -1, -1} {
		start = time.Now()
		_, offset, scrollErr := term.HistoryScroll(context.Background(), delta, *rows)
		lines := term.VisibleLines()
		sample := append([]string(nil), lines...)
		if len(sample) > 3 {
			sample = sample[:3]
		}
		r := record{
			Phase: "scroll", Endpoint: cfg.Name, Terminal: *terminalID,
			Health: mgr.Health(cfg.Name), Persistent: term.HasPersistentHistory(),
			Delta: delta, Offset: offset, HistoryActive: term.HistoryActive(),
			Rows: len(lines), Sample: sample, DurationMS: time.Since(start).Milliseconds(),
		}
		if scrollErr != nil {
			r.Error = scrollErr.Error()
			r.Diagnosis = classify(scrollErr)
			var apiErr *endpoint.APIError
			if errors.As(scrollErr, &apiErr) {
				r.Diagnosis += ":api_code=" + fmt.Sprint(apiErr.Code)
			}
			emit(r)
			break
		}
		r.Diagnosis = "remote_history_window_ok"
		emit(r)
	}

	if err := term.HistoryRelease(context.Background()); err != nil {
		emit(record{Phase: "release", Endpoint: cfg.Name, Terminal: *terminalID, Error: err.Error(), Diagnosis: classify(err)})
		os.Exit(1)
	}
	emit(record{Phase: "release", Endpoint: cfg.Name, Terminal: *terminalID, Health: mgr.Health(cfg.Name), Diagnosis: "history_release_ok"})
}
