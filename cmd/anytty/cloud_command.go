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

	clouddaemon "github.com/anytty/anytty/cloud/daemon"
	"github.com/anytty/anytty/proto/apipb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func cloudCommand(socket, logFile *string) *cobra.Command {
	command := &cobra.Command{Use: "cloud", Short: "Manage AnyTTY Cloud enrollment", Args: cobra.NoArgs}
	command.AddCommand(cloudEnrollCommand())
	command.AddCommand(cloudEdgeCommand(socket, logFile))
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
		rows = append(rows, []string{locator.GetEdgeId(), locator.GetName(), locator.GetRegion(), latency, failures, candidate.GetStatus(), strings.TrimSpace(flags)})
	}
	return writeCLITable(cmd.OutOrStdout(), []string{"EDGE ID", "NAME", "REGION", "LATENCY", "CONNECT FAIL", "STATUS", "FLAGS"}, rows)
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
	command.Flags().StringVar(&controllerOrigin, "controller", "", "Controller HTTPS origin")
	command.Flags().StringVar(&controllerAddress, "controller-address", "", "Controller gRPC address override")
	command.Flags().StringVar(&controllerServerName, "controller-server-name", "", "Controller TLS server name override")
	_ = command.MarkFlagRequired("controller")
	return command
}

func resolveController(origin, addressOverride, serverOverride string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.Path != "" {
		return "", "", fmt.Errorf("--controller must be an HTTPS origin")
	}
	serverName := strings.TrimSpace(serverOverride)
	if serverName == "" {
		serverName = parsed.Hostname()
	}
	address := strings.TrimSpace(addressOverride)
	if address == "" {
		port := parsed.Port()
		if port == "" {
			port = "443"
		}
		address = net.JoinHostPort(parsed.Hostname(), port)
	}
	return address, serverName, nil
}

func v3CloudEnrollmentRecordPath() string {
	return filepath.Join(v3RemoteIdentityDir(), "cloud_enrollment.json")
}
