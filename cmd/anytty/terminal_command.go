package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	endpointdomain "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/spf13/cobra"
)

type terminalProtocolClient interface {
	TerminalDefaults(context.Context, *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error)
	TerminalCreate(context.Context, *apipb.TerminalCreateCommand) (*apipb.TerminalCreateResult, error)
	TerminalList(context.Context, *apipb.TerminalListCommand) (*apipb.TerminalListResult, error)
	TerminalGet(context.Context, *apipb.TerminalGetCommand) (*apipb.TerminalGetResult, error)
	TerminalRestart(context.Context, *apipb.TerminalRestartCommand) error
	TerminalKill(context.Context, *apipb.TerminalKillCommand) error
	TerminalRemove(context.Context, *apipb.TerminalRemoveCommand) error
	TerminalSetTags(context.Context, *apipb.TerminalSetTagsCommand) error
	TerminalSetMetadata(context.Context, *apipb.TerminalSetMetadataCommand) error
	TerminalAttach(context.Context, *apipb.TerminalAttachCommand) (*apipb.TerminalAttachResult, error)
	TerminalDetach(context.Context, *apipb.TerminalDetachCommand) error
	TerminalInput(context.Context, *apipb.TerminalInputCommand) error
	OpenResourceStream(*apipb.ResourceHandle) (clientruntime.ResourceStream, error)
	TerminalResize(context.Context, *apipb.TerminalResizeCommand) (*apipb.TerminalResizeResult, error)
	HistoryWindow(context.Context, *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error)
	HistoryCopy(context.Context, *apipb.HistoryCopyCommand) (*apipb.HistoryCopyResult, error)
	HistoryRelease(context.Context, *apipb.HistoryReleaseCommand) error
	LiveScreenNext(context.Context, *apipb.LiveScreenNextCommand) (*apipb.NativeScreenResult, error)
	EventSubscribe(context.Context, *apipb.EventSubscribeCommand) (*apipb.EventSubscriptionResult, <-chan *apipb.EventEnvelope, error)
	Close() error
}

type terminalCommandRuntime struct {
	socket     *string
	logFile    *string
	configPath *string
	endpointID *string
}

type terminalView struct {
	Target     string            `json:"target"`
	EndpointID string            `json:"endpoint_id"`
	TerminalID string            `json:"terminal_id"`
	Name       string            `json:"name"`
	Command    []string          `json:"command"`
	CWD        string            `json:"cwd,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	State      string            `json:"state"`
	Cols       uint16            `json:"cols"`
	Rows       uint16            `json:"rows"`
	CreatedAt  string            `json:"created_at,omitempty"`
	ExitCode   *int              `json:"exit_code,omitempty"`
	ExitedAt   string            `json:"exited_at,omitempty"`
}

type terminalListEnvelope struct {
	SchemaVersion int            `json:"schema_version"`
	Kind          string         `json:"kind"`
	Items         []terminalView `json:"items"`
}

type terminalItemEnvelope struct {
	SchemaVersion int          `json:"schema_version"`
	Kind          string       `json:"kind"`
	Item          terminalView `json:"item"`
}

type commandResultEnvelope struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Target        string `json:"target"`
	State         string `json:"state,omitempty"`
}

type cliError struct {
	code    int
	message string
	cause   error
}

func (err *cliError) Error() string {
	if err == nil {
		return "command failed"
	}
	return err.message
}

func (err *cliError) Unwrap() error { return err.cause }

func newTerminalCommand(runtime terminalCommandRuntime) *cobra.Command {
	endpointID := ""
	runtime.endpointID = &endpointID
	command := &cobra.Command{
		Use:   "terminal",
		Short: "Manage terminals owned by a daemon endpoint",
		Long:  "Manage terminal lifecycle through its owning local, Direct, or SSH daemon endpoint.",
	}
	command.PersistentFlags().StringVar(runtime.endpointID, "endpoint", "", "owning endpoint ID (default: registry default)")
	command.AddCommand(
		newTerminalCreateCommand(runtime, "create"),
		newTerminalListCommand(runtime, "list"),
		newTerminalShowCommand(runtime),
		newTerminalAttachCommand(runtime, "attach"),
		newTerminalMutationCommand(runtime, "restart", "Restart a terminal from its daemon-owned process specification", terminalRestart),
		newTerminalMutationCommand(runtime, "kill", "Stop a terminal process and preserve its record and history", terminalKill),
		newTerminalMutationCommand(runtime, "remove", "Remove an exited terminal record", terminalRemove),
		newTerminalRenameCommand(runtime),
		newTerminalTagCommand(runtime),
		newTerminalSendCommand(runtime),
		newTerminalStreamCommand(runtime),
		newTerminalCaptureCommand(runtime),
		newTerminalResizeCommand(runtime),
		newTerminalWaitCommand(runtime),
		newTerminalEventsCommand(runtime),
	)
	return command
}

func newTerminalAliasCommands(runtime terminalCommandRuntime) []*cobra.Command {
	wrap := func(build func(terminalCommandRuntime) *cobra.Command) *cobra.Command {
		copyRuntime := runtime
		endpointID := ""
		copyRuntime.endpointID = &endpointID
		command := build(copyRuntime)
		command.Flags().StringVar(copyRuntime.endpointID, "endpoint", "", "owning endpoint ID (default: registry default)")
		return command
	}
	return []*cobra.Command{
		wrap(func(value terminalCommandRuntime) *cobra.Command { return newTerminalCreateCommand(value, "new") }),
		wrap(func(value terminalCommandRuntime) *cobra.Command { return newTerminalListCommand(value, "ls") }),
		wrap(func(value terminalCommandRuntime) *cobra.Command { return newTerminalAttachCommand(value, "attach") }),
		wrap(func(value terminalCommandRuntime) *cobra.Command {
			return newTerminalMutationCommand(value, "kill", "Stop a terminal process and preserve its record and history", terminalKill)
		}),
		wrap(func(value terminalCommandRuntime) *cobra.Command {
			return newTerminalMutationCommand(value, "rm", "Remove an exited terminal record", terminalRemove)
		}),
	}
}

func (runtime terminalCommandRuntime) requestedEndpoint() string {
	if runtime.endpointID == nil {
		return ""
	}
	return *runtime.endpointID
}

func (runtime terminalCommandRuntime) open(cmd *cobra.Command, cfg endpointdomain.Endpoint) (terminalProtocolClient, func(), error) {
	// terminal lifecycle 与 metadata truth 始终来自 owning endpoint 的 daemon client。
	// 参数已经在进入本函数前完成校验；transport/protocol 失败不得附带 Cobra usage。
	cmd.Root().SilenceUsage = true
	var client *protocoladapter.ApplicationClient
	var closeClient func()
	var err error
	client, closeClient, err = openEndpointProtocolClient(cmd.Context(), cfg, *runtime.socket, *runtime.logFile)
	if err != nil {
		return nil, func() {}, classifyCLIError(err)
	}
	return client, closeClient, nil
}

func newTerminalCreateCommand(runtime terminalCommandRuntime, use string) *cobra.Command {
	var name, cwd string
	var environment []string
	var tags map[string]string
	var cols, rows uint16
	var jsonOutput, attach bool
	command := &cobra.Command{
		Use:   use + " [-- COMMAND...]",
		Short: "Create a terminal on an owning daemon endpoint",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if (cols == 0) != (rows == 0) {
				return usageCLIError("--cols and --rows must be provided together")
			}
			registry, err := loadNormalizedConnectionRegistry()
			if err != nil {
				return err
			}
			endpoint, err := resolveEndpointConfig(runtime.requestedEndpoint(), registry)
			if err != nil {
				return err
			}
			client, closeClient, err := runtime.open(cmd, endpoint)
			if err != nil {
				return err
			}
			defer closeClient()
			if len(args) == 0 {
				defaults, defaultErr := client.TerminalDefaults(cmd.Context(), &apipb.TerminalDefaultsCommand{})
				if defaultErr != nil {
					return classifyCLIError(defaultErr)
				}
				args = append([]string(nil), defaults.GetDefaults().GetDefaultCommand()...)
				if len(args) == 0 {
					return &cliError{code: 6, message: fmt.Sprintf("endpoint %s returned no default command", endpoint.ID)}
				}
				if strings.TrimSpace(cwd) == "" {
					cwd = defaults.GetDefaults().GetDefaultCwd()
				}
			}
			size := &apipb.TerminalSize{Cols: uint32(cols), Rows: uint32(rows)}
			if cols == 0 {
				current := currentSize()
				size = &apipb.TerminalSize{Cols: uint32(current.Cols), Rows: uint32(current.Rows)}
			}
			terminalID := newV3TerminalID()
			if strings.TrimSpace(name) != "" {
				terminalID = strings.TrimSpace(name)
			}
			created, err := client.TerminalCreate(cmd.Context(), &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
				TerminalId: terminalID, Name: strings.TrimSpace(name), Command: append([]string(nil), args...), Cwd: strings.TrimSpace(cwd), Env: append([]string(nil), environment...), Tags: cloneTerminalTags(tags), Size: size,
			}})
			if err != nil {
				return classifyCLIError(err)
			}
			createdInfo := created.GetTerminal()
			createdID := createdInfo.GetRef().GetTerminalId()
			target := string(endpoint.ID) + ":" + createdID
			if jsonOutput {
				if err := json.NewEncoder(cmd.OutOrStdout()).Encode(commandResultEnvelope{SchemaVersion: 1, Kind: "terminal_created", Target: target, State: terminalStateString(createdInfo.GetState())}); err != nil {
					return err
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), target)
			}
			if attach {
				return runAttachCommand(cmd, string(endpoint.ID), createdID, *runtime.socket, *runtime.logFile, *runtime.configPath)
			}
			return nil
		},
	}
	command.Flags().StringVar(&name, "name", "", "stable daemon-local terminal name")
	command.Flags().StringVar(&cwd, "cwd", "", "working directory on the daemon host")
	command.Flags().StringArrayVar(&environment, "env", nil, "environment entry KEY=VALUE (repeatable)")
	command.Flags().StringToStringVar(&tags, "tag", nil, "terminal tag KEY=VALUE (repeatable)")
	command.Flags().Uint16Var(&cols, "cols", 0, "initial terminal columns")
	command.Flags().Uint16Var(&rows, "rows", 0, "initial terminal rows")
	command.Flags().BoolVar(&attach, "attach", false, "attach after creation")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newTerminalListCommand(runtime terminalCommandRuntime, use string) *cobra.Command {
	var jsonOutput, noHeader, allEndpoints bool
	var stateFilter, format string
	var tagFilters []string
	command := &cobra.Command{
		Use:   use,
		Short: "List terminals from one or all daemon endpoints",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if jsonOutput && format != "" {
				return usageCLIError("--json and --format are mutually exclusive")
			}
			registry, err := loadNormalizedConnectionRegistry()
			if err != nil {
				return err
			}
			configs := make([]endpointdomain.Endpoint, 0, len(registry.Endpoints))
			if allEndpoints {
				if runtime.requestedEndpoint() != "" {
					return usageCLIError("--all-endpoints and --endpoint are mutually exclusive")
				}
				for _, cfg := range registry.List() {
					if cfg.Enabled {
						configs = append(configs, cfg)
					}
				}
			} else {
				cfg, err := resolveEndpointConfig(runtime.requestedEndpoint(), registry)
				if err != nil {
					return err
				}
				configs = append(configs, cfg)
			}
			views := make([]terminalView, 0)
			for _, cfg := range configs {
				client, closeClient, openErr := runtime.open(cmd, cfg)
				if openErr != nil {
					return openErr
				}
				result, listErr := client.TerminalList(cmd.Context(), &apipb.TerminalListCommand{})
				closeClient()
				if listErr != nil {
					return classifyCLIError(listErr)
				}
				for _, item := range result.Terminals {
					if stateFilter != "" && terminalStateString(item.GetState()) != stateFilter || !terminalMatchesTags(item, tagFilters) {
						continue
					}
					views = append(views, terminalInfoView(cfg.ID, item))
				}
			}
			sort.Slice(views, func(i, j int) bool { return views[i].Target < views[j].Target })
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(terminalListEnvelope{SchemaVersion: 1, Kind: "terminal_list", Items: views})
			}
			if format != "" {
				return writeTerminalFormat(cmd.OutOrStdout(), format, views)
			}
			return writeTerminalTable(cmd.OutOrStdout(), views, noHeader)
		},
	}
	command.Flags().StringVar(&stateFilter, "state", "", "filter by exact lifecycle state")
	command.Flags().StringArrayVar(&tagFilters, "tag", nil, "filter by KEY or KEY=VALUE (repeatable)")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	command.Flags().BoolVar(&noHeader, "no-header", false, "omit the human table header")
	command.Flags().BoolVar(&allEndpoints, "all-endpoints", false, "aggregate every enabled endpoint")
	command.Flags().StringVar(&format, "format", "", "Go template over stable lowercase terminal fields")
	return command
}

func newTerminalShowCommand(runtime terminalCommandRuntime) *cobra.Command {
	var jsonOutput bool
	var format string
	command := &cobra.Command{
		Use:   "show TARGET",
		Short: "Show one terminal from its owning daemon endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput && format != "" {
				return usageCLIError("--json and --format are mutually exclusive")
			}
			registry, err := loadNormalizedConnectionRegistry()
			if err != nil {
				return err
			}
			ref, err := resolveTerminalRef(args[0], runtime.requestedEndpoint(), registry)
			if err != nil {
				return err
			}
			client, closeClient, err := runtime.open(cmd, registry.Endpoints[ref.EndpointID])
			if err != nil {
				return err
			}
			defer closeClient()
			item, err := findTerminal(cmd.Context(), client, ref)
			if err != nil {
				return err
			}
			view := terminalInfoView(ref.EndpointID, item)
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(terminalItemEnvelope{SchemaVersion: 1, Kind: "terminal", Item: view})
			}
			if format != "" {
				return writeTerminalFormat(cmd.OutOrStdout(), format, []terminalView{view})
			}
			return writeTerminalDetail(cmd.OutOrStdout(), view)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	command.Flags().StringVar(&format, "format", "", "Go template over stable lowercase terminal fields")
	return command
}

func newTerminalAttachCommand(runtime terminalCommandRuntime, use string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " TARGET",
		Short: "Attach the TUI to a terminal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, err := loadNormalizedConnectionRegistry()
			if err != nil {
				return err
			}
			ref, err := resolveTerminalRef(args[0], runtime.requestedEndpoint(), registry)
			if err != nil {
				return err
			}
			cmd.Root().SilenceUsage = true
			return runAttachCommand(cmd, string(ref.EndpointID), ref.TerminalID, *runtime.socket, *runtime.logFile, *runtime.configPath)
		},
	}
}

type terminalMutation int

const (
	terminalRestart terminalMutation = iota + 1
	terminalKill
	terminalRemove
)

func newTerminalMutationCommand(runtime terminalCommandRuntime, use, short string, mutation terminalMutation) *cobra.Command {
	var jsonOutput, quiet bool
	command := &cobra.Command{
		Use:   use + " TARGET",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, err := loadNormalizedConnectionRegistry()
			if err != nil {
				return err
			}
			ref, err := resolveTerminalRef(args[0], runtime.requestedEndpoint(), registry)
			if err != nil {
				return err
			}
			client, closeClient, err := runtime.open(cmd, registry.Endpoints[ref.EndpointID])
			if err != nil {
				return err
			}
			defer closeClient()
			item, err := findTerminal(cmd.Context(), client, ref)
			if err != nil {
				return err
			}
			switch mutation {
			case terminalRestart:
				err = client.TerminalRestart(cmd.Context(), &apipb.TerminalRestartCommand{Terminal: item.GetRef()})
			case terminalKill:
				err = client.TerminalKill(cmd.Context(), &apipb.TerminalKillCommand{Terminal: item.GetRef()})
			case terminalRemove:
				if item.GetState() == apipb.TerminalState_TERMINAL_STATE_RUNNING {
					return &cliError{code: 4, message: fmt.Sprintf("terminal %s is running; kill it before remove", ref.String())}
				}
				err = client.TerminalRemove(cmd.Context(), &apipb.TerminalRemoveCommand{Terminal: item.GetRef()})
			}
			if err != nil {
				return classifyCLIError(err)
			}
			if quiet {
				return nil
			}
			kind := "terminal_restarted"
			if mutation == terminalKill {
				kind = "terminal_killed"
			} else if mutation == terminalRemove {
				kind = "terminal_removed"
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(commandResultEnvelope{SchemaVersion: 1, Kind: kind, Target: ref.String()})
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Target", Value: ref.String()},
				cliField{Label: "Status", Value: strings.TrimPrefix(kind, "terminal_")},
			)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	command.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress success output")
	return command
}

func newTerminalRenameCommand(runtime terminalCommandRuntime) *cobra.Command {
	return &cobra.Command{
		Use:   "rename TARGET NAME",
		Short: "Rename a terminal without changing its stable ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(args[1]) == "" {
				return usageCLIError("terminal name cannot be empty")
			}
			registry, err := loadNormalizedConnectionRegistry()
			if err != nil {
				return err
			}
			ref, err := resolveTerminalRef(args[0], runtime.requestedEndpoint(), registry)
			if err != nil {
				return err
			}
			client, closeClient, err := runtime.open(cmd, registry.Endpoints[ref.EndpointID])
			if err != nil {
				return err
			}
			defer closeClient()
			item, err := findTerminal(cmd.Context(), client, ref)
			if err != nil {
				return err
			}
			if err := client.TerminalSetMetadata(cmd.Context(), &apipb.TerminalSetMetadataCommand{Terminal: item.GetRef(), Name: strings.TrimSpace(args[1]), Tags: cloneTerminalTags(item.GetTags())}); err != nil {
				return classifyCLIError(err)
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Target", Value: ref.String()},
				cliField{Label: "Name", Value: strings.TrimSpace(args[1])},
				cliField{Label: "Status", Value: "renamed"},
			)
		},
	}
}

func newTerminalTagCommand(runtime terminalCommandRuntime) *cobra.Command {
	var removeKeys []string
	command := &cobra.Command{
		Use:   "tag TARGET [KEY=VALUE...]",
		Short: "Set or remove terminal tags",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && len(removeKeys) == 0 {
				return usageCLIError("provide at least one KEY=VALUE or --remove KEY")
			}
			updates, err := parseTerminalTags(args[1:])
			if err != nil {
				return err
			}
			registry, err := loadNormalizedConnectionRegistry()
			if err != nil {
				return err
			}
			ref, err := resolveTerminalRef(args[0], runtime.requestedEndpoint(), registry)
			if err != nil {
				return err
			}
			client, closeClient, err := runtime.open(cmd, registry.Endpoints[ref.EndpointID])
			if err != nil {
				return err
			}
			defer closeClient()
			item, err := findTerminal(cmd.Context(), client, ref)
			if err != nil {
				return err
			}
			tags := cloneTerminalTags(item.GetTags())
			if tags == nil {
				tags = make(map[string]string)
			}
			for key, value := range updates {
				tags[key] = value
			}
			for _, key := range removeKeys {
				key = strings.TrimSpace(key)
				if key == "" {
					return usageCLIError("tag remove key cannot be empty")
				}
				delete(tags, key)
			}
			if err := client.TerminalSetTags(cmd.Context(), &apipb.TerminalSetTagsCommand{Terminal: item.GetRef(), Tags: tags}); err != nil {
				return classifyCLIError(err)
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Target", Value: ref.String()},
				cliField{Label: "Status", Value: "tags updated"},
			)
		},
	}
	command.Flags().StringArrayVar(&removeKeys, "remove", nil, "remove a tag key (repeatable)")
	return command
}

func findTerminal(ctx context.Context, client terminalProtocolClient, ref resolvedTerminalRef) (*apipb.TerminalInfo, error) {
	// TerminalRef 已在 registry 边界解析；查询只进入 owning daemon，CLI 不建立第二份 terminal inventory。
	result, err := client.TerminalGet(ctx, &apipb.TerminalGetCommand{Terminal: &apipb.TerminalRef{EndpointId: string(ref.EndpointID), TerminalId: ref.TerminalID}})
	if err != nil {
		return nil, classifyCLIError(err)
	}
	if result.GetTerminal() == nil {
		return nil, &cliError{code: 3, message: fmt.Sprintf("terminal %s was not found", ref.String())}
	}
	return result.GetTerminal(), nil
}

func terminalInfoView(endpointID endpointdomain.EndpointID, item *apipb.TerminalInfo) terminalView {
	exitCode := int32PointerToCLIInt(item.ExitCode)
	return terminalView{
		Target: string(endpointID) + ":" + item.GetRef().GetTerminalId(), EndpointID: string(endpointID), TerminalID: item.GetRef().GetTerminalId(),
		Name: item.GetName(), Command: append([]string(nil), item.GetCommand()...), CWD: item.GetCwd(),
		Tags: cloneTerminalTags(item.GetTags()), State: terminalStateString(item.GetState()), Cols: uint16(item.GetSize().GetCols()), Rows: uint16(item.GetSize().GetRows()),
		CreatedAt: formatTerminalUnixNano(item.GetCreatedAtUnixNano()), ExitCode: exitCode, ExitedAt: formatTerminalUnixNano(item.GetExitedAtUnixNano()),
	}
}

func formatTerminalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func formatTerminalUnixNano(value int64) string {
	if value == 0 {
		return ""
	}
	return formatTerminalTime(time.Unix(0, value))
}

func int32PointerToCLIInt(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func terminalStateString(value apipb.TerminalState) string {
	switch value {
	case apipb.TerminalState_TERMINAL_STATE_CREATED:
		return "created"
	case apipb.TerminalState_TERMINAL_STATE_RUNNING:
		return "running"
	case apipb.TerminalState_TERMINAL_STATE_EXITED:
		return "exited"
	case apipb.TerminalState_TERMINAL_STATE_REMOVED:
		return "removed"
	default:
		return ""
	}
}

func writeTerminalTable(writer io.Writer, views []terminalView, noHeader bool) error {
	rows := make([][]string, 0, len(views))
	for _, view := range views {
		rows = append(rows, []string{view.Target, view.Name, view.State, fmt.Sprintf("%dx%d", view.Cols, view.Rows), strings.Join(view.Command, " ")})
	}
	var header []string
	if !noHeader {
		header = []string{"TARGET", "NAME", "STATE", "SIZE", "COMMAND"}
	}
	return writeCLITable(writer, header, rows)
}

func writeTerminalDetail(writer io.Writer, view terminalView) error {
	return writeCLIFields(writer,
		cliField{Label: "Target", Value: view.Target},
		cliField{Label: "Name", Value: view.Name},
		cliField{Label: "State", Value: view.State},
		cliField{Label: "Command", Value: strings.Join(view.Command, " ")},
		cliField{Label: "CWD", Value: view.CWD},
		cliField{Label: "Size", Value: fmt.Sprintf("%dx%d", view.Cols, view.Rows)},
	)
}

func terminalMatchesTags(item *apipb.TerminalInfo, filters []string) bool {
	for _, filter := range filters {
		key, value, hasValue := strings.Cut(filter, "=")
		actual, exists := item.GetTags()[strings.TrimSpace(key)]
		if !exists || hasValue && actual != value {
			return false
		}
	}
	return true
}

func parseTerminalTags(values []string) (map[string]string, error) {
	tags := make(map[string]string, len(values))
	for _, value := range values {
		key, tagValue, found := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, usageCLIError("tags must use KEY=VALUE")
		}
		tags[key] = tagValue
	}
	return tags, nil
}

func cloneTerminalTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for key, value := range tags {
		out[key] = value
	}
	return out
}

func usageCLIError(message string) error { return &cliError{code: 2, message: message} }

func classifyCLIError(err error) error {
	if err == nil {
		return nil
	}
	var classified *cliError
	if errors.As(err, &classified) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &cliError{code: 7, message: err.Error(), cause: err}
	}
	var requestError *protocol.RequestError
	if errors.As(err, &requestError) {
		// 退出码只读取 daemon 的 typed protocol code，禁止根据可变错误文案猜测领域状态。
		code := 1
		switch requestError.Code {
		case 400:
			code = 4
		case 401, 403:
			code = 5
		case 404:
			code = 3
		case 503:
			code = 6
		}
		return &cliError{code: code, message: requestError.Message, cause: err}
	}
	if runtimeCode := clientruntime.CodeOf(err); runtimeCode != clientruntime.ErrorUnavailable {
		code := 6
		switch runtimeCode {
		case clientruntime.ErrorInvalidRequest:
			code = 4
		case clientruntime.ErrorAuthorization:
			code = 5
		case clientruntime.ErrorNotFound:
			code = 3
		case clientruntime.ErrorCanceled:
			code = 7
		}
		return &cliError{code: code, message: err.Error(), cause: err}
	}
	return &cliError{code: 6, message: err.Error(), cause: err}
}

func cliExitCode(err error) int {
	if err == nil {
		return 0
	}
	var classified *cliError
	if errors.As(err, &classified) {
		return classified.code
	}
	return 1
}
