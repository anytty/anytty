// anytty-keyprobe diagnoses the same host parser and input router used by the TUI.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/anytty/anytty/tui/config"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
	"github.com/anytty/anytty/tui/terminalhost"
	"github.com/muesli/cancelreader"
)

type eventReport struct {
	Kind            input.EventKind        `json:"kind"`
	Key             input.Key              `json:"key,omitempty"`
	Char            string                 `json:"char,omitempty"`
	Ctrl            bool                   `json:"ctrl"`
	Alt             bool                   `json:"alt"`
	Shift           bool                   `json:"shift"`
	Protocol        input.KeyboardProtocol `json:"protocol,omitempty"`
	RawHex          string                 `json:"raw_hex"`
	Route           input.IntentKind       `json:"route"`
	Action          string                 `json:"action,omitempty"`
	OutputHex       string                 `json:"output_hex"`
	ForcedOutputHex string                 `json:"forced_output_hex"`
	Reason          string                 `json:"reason,omitempty"`
}

func describe(event input.InputEvent, shortcuts state.TUIShortcutConfig) eventReport {
	routed := input.RouteWithOptions(event, input.RouteOptions{Shortcuts: shortcuts})
	forced := input.RouteWithOptions(event, input.RouteOptions{ForceTerminalPassthrough: true})
	return eventReport{Kind: event.Kind, Key: event.Key, Char: event.Char, Ctrl: event.Ctrl, Alt: event.Alt, Shift: event.Shift,
		Protocol: event.KeyboardProtocol, RawHex: hex.EncodeToString([]byte(event.RawSeq)), Route: routed.Kind,
		Action: string(routed.Invocation.ID), OutputHex: hex.EncodeToString(routed.Bytes), ForcedOutputHex: hex.EncodeToString(forced.Bytes), Reason: routed.Reason}
}

type recorder struct {
	mu      sync.Mutex
	encoder *json.Encoder
	err     error
}

func (r *recorder) write(value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		r.err = r.encoder.Encode(value)
	}
}
func (r *recorder) result() error { r.mu.Lock(); defer r.mu.Unlock(); return r.err }

type recordingReader struct {
	terminalhost.CancelReader
	log *recorder
}

func (r *recordingReader) Read(p []byte) (int, error) {
	n, err := r.CancelReader.Read(p)
	if n > 0 {
		r.log.write(map[string]any{"type": "raw", "time": time.Now().UTC(), "hex": hex.EncodeToString(p[:n])})
	}
	return n, err
}
func (r *recordingReader) Close() error {
	if closer, ok := r.CancelReader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func capture(duration time.Duration, output io.Writer, shortcuts state.TUIShortcutConfig) error {
	log := &recorder{encoder: json.NewEncoder(output)}
	log.write(map[string]any{"type": "session", "term": os.Getenv("TERM"), "term_program": os.Getenv("TERM_PROGRAM"), "duration": duration.String()})
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	host := terminalhost.New(terminalhost.WithThemeProbe(false), terminalhost.WithCancelReaderFactory(func(reader io.Reader) (terminalhost.CancelReader, error) {
		cr, err := cancelreader.NewReader(reader)
		if err != nil {
			return nil, err
		}
		return &recordingReader{CancelReader: cr, log: log}, nil
	}))
	if err := host.Enter(ctx); err != nil {
		return err
	}
	defer host.Close()
	fmt.Fprint(os.Stdout, "\x1b[2J\x1b[HAnyTTY key probe: press keys; Esc three times (pause between presses) exits.\r\nNo keys are forwarded or actions executed. Capture ends automatically.\r\n")
	escapes := 0
	for {
		select {
		case <-ctx.Done():
			if err := host.Close(); err != nil {
				return err
			}
			return log.result()
		case event := <-host.InputEvents():
			report := describe(event, shortcuts)
			log.write(map[string]any{"type": "event", "time": time.Now().UTC(), "event": report})
			data, _ := json.Marshal(report)
			fmt.Fprintf(os.Stdout, "%s\r\n", data)
			if event.Kind == input.EventKindKey && event.Key == input.KeyEsc && !event.Ctrl && !event.Alt && !event.Shift {
				escapes++
			} else {
				escapes = 0
			}
			if escapes == 3 {
				if err := host.Close(); err != nil {
					return err
				}
				return log.result()
			}
		}
	}
}

func run() error {
	mode := flag.String("mode", "capture", "capture real keys or matrix synthetic combinations")
	path := flag.String("output", "", "required output file (created exclusively)")
	configPath := flag.String("config", "", "optional TUI config; otherwise built-in defaults")
	duration := flag.Duration("duration", 2*time.Minute, "capture duration")
	flag.Parse()
	if *path == "" {
		return fmt.Errorf("-output is required")
	}
	if *mode != "capture" && *mode != "matrix" {
		return fmt.Errorf("unknown mode %q", *mode)
	}
	if *duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	shortcuts := config.Default().Shortcuts
	if *configPath != "" {
		cfg, err := config.Load(*configPath, nil)
		if err != nil {
			return err
		}
		shortcuts = cfg.Shortcuts
	}
	file, err := os.OpenFile(*path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if *mode == "matrix" {
		return writeMatrix(file, shortcuts)
	}
	return capture(*duration, file, shortcuts)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
