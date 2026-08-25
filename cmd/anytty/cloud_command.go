package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	cloudclient "github.com/anytty/anytty/cloud/client"
	clouddaemon "github.com/anytty/anytty/cloud/daemon"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// defaultCloudControllerOrigin 是官方 AnyTTY Cloud API 域名，作为注册默认值内置。
const defaultCloudControllerOrigin = "https://api.anytty.com"

func cloudCommand(socket, logFile, configPath *string) *cobra.Command {
	command := &cobra.Command{Use: "cloud", Short: "Manage AnyTTY Cloud enrollment", Args: cobra.NoArgs}
	command.AddCommand(cloudEnrollCommand())
	command.AddCommand(cloudStatusCommand(socket, logFile, configPath))
	command.AddCommand(cloudEnableCommand(socket, logFile, configPath))
	command.AddCommand(cloudDisableCommand(socket, logFile, configPath))
	command.AddCommand(cloudEdgeCommand(socket, logFile))
	return command
}

func cloudStatusCommand(socket, logFile, configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "status", Aliases: []string{"state", "states"}, Short: "Show local daemon Cloud runtime state", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		status, err := readCloudStatusForCLI(cmd, socket, logFile, configPath)
		if err != nil {
			return err
		}
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		}
		return writeCloudStatus(cmd, status)
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func cloudEnableCommand(socket, logFile, configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "enable", Aliases: []string{"resume", "on"}, Short: "Resume Cloud runtime without re-enrolling", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := removeV3CloudDisabled(); err != nil {
			return err
		}
		status, err := runCloudControlWhenRunning(cmd, socket, logFile, configPath, func(ctx context.Context, application localApplicationSession) (*apipb.RemoteCloudStatusResult, error) {
			return application.RemoteCloudEnable(ctx, &apipb.RemoteCloudEnableCommand{})
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Cloud runtime enabled.")
		return writeCloudStatus(cmd, status)
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func cloudDisableCommand(socket, logFile, configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "disable", Aliases: []string{"pause", "off"}, Short: "Temporarily stop using Cloud without deleting enrollment", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := writeV3CloudDisabled("disabled by local command"); err != nil {
			return err
		}
		status, err := runCloudControlWhenRunning(cmd, socket, logFile, configPath, func(ctx context.Context, application localApplicationSession) (*apipb.RemoteCloudStatusResult, error) {
			return application.RemoteCloudDisable(ctx, &apipb.RemoteCloudDisableCommand{})
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Cloud runtime disabled.")
		return writeCloudStatus(cmd, status)
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func cloudEdgeCommand(socket, logFile *string) *cobra.Command {
	command := &cobra.Command{Use: "edge", Short: "Inspect and reselect the daemon Cloud Edge", Args: cobra.NoArgs}
	command.AddCommand(cloudEdgeListCommand(socket, logFile), cloudEdgePreferCommand(socket, logFile), cloudEdgeReselectCommand(socket, logFile))
	return command
}

func cloudEdgeListCommand(socket, logFile *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "list", Short: "Probe and rank available Edge servers", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		selection, err := callCloudEdge(cmd, socket, logFile, func(ctx context.Context, application localApplicationSession) (*apipb.RemoteCloudEdgesResult, error) {
			return application.RemoteCloudEdges(ctx, &apipb.RemoteCloudEdgesCommand{})
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(selection)
		}
		return writeCloudEdgeSelection(cmd, selection)
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func cloudEdgePreferCommand(socket, logFile *string) *cobra.Command {
	return &cobra.Command{Use: "prefer EDGE_ID_OR_NAME", Short: "Prefer one Edge and immediately reselect", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		current, err := callCloudEdge(cmd, socket, logFile, func(ctx context.Context, application localApplicationSession) (*apipb.RemoteCloudEdgesResult, error) {
			return application.RemoteCloudEdges(ctx, &apipb.RemoteCloudEdgesCommand{})
		})
		if err != nil {
			return err
		}
		edgeID, err := resolveCloudEdgeSelector(current, args[0])
		if err != nil {
			return err
		}
		selection, err := callCloudEdge(cmd, socket, logFile, func(ctx context.Context, application localApplicationSession) (*apipb.RemoteCloudEdgesResult, error) {
			return application.RemoteCloudPreferEdge(ctx, &apipb.RemoteCloudPreferEdgeCommand{EdgeId: edgeID, ExpectedPreferenceRevision: current.GetPreferenceRevision()})
		})
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Edge preference saved; Cloud connection is reselecting without restarting the daemon.")
		return writeCloudEdgeSelection(cmd, selection)
	}}
}

func cloudEdgeReselectCommand(socket, logFile *string) *cobra.Command {
	return &cobra.Command{Use: "reselect", Short: "Probe and reselect an Edge without restarting the daemon", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		selection, err := callCloudEdge(cmd, socket, logFile, func(ctx context.Context, application localApplicationSession) (*apipb.RemoteCloudEdgesResult, error) {
			return application.RemoteCloudReselectEdge(ctx, &apipb.RemoteCloudReselectEdgeCommand{})
		})
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Cloud Edge reselected without restarting the daemon.")
		return writeCloudEdgeSelection(cmd, selection)
	}}
}

type localApplicationSession interface {
	RemoteCloudStatus(context.Context, *apipb.RemoteCloudStatusCommand) (*apipb.RemoteCloudStatusResult, error)
	RemoteCloudEnable(context.Context, *apipb.RemoteCloudEnableCommand) (*apipb.RemoteCloudStatusResult, error)
	RemoteCloudDisable(context.Context, *apipb.RemoteCloudDisableCommand) (*apipb.RemoteCloudStatusResult, error)
	RemoteCloudEdges(context.Context, *apipb.RemoteCloudEdgesCommand) (*apipb.RemoteCloudEdgesResult, error)
	RemoteCloudPreferEdge(context.Context, *apipb.RemoteCloudPreferEdgeCommand) (*apipb.RemoteCloudEdgesResult, error)
	RemoteCloudReselectEdge(context.Context, *apipb.RemoteCloudReselectEdgeCommand) (*apipb.RemoteCloudEdgesResult, error)
}

func callCloudEdge(cmd *cobra.Command, socket, logFile *string, call func(context.Context, localApplicationSession) (*apipb.RemoteCloudEdgesResult, error)) (*cloudv1.DaemonEdgeSelection, error) {
	ctx := cmd.Context()
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
	}
	application, client, err := dialLocalApplicationSession(ctx, resolveV3Socket(*socket), resolveV3LogFilePath(*logFile))
	if err != nil {
		return nil, err
	}
	defer client.Close()
	response, err := call(ctx, application)
	if err != nil {
		return nil, err
	}
	if response.GetSelection() == nil {
		return nil, errors.New("daemon returned no Edge selection")
	}
	return response.GetSelection(), nil
}

func readCloudStatusForCLI(cmd *cobra.Command, socket, logFile, configPath *string) (cloudStatusView, error) {
	socketPath := resolveV3Socket(*socket)
	daemon, _, err := daemonStatus(socketPath, *logFile, *configPath)
	if err != nil {
		return cloudStatusView{}, err
	}
	if daemon.State == "running" {
		response, err := callCloudStatus(cmd, socket, logFile, func(ctx context.Context, application localApplicationSession) (*apipb.RemoteCloudStatusResult, error) {
			return application.RemoteCloudStatus(ctx, &apipb.RemoteCloudStatusCommand{})
		})
		if err != nil {
			return cloudStatusView{}, err
		}
		return cloudStatusViewFromProto(response), nil
	}
	status, err := loadV3CloudStatus(v3CloudEnrollmentRecordPath(), v3CloudDisabledPath(), nil, false, "")
	if err != nil {
		return cloudStatusView{}, err
	}
	return cloudStatusViewFromCore(status), nil
}

func runCloudControlWhenRunning(cmd *cobra.Command, socket, logFile, configPath *string, call func(context.Context, localApplicationSession) (*apipb.RemoteCloudStatusResult, error)) (cloudStatusView, error) {
	socketPath := resolveV3Socket(*socket)
	daemon, _, err := daemonStatus(socketPath, *logFile, *configPath)
	if err != nil {
		return cloudStatusView{}, err
	}
	if daemon.State == "running" {
		response, err := callCloudStatus(cmd, socket, logFile, call)
		if err != nil {
			return cloudStatusView{}, err
		}
		return cloudStatusViewFromProto(response), nil
	}
	status, err := loadV3CloudStatus(v3CloudEnrollmentRecordPath(), v3CloudDisabledPath(), nil, false, "")
	if err != nil {
		return cloudStatusView{}, err
	}
	return cloudStatusViewFromCore(status), nil
}

func callCloudStatus(cmd *cobra.Command, socket, logFile *string, call func(context.Context, localApplicationSession) (*apipb.RemoteCloudStatusResult, error)) (*apipb.RemoteCloudStatusResult, error) {
	ctx := cmd.Context()
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	application, client, err := dialLocalApplicationSession(ctx, resolveV3Socket(*socket), resolveV3LogFilePath(*logFile))
	if err != nil {
		return nil, err
	}
	defer client.Close()
	response, err := call(ctx, application)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, errors.New("daemon returned no Cloud status")
	}
	return response, nil
}

func resolveCloudEdgeSelector(selection *cloudv1.DaemonEdgeSelection, selector string) (string, error) {
	selector = strings.TrimSpace(selector)
	if strings.EqualFold(selector, "auto") {
		return "", nil
	}
	matches := make([]string, 0, 1)
	for _, candidate := range selection.GetCandidates() {
		locator := candidate.GetLocator()
		if locator.GetEdgeId() == selector {
			return locator.GetEdgeId(), nil
		}
		if strings.EqualFold(locator.GetName(), selector) {
			matches = append(matches, locator.GetEdgeId())
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("Edge name %q is ambiguous; use an Edge ID", selector)
	}
	return "", fmt.Errorf("Edge %q was not found", selector)
}

func writeCloudEdgeSelection(cmd *cobra.Command, selection *cloudv1.DaemonEdgeSelection) error {
	preferred := selection.GetPreferredEdgeId()
	if preferred == "" {
		preferred = "auto"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Daemon: %s\nPreferred: %s\nSelected: %s\n", selection.GetDaemonId(), preferred, selection.GetSelectedEdgeId())
	rows := make([][]string, 0, len(selection.GetCandidates()))
	for _, candidate := range selection.GetCandidates() {
		locator, measurement := candidate.GetLocator(), candidate.GetMeasurement()
		latency, failures := "-", "-"
		if measurement != nil {
			latency = fmt.Sprintf("%d ms", measurement.GetConnectLatencyMs())
			failures = fmt.Sprintf("%.0f%%", measurement.GetConnectionFailureRate()*100)
		}
		flags := ""
		if candidate.GetCurrent() {
			flags += "current "
		}
		if candidate.GetPreferred() {
			flags += "preferred"
		}
		rows = append(rows, []string{locator.GetEdgeId(), locator.GetName(), locator.GetRegion(), latency, failures, fmt.Sprintf("%.1f", candidate.GetScore()), candidate.GetStatus(), strings.TrimSpace(flags)})
	}
	return writeCLITable(cmd.OutOrStdout(), []string{"EDGE ID", "NAME", "REGION", "LATENCY", "CONNECT FAIL", "SCORE", "STATUS", "FLAGS"}, rows)
}

type cloudStatusView struct {
	Enrolled          bool   `json:"enrolled"`
	Enabled           bool   `json:"enabled"`
	Running           bool   `json:"running"`
	Ready             bool   `json:"ready"`
	State             string `json:"state"`
	Detail            string `json:"detail,omitempty"`
	DaemonID          string `json:"daemon_id,omitempty"`
	AccountID         string `json:"account_id,omitempty"`
	EdgeID            string `json:"edge_id,omitempty"`
	EdgeName          string `json:"edge_name,omitempty"`
	EdgeRegion        string `json:"edge_region,omitempty"`
	PublicEndpoint    string `json:"public_endpoint,omitempty"`
	ServerName        string `json:"server_name,omitempty"`
	LifecycleState    string `json:"lifecycle_state,omitempty"`
	LifecycleRevision uint64 `json:"lifecycle_revision,omitempty"`
	ActiveSessions    int    `json:"active_sessions,omitempty"`
	EnrolledAt        string `json:"enrolled_at,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
	RecordPath        string `json:"record_path,omitempty"`
	DisabledPath      string `json:"disabled_path,omitempty"`
}

func cloudStatusViewFromCore(status corev2.RemoteCloudStatus) cloudStatusView {
	return cloudStatusView{
		Enrolled: status.Enrolled, Enabled: status.Enabled, Running: status.Running, Ready: status.Ready,
		State: status.State, Detail: status.Detail, DaemonID: status.DaemonID, AccountID: status.AccountID,
		EdgeID: status.EdgeID, EdgeName: status.EdgeName, EdgeRegion: status.EdgeRegion,
		PublicEndpoint: status.PublicEndpoint, ServerName: status.ServerName,
		LifecycleState: status.LifecycleState, LifecycleRevision: status.LifecycleRevision, ActiveSessions: status.ActiveSessions,
		EnrolledAt: formatCloudTime(status.EnrolledAt), UpdatedAt: formatCloudTime(status.UpdatedAt),
		RecordPath: status.RecordPath, DisabledPath: status.DisabledPath,
	}
}

func cloudStatusViewFromProto(status *apipb.RemoteCloudStatusResult) cloudStatusView {
	if status == nil {
		return cloudStatusView{}
	}
	return cloudStatusView{
		Enrolled: status.GetEnrolled(), Enabled: status.GetEnabled(), Running: status.GetRunning(), Ready: status.GetReady(),
		State: status.GetState(), Detail: status.GetDetail(), DaemonID: status.GetDaemonId(), AccountID: status.GetAccountId(),
		EdgeID: status.GetEdgeId(), EdgeName: status.GetEdgeName(), EdgeRegion: status.GetEdgeRegion(),
		PublicEndpoint: status.GetPublicEndpoint(), ServerName: status.GetServerName(),
		LifecycleState: status.GetLifecycleState(), LifecycleRevision: status.GetLifecycleRevision(), ActiveSessions: int(status.GetActiveSessions()),
		EnrolledAt: formatCloudUnixNano(status.GetEnrolledAtUnixNano()), UpdatedAt: formatCloudUnixNano(status.GetUpdatedAtUnixNano()),
		RecordPath: status.GetRecordPath(), DisabledPath: status.GetDisabledPath(),
	}
}

func writeCloudStatus(cmd *cobra.Command, status cloudStatusView) error {
	fields := []cliField{
		{Label: "State", Value: status.State},
		{Label: "Enrolled", Value: formatCloudBool(status.Enrolled)},
		{Label: "Enabled", Value: formatCloudBool(status.Enabled)},
		{Label: "Running", Value: formatCloudBool(status.Running)},
		{Label: "Ready", Value: formatCloudBool(status.Ready)},
	}
	if status.Detail != "" {
		fields = append(fields, cliField{Label: "Detail", Value: status.Detail})
	}
	if status.DaemonID != "" {
		fields = append(fields, cliField{Label: "Daemon", Value: status.DaemonID})
	}
	if status.AccountID != "" {
		fields = append(fields, cliField{Label: "Account", Value: status.AccountID})
	}
	if status.EdgeID != "" {
		edge := status.EdgeID
		if status.EdgeName != "" {
			edge = fmt.Sprintf("%s (%s)", status.EdgeName, status.EdgeID)
		}
		if status.EdgeRegion != "" {
			edge += " " + status.EdgeRegion
		}
		fields = append(fields, cliField{Label: "Edge", Value: edge})
	}
	if status.PublicEndpoint != "" {
		fields = append(fields, cliField{Label: "Endpoint", Value: status.PublicEndpoint})
	}
	if status.ServerName != "" {
		fields = append(fields, cliField{Label: "ServerName", Value: status.ServerName})
	}
	if status.LifecycleState != "" {
		fields = append(fields, cliField{Label: "Lifecycle", Value: fmt.Sprintf("%s rev=%d", status.LifecycleState, status.LifecycleRevision)})
	}
	if status.ActiveSessions > 0 {
		fields = append(fields, cliField{Label: "Sessions", Value: fmt.Sprintf("%d", status.ActiveSessions)})
	}
	if status.EnrolledAt != "" {
		fields = append(fields, cliField{Label: "EnrolledAt", Value: status.EnrolledAt})
	}
	if status.UpdatedAt != "" {
		fields = append(fields, cliField{Label: "UpdatedAt", Value: status.UpdatedAt})
	}
	if status.RecordPath != "" {
		fields = append(fields, cliField{Label: "Record", Value: status.RecordPath})
	}
	if status.DisabledPath != "" {
		fields = append(fields, cliField{Label: "DisabledMarker", Value: status.DisabledPath})
	}
	return writeCLIFields(cmd.OutOrStdout(), fields...)
}

func formatCloudBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func formatCloudUnixNano(value int64) string {
	if value == 0 {
		return ""
	}
	return time.Unix(0, value).UTC().Format(time.RFC3339)
}

func formatCloudTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func cloudEnrollCommand() *cobra.Command {
	var controllerOrigin, controllerAddress, controllerServerName string
	command := &cobra.Command{
		Use: "enroll CODE", Short: "Enroll this daemon DeviceIdentity into AnyTTY Cloud", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address, serverName, err := resolveController(controllerOrigin, controllerAddress, controllerServerName)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if _, ok := ctx.Deadline(); !ok {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
			}
			record, err := clouddaemon.EnrollLocal(ctx, address, serverName, args[0], v3RemoteIdentityDir(), v3CloudEnrollmentRecordPath())
			if err != nil {
				if failure := cloudclient.EntitlementFailure(err); failure.GetCode() == cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE {
					return fmt.Errorf("AnyTTY Cloud subscription is inactive; ask the account owner to manage it at %s/subscription", strings.TrimRight(controllerOrigin, "/"))
				}
				if status.Code(err) == codes.ResourceExhausted && strings.Contains(status.Convert(err).Message(), "cloud_daemon_limit_exhausted") {
					return fmt.Errorf("Cloud daemon limit reached; upgrade the plan or permanently delete an unused daemon at %s/devices", strings.TrimRight(controllerOrigin, "/"))
				}
				return fmt.Errorf("enroll daemon in AnyTTY Cloud: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Cloud enrollment complete: daemon=%s account=%s\n", record.DaemonID, record.AccountID)
			if record.DaemonLimit > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Registered daemon capacity: %d / %d. Manage the plan at %s/devices\n", record.DaemonCount, record.DaemonLimit, strings.TrimRight(controllerOrigin, "/"))
			}
			return nil
		},
	}
	command.Flags().StringVar(&controllerOrigin, "controller", defaultCloudControllerOrigin, "Controller HTTPS origin (built-in default; override for self-hosted controllers)")
	command.Flags().StringVar(&controllerAddress, "controller-address", "", "Controller gRPC address override")
	command.Flags().StringVar(&controllerServerName, "controller-server-name", "", "Controller TLS server name override")
	return command
}

func resolveController(origin, addressOverride, serverOverride string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.Path != "" {
		return "", "", fmt.Errorf("controller must be an HTTPS origin")
	}
	serverName := strings.TrimSpace(serverOverride)
	if serverName == "" {
		serverName = parsed.Hostname()
	}
	address := strings.TrimSpace(addressOverride)
	if address == "" {
		port := parsed.Port()
		if port == "" && parsed.Hostname() == defaultCloudControllerServerName {
			address = defaultCloudControllerAddress
		} else {
			if port == "" {
				port = "443"
			}
			address = net.JoinHostPort(parsed.Hostname(), port)
		}
	}
	return address, serverName, nil
}

func v3CloudEnrollmentRecordPath() string {
	return filepath.Join(v3RemoteIdentityDir(), "cloud_enrollment.json")
}
