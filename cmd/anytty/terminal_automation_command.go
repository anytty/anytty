package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/spf13/cobra"
)

const maxTerminalSendBytes = 16 << 20

type terminalAutomationTarget struct {
	Ref      resolvedTerminalRef
	Endpoint endpointdomain.Endpoint
	Client   terminalProtocolClient
	Close    func()
}

type terminalSendEnvelope struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Target        string `json:"target"`
	Bytes         int    `json:"bytes"`
}

type terminalCaptureEnvelope struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Target        string `json:"target"`
	Source        string `json:"source"`
	Text          string `json:"text"`
}

type terminalResizeEnvelope struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Target        string `json:"target"`
	Cols          uint16 `json:"cols"`
	Rows          uint16 `json:"rows"`
	Resized       bool   `json:"resized"`
	CanResize     bool   `json:"can_resize"`
	Reason        string `json:"reason,omitempty"`
	OwnerSurface  string `json:"owner_surface_id,omitempty"`
	OwnerView     string `json:"owner_view_id,omitempty"`
}

type terminalWaitEnvelope struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Target        string `json:"target"`
	State         string `json:"state"`
	ExitCode      *int   `json:"exit_code,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
}

type terminalEventEnvelope struct {
	SchemaVersion int            `json:"schema_version"`
	Kind          string         `json:"kind"`
	EndpointID    string         `json:"endpoint_id"`
	Target        string         `json:"target,omitempty"`
	Type          string         `json:"type"`
	Timestamp     string         `json:"timestamp,omitempty"`
	Data          map[string]any `json:"data,omitempty"`
}

func newTerminalSendCommand(runtime terminalCommandRuntime) *cobra.Command {
	var stdin, enter, jsonOutput bool
	var literals, keys, hexValues []string
	var timeout time.Duration
	command := &cobra.Command{
		Use:   "send TARGET [TEXT...]",
		Short: "Send literal bytes, named keys, hex bytes, or stdin to a terminal",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := terminalSendPayload(cmd.InOrStdin(), args[1:], literals, keys, hexValues, stdin, enter)
			if err != nil {
				return err
			}
			ctx, cancel, err := terminalCommandContext(cmd.Context(), timeout)
			if err != nil {
				return err
			}
			defer cancel()
			target, err := openTerminalAutomationTarget(ctx, cmd, runtime, args[0])
			if err != nil {
				return err
			}
			defer target.Close()
			attachment, _, detach, err := attachTerminalAutomation(ctx, target.Client, target.Ref, apipb.ResizePolicy_RESIZE_POLICY_FOLLOWER, "send")
			if err != nil {
				return classifyCLIError(err)
			}
			defer detach()
			if err := target.Client.TerminalInput(ctx, &apipb.TerminalInputCommand{Attachment: attachment.GetAttachment().GetResource(), Data: data}); err != nil {
				return classifyCLIError(err)
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(terminalSendEnvelope{1, "terminal_input_sent", target.Ref.String(), len(data)})
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Target", Value: target.Ref.String()},
				cliField{Label: "Status", Value: "sent"},
				cliField{Label: "Bytes", Value: strconv.Itoa(len(data))},
			)
		},
	}
	command.Flags().StringArrayVar(&literals, "literal", nil, "literal UTF-8 text (repeatable)")
	command.Flags().StringArrayVar(&keys, "key", nil, "named key such as Enter, Ctrl-C, Up, or Escape (repeatable)")
	command.Flags().StringArrayVar(&hexValues, "hex", nil, "hex-encoded bytes (repeatable)")
	command.Flags().BoolVar(&stdin, "stdin", false, "read bytes from stdin")
	command.Flags().BoolVar(&enter, "enter", false, "append a carriage return")
	command.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "operation timeout")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func terminalSendPayload(reader io.Reader, args, literals, keys, hexValues []string, stdin, enter bool) ([]byte, error) {
	sources := 0
	if len(args) > 0 {
		sources++
	}
	if len(literals) > 0 {
		sources++
	}
	if len(keys) > 0 {
		sources++
	}
	if len(hexValues) > 0 {
		sources++
	}
	if stdin {
		sources++
	}
	if sources == 0 && !enter {
		return nil, usageCLIError("provide TEXT, --literal, --key, --hex, --stdin, or --enter")
	}
	if sources > 1 {
		return nil, usageCLIError("terminal send accepts exactly one input source")
	}
	var payload []byte
	switch {
	case len(args) > 0:
		payload = []byte(strings.Join(args, " "))
	case len(literals) > 0:
		payload = []byte(strings.Join(literals, ""))
	case len(keys) > 0:
		for _, key := range keys {
			encoded, err := encodeTerminalKey(key)
			if err != nil {
				return nil, err
			}
			payload = append(payload, encoded...)
		}
	case len(hexValues) > 0:
		for _, value := range hexValues {
			encoded, err := hex.DecodeString(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
			if err != nil {
				return nil, usageCLIError(fmt.Sprintf("invalid --hex value: %v", err))
			}
			payload = append(payload, encoded...)
		}
	case stdin:
		limited, err := io.ReadAll(io.LimitReader(reader, maxTerminalSendBytes+1))
		if err != nil {
			return nil, err
		}
		if len(limited) > maxTerminalSendBytes {
			return nil, usageCLIError("stdin exceeds 16 MiB terminal input limit")
		}
		payload = limited
	}
	if enter {
		payload = append(payload, '\r')
	}
	if len(payload) > maxTerminalSendBytes {
		return nil, usageCLIError("terminal input exceeds 16 MiB limit")
	}
	return payload, nil
}

func encodeTerminalKey(value string) ([]byte, error) {
	key := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(key, "c-") {
		key = "ctrl-" + strings.TrimPrefix(key, "c-")
	}
	keys := map[string]string{
		"enter": "\r", "return": "\r", "tab": "\t", "escape": "\x1b", "esc": "\x1b",
		"backspace": "\x7f", "space": " ", "up": "\x1b[A", "down": "\x1b[B", "right": "\x1b[C", "left": "\x1b[D",
		"home": "\x1b[H", "end": "\x1b[F", "insert": "\x1b[2~", "delete": "\x1b[3~", "pageup": "\x1b[5~", "pagedown": "\x1b[6~", "shift-tab": "\x1b[Z",
		"f1": "\x1bOP", "f2": "\x1bOQ", "f3": "\x1bOR", "f4": "\x1bOS", "f5": "\x1b[15~", "f6": "\x1b[17~",
		"f7": "\x1b[18~", "f8": "\x1b[19~", "f9": "\x1b[20~", "f10": "\x1b[21~", "f11": "\x1b[23~", "f12": "\x1b[24~",
		"ctrl-space": "\x00", "ctrl-@": "\x00", "ctrl-[": "\x1b", "ctrl-\\": "\x1c", "ctrl-]": "\x1d", "ctrl-^": "\x1e", "ctrl-_": "\x1f", "ctrl-?": "\x7f",
	}
	if encoded, ok := keys[key]; ok {
		return []byte(encoded), nil
	}
	if strings.HasPrefix(key, "ctrl-") && utf8.RuneCountInString(strings.TrimPrefix(key, "ctrl-")) == 1 {
		r, _ := utf8.DecodeRuneInString(strings.TrimPrefix(key, "ctrl-"))
		if r >= 'a' && r <= 'z' {
			return []byte{byte(r-'a') + 1}, nil
		}
	}
	return nil, usageCLIError(fmt.Sprintf("unsupported terminal key %q", value))
}

func newTerminalCaptureCommand(runtime terminalCommandRuntime) *cobra.Command {
	var live, jsonOutput bool
	var lines, cols int
	var timeout time.Duration
	command := &cobra.Command{
		Use:   "capture TARGET",
		Short: "Capture authoritative terminal history or the latest native screen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if lines <= 0 || lines > 100000 || cols <= 0 || cols > 65535 {
				return usageCLIError("--lines must be 1..100000 and --cols must be 1..65535")
			}
			ctx, cancel, err := terminalCommandContext(cmd.Context(), timeout)
			if err != nil {
				return err
			}
			defer cancel()
			target, err := openTerminalAutomationTarget(ctx, cmd, runtime, args[0])
			if err != nil {
				return err
			}
			defer target.Close()
			source := "history"
			var text string
			if live {
				source = "live"
				snapshot, err := target.Client.LiveScreenNext(ctx, &apipb.LiveScreenNextCommand{Terminal: &apipb.TerminalRef{EndpointId: string(target.Ref.EndpointID), TerminalId: target.Ref.TerminalID}})
				if err != nil {
					return classifyCLIError(err)
				}
				text = nativeScreenText(snapshot)
			} else {
				text, err = captureTerminalHistory(ctx, target.Client, target.Ref, lines, cols)
				if err != nil {
					return classifyCLIError(err)
				}
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(terminalCaptureEnvelope{1, "terminal_capture", target.Ref.String(), source, text})
			}
			if _, err := io.WriteString(cmd.OutOrStdout(), text); err != nil {
				return err
			}
			if text != "" && !strings.HasSuffix(text, "\n") {
				_, err = io.WriteString(cmd.OutOrStdout(), "\n")
			}
			return err
		},
	}
	command.Flags().BoolVar(&live, "live", false, "capture the latest native screen instead of history")
	command.Flags().IntVar(&lines, "lines", 200, "maximum projected history rows")
	command.Flags().IntVar(&cols, "cols", 80, "history projection width")
	command.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "operation timeout")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func captureTerminalHistory(ctx context.Context, client terminalProtocolClient, ref resolvedTerminalRef, lines, cols int) (string, error) {
	// copy 必须绑定 daemon 签发的 frozen token；CLI 不从 window rows 自行拼第二份 history truth。
	terminal := &apipb.TerminalRef{EndpointId: string(ref.EndpointID), TerminalId: ref.TerminalID}
	window, err := client.HistoryWindow(ctx, &apipb.HistoryWindowCommand{Terminal: terminal, Limit: int32(lines), Cols: int32(cols), Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST})
	if err != nil {
		return "", err
	}
	if window.GetToken() == "" {
		return "", fmt.Errorf("daemon returned history without a copy token")
	}
	defer func() {
		releaseContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = client.HistoryRelease(releaseContext, &apipb.HistoryReleaseCommand{Terminal: terminal, Token: window.GetToken()})
	}()
	if len(window.GetRows()) == 0 {
		return "", nil
	}
	endCol := int32(0)
	for _, cell := range window.GetRows()[len(window.GetRows())-1].GetRow().GetCells() {
		if cell.GetWidth() > 0 {
			endCol += cell.GetWidth()
		}
	}
	result, err := client.HistoryCopy(ctx, &apipb.HistoryCopyCommand{Terminal: terminal, Window: &apipb.HistoryWindowCommand{Token: window.GetToken(), Cols: int32(cols), HistoryGeneration: window.GetHistoryGeneration(), Range: &apipb.HistoryRange{StartLineId: window.GetFirstLineId(), EndLineId: window.GetLastLineId(), EndCol: endCol}}})
	if err != nil {
		return "", err
	}
	return result.GetText(), nil
}

func nativeScreenText(snapshot *apipb.NativeScreenResult) string {
	if snapshot == nil {
		return ""
	}
	rows := make([]string, 0, len(snapshot.GetRowReplacements()))
	for _, replacement := range snapshot.GetRowReplacements() {
		row := replacement.GetRow()
		var text strings.Builder
		for _, cell := range row.GetCells() {
			text.WriteString(cell.GetContent())
		}
		rows = append(rows, strings.TrimRight(text.String(), " \t"))
	}
	for len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	return strings.Join(rows, "\n")
}

func newTerminalResizeCommand(runtime terminalCommandRuntime) *cobra.Command {
	var jsonOutput bool
	var timeout time.Duration
	command := &cobra.Command{
		Use:   "resize TARGET COLS ROWS",
		Short: "Request an owner-authorized terminal resize",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cols, rows, err := parseTerminalSize(args[1], args[2])
			if err != nil {
				return err
			}
			ctx, cancel, err := terminalCommandContext(cmd.Context(), timeout)
			if err != nil {
				return err
			}
			defer cancel()
			target, err := openTerminalAutomationTarget(ctx, cmd, runtime, args[0])
			if err != nil {
				return err
			}
			defer target.Close()
			attachment, _, detach, err := attachTerminalAutomation(ctx, target.Client, target.Ref, apipb.ResizePolicy_RESIZE_POLICY_OWNER, "resize")
			if err != nil {
				return classifyCLIError(err)
			}
			defer detach()
			result, err := target.Client.TerminalResize(ctx, &apipb.TerminalResizeCommand{Attachment: attachment.GetAttachment().GetResource(), Size: &apipb.TerminalSize{Cols: uint32(cols), Rows: uint32(rows)}, ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER})
			if err != nil {
				return classifyCLIError(err)
			}
			view := terminalResizeEnvelope{SchemaVersion: 1, Kind: "terminal_resized", Target: target.Ref.String(), Cols: uint16(result.GetSize().GetCols()), Rows: uint16(result.GetSize().GetRows()), Resized: result.GetResized()}
			if control := result.GetResizeControl(); control != nil {
				view.CanResize, view.Reason = control.GetCanResize(), resizeControlReasonString(control.GetReason())
				view.OwnerSurface, view.OwnerView = control.GetOwnerSurfaceId(), control.GetOwnerViewId()
			}
			if !view.Resized || !view.CanResize {
				return &cliError{code: 4, message: fmt.Sprintf("terminal %s resize rejected: %s", target.Ref.String(), terminalResizeReason(view))}
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(view)
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Target", Value: target.Ref.String()},
				cliField{Label: "Status", Value: "resized"},
				cliField{Label: "Size", Value: fmt.Sprintf("%dx%d", view.Cols, view.Rows)},
			)
		},
	}
	command.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "operation timeout")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func parseTerminalSize(colsValue, rowsValue string) (uint16, uint16, error) {
	colsValue = strings.TrimSpace(colsValue)
	rowsValue = strings.TrimSpace(rowsValue)
	colsParsed, err := strconv.ParseUint(colsValue, 10, 16)
	if err != nil || colsParsed == 0 {
		return 0, 0, usageCLIError("COLS must be an integer from 1 to 65535")
	}
	rowsParsed, err := strconv.ParseUint(rowsValue, 10, 16)
	if err != nil || rowsParsed == 0 {
		return 0, 0, usageCLIError("ROWS must be an integer from 1 to 65535")
	}
	return uint16(colsParsed), uint16(rowsParsed), nil
}

func terminalResizeReason(view terminalResizeEnvelope) string {
	if view.Reason != "" {
		return view.Reason
	}
	return "daemon did not grant resize ownership"
}

func newTerminalWaitCommand(runtime terminalCommandRuntime) *cobra.Command {
	var state string
	var timeout time.Duration
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "wait TARGET",
		Short: "Wait for a daemon-owned terminal lifecycle state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			state = strings.ToLower(strings.TrimSpace(state))
			if state != "created" && state != "running" && state != "exited" && state != "removed" {
				return usageCLIError("--state must be created, running, exited, or removed")
			}
			ctx, cancel, err := terminalCommandContext(cmd.Context(), timeout)
			if err != nil {
				return err
			}
			defer cancel()
			target, err := openTerminalAutomationTarget(ctx, cmd, runtime, args[0])
			if err != nil {
				return err
			}
			defer target.Close()
			result, err := waitForTerminalState(ctx, target.Client, target.Ref, state)
			if err != nil {
				return classifyCLIError(err)
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			fields := []cliField{
				{Label: "Target", Value: result.Target},
				{Label: "State", Value: result.State},
			}
			if result.ExitCode != nil {
				fields = append(fields, cliField{Label: "Exit code", Value: strconv.Itoa(*result.ExitCode)})
			}
			if result.Timestamp != "" {
				fields = append(fields, cliField{Label: "Timestamp", Value: result.Timestamp})
			}
			return writeCLIFields(cmd.OutOrStdout(), fields...)
		},
	}
	command.Flags().StringVar(&state, "state", "exited", "created, running, exited, or removed")
	command.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "maximum wait duration")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func waitForTerminalState(ctx context.Context, client terminalProtocolClient, ref resolvedTerminalRef, desired string) (terminalWaitEnvelope, error) {
	// 先订阅再读取 inventory，避免状态在 snapshot 与 event arm 之间变化造成丢边沿。
	_, events, err := client.EventSubscribe(ctx, &apipb.EventSubscribeCommand{Terminal: &apipb.TerminalRef{EndpointId: string(ref.EndpointID), TerminalId: ref.TerminalID}, Types: []apipb.ApplicationEventType{apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE}})
	if err != nil {
		return terminalWaitEnvelope{}, err
	}
	list, err := client.TerminalList(ctx, &apipb.TerminalListCommand{})
	if err != nil {
		return terminalWaitEnvelope{}, err
	}
	found := false
	for _, item := range list.Terminals {
		if item.GetRef().GetTerminalId() == ref.TerminalID {
			found = true
			if desired == "created" || terminalStateString(item.GetState()) == desired {
				return terminalWaitEnvelope{1, "terminal_wait", ref.String(), desired, int32PointerToCLIInt(item.ExitCode), formatTerminalUnixNano(item.GetExitedAtUnixNano())}, nil
			}
			break
		}
	}
	if desired == "removed" && !found {
		return terminalWaitEnvelope{1, "terminal_wait", ref.String(), desired, nil, ""}, nil
	}
	for {
		select {
		case <-ctx.Done():
			return terminalWaitEnvelope{}, ctx.Err()
		case event, ok := <-events:
			if !ok {
				return terminalWaitEnvelope{}, io.EOF
			}
			if result, ok := terminalWaitResult(ref, desired, event); ok {
				return result, nil
			}
		}
	}
}

func terminalWaitResult(ref resolvedTerminalRef, desired string, event *apipb.EventEnvelope) (terminalWaitEnvelope, bool) {
	terminal := event.GetTerminalLifecycle().GetTerminal()
	if terminal == nil || terminal.GetRef().GetTerminalId() != ref.TerminalID {
		return terminalWaitEnvelope{}, false
	}
	state := terminalStateString(terminal.GetState())
	if desired == "created" {
		if state != "created" && state != "running" {
			return terminalWaitEnvelope{}, false
		}
	} else if state != desired {
		return terminalWaitEnvelope{}, false
	}
	return terminalWaitEnvelope{SchemaVersion: 1, Kind: "terminal_wait", Target: ref.String(), State: desired, ExitCode: int32PointerToCLIInt(terminal.ExitCode), Timestamp: formatTerminalUnixNano(event.GetTimestampUnixNano())}, true
}

func newTerminalEventsCommand(runtime terminalCommandRuntime) *cobra.Command {
	var output string
	var types []string
	var count int
	var timeout time.Duration
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "events [TARGET]",
		Short: "Stream daemon terminal events as human records or NDJSON",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput {
				output = "ndjson"
			}
			output = strings.ToLower(strings.TrimSpace(output))
			if output != "human" && output != "ndjson" {
				return usageCLIError("--output must be human or ndjson")
			}
			if count < 0 {
				return usageCLIError("--count cannot be negative")
			}
			eventTypes, err := parseTerminalEventTypes(types)
			if err != nil {
				return err
			}
			ctx, cancel, err := terminalCommandContextOptional(cmd.Context(), timeout)
			if err != nil {
				return err
			}
			defer cancel()
			registry, err := loadNormalizedConnectionRegistry()
			if err != nil {
				return err
			}
			var ref resolvedTerminalRef
			var endpoint endpointdomain.Endpoint
			if len(args) == 1 {
				ref, err = resolveTerminalRef(args[0], runtime.requestedEndpoint(), registry)
				if err != nil {
					return err
				}
				endpoint = registry.Endpoints[ref.EndpointID]
			} else {
				endpoint, err = resolveEndpointConfig(runtime.requestedEndpoint(), registry)
				if err != nil {
					return err
				}
				ref.EndpointID = endpoint.ID
			}
			client, closeClient, err := runtime.open(cmd, endpoint)
			if err != nil {
				return err
			}
			defer closeClient()
			command := &apipb.EventSubscribeCommand{Types: eventTypes}
			if ref.TerminalID != "" {
				command.Terminal = &apipb.TerminalRef{EndpointId: string(ref.EndpointID), TerminalId: ref.TerminalID}
			}
			_, events, err := client.EventSubscribe(ctx, command)
			if err != nil {
				return classifyCLIError(err)
			}
			narrowHumanOutput := false
			if output == "human" {
				width := cliTerminalWidth(cmd.OutOrStdout())
				narrowHumanOutput = width > 0 && width < 56
				if !narrowHumanOutput {
					if err := writeCLIFixedRow(cmd.OutOrStdout(), []int{30, 16}, "TIME", "TYPE", "TARGET"); err != nil {
						return err
					}
				}
			}
			written := 0
			for count == 0 || written < count {
				select {
				case <-ctx.Done():
					return classifyCLIError(ctx.Err())
				case event, ok := <-events:
					if !ok {
						return nil
					}
					view := terminalEventView(endpoint.ID, event)
					if output == "ndjson" {
						if err := json.NewEncoder(cmd.OutOrStdout()).Encode(view); err != nil {
							return err
						}
					} else if narrowHumanOutput {
						if written > 0 {
							if _, err := io.WriteString(cmd.OutOrStdout(), "\n"); err != nil {
								return err
							}
						}
						if err := writeCLIFields(cmd.OutOrStdout(),
							cliField{Label: "Time", Value: view.Timestamp},
							cliField{Label: "Type", Value: view.Type},
							cliField{Label: "Target", Value: view.Target},
						); err != nil {
							return err
						}
					} else if err := writeCLIFixedRow(cmd.OutOrStdout(), []int{30, 16}, view.Timestamp, view.Type, view.Target); err != nil {
						return err
					}
					written++
				}
			}
			return nil
		},
	}
	command.Flags().StringVar(&output, "output", "human", "human or ndjson")
	command.Flags().StringArrayVar(&types, "type", nil, "event type filter (repeatable)")
	command.Flags().IntVar(&count, "count", 0, "stop after N events (0 streams until canceled)")
	command.Flags().DurationVar(&timeout, "timeout", 0, "optional stream timeout")
	command.Flags().BoolVar(&jsonOutput, "json", false, "write NDJSON event records")
	return command
}

func parseTerminalEventTypes(values []string) ([]apipb.ApplicationEventType, error) {
	if len(values) == 0 {
		return []apipb.ApplicationEventType{apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE}, nil
	}
	names := map[string]apipb.ApplicationEventType{
		"created": apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE, "state": apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE,
		"removed": apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE, "metadata": apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE,
	}
	out := make([]apipb.ApplicationEventType, 0, len(values))
	for _, value := range values {
		typ, ok := names[strings.ToLower(strings.TrimSpace(value))]
		if !ok {
			return nil, usageCLIError(fmt.Sprintf("unknown terminal event type %q", value))
		}
		out = append(out, typ)
	}
	return out, nil
}

func terminalEventView(endpointID endpointdomain.EndpointID, event *apipb.EventEnvelope) terminalEventEnvelope {
	view := terminalEventEnvelope{
		SchemaVersion: 1, Kind: "terminal_event", EndpointID: string(endpointID),
		Type: terminalEventTypeName(event), Timestamp: formatTerminalUnixNano(event.GetTimestampUnixNano()),
	}
	if terminal := event.GetTerminalLifecycle().GetTerminal(); terminal != nil {
		view.Target = string(endpointID) + ":" + terminal.GetRef().GetTerminalId()
		view.Data = map[string]any{"state": terminalStateString(terminal.GetState()), "name": terminal.GetName(), "command": terminal.GetCommand(), "cols": terminal.GetSize().GetCols(), "rows": terminal.GetSize().GetRows(), "exit_code": terminal.ExitCode}
	}
	return view
}

func terminalEventTypeName(event *apipb.EventEnvelope) string {
	switch event.GetEvent().(type) {
	case *apipb.EventEnvelope_TerminalLifecycle:
		return "lifecycle"
	default:
		return "unknown"
	}
}

func openTerminalAutomationTarget(ctx context.Context, cmd *cobra.Command, runtime terminalCommandRuntime, value string) (terminalAutomationTarget, error) {
	registry, err := loadNormalizedConnectionRegistry()
	if err != nil {
		return terminalAutomationTarget{}, err
	}
	ref, err := resolveTerminalRef(value, runtime.requestedEndpoint(), registry)
	if err != nil {
		return terminalAutomationTarget{}, err
	}
	endpoint := registry.Endpoints[ref.EndpointID]
	client, closeClient, err := runtime.open(cmd, endpoint)
	if err != nil {
		return terminalAutomationTarget{}, err
	}
	return terminalAutomationTarget{Ref: ref, Endpoint: endpoint, Client: client, Close: closeClient}, nil
}

func attachTerminalAutomation(ctx context.Context, client terminalProtocolClient, ref resolvedTerminalRef, resizePolicy apipb.ResizePolicy, operation string) (*apipb.TerminalAttachResult, string, func(), error) {
	// send/resize 只能通过 owning daemon 签发的临时 attachment；操作结束立即 detach，不持有隐式 CLI session。
	identity := attachmentIdentity(ref, operation)
	result, err := client.TerminalAttach(ctx, &apipb.TerminalAttachCommand{Terminal: &apipb.TerminalRef{EndpointId: string(ref.EndpointID), TerminalId: ref.TerminalID}, Mode: apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, ResizePolicy: resizePolicy, SurfaceId: identity, ViewId: identity})
	if err != nil {
		return nil, "", func() {}, err
	}
	detach := func() {
		detachContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = client.TerminalDetach(detachContext, &apipb.TerminalDetachCommand{Attachment: result.GetAttachment().GetResource()})
	}
	return result, identity, detach, nil
}

func resizeControlReasonString(reason apipb.ResizeControlReason) string {
	switch reason {
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OWNER:
		return "owner"
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_FOLLOWER:
		return "follower"
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OBSERVER:
		return "observer"
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_SIZE_LOCKED:
		return "size_locked"
	default:
		return ""
	}
}

func attachmentIdentity(ref resolvedTerminalRef, operation string) string {
	return fmt.Sprintf("cli:%s:%s:%d:%d", operation, ref.String(), os.Getpid(), time.Now().UnixNano())
}

func terminalCommandContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	if timeout <= 0 {
		return nil, func() {}, usageCLIError("--timeout must be positive")
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	return ctx, cancel, nil
}

func terminalCommandContextOptional(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	if timeout < 0 {
		return nil, func() {}, usageCLIError("--timeout cannot be negative")
	}
	if timeout == 0 {
		ctx, cancel := context.WithCancel(parent)
		return ctx, cancel, nil
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	return ctx, cancel, nil
}
