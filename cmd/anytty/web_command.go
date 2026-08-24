package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/localweb"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/spf13/cobra"
)

type localWebApplicationSession interface {
	RemoteLocalEnable(context.Context, *apipb.RemoteLocalEnableCommand) (*apipb.RemoteLocalStatusResult, error)
	RemoteLocalStatus(context.Context, *apipb.RemoteLocalStatusCommand) (*apipb.RemoteLocalStatusResult, error)
	RemoteLocalDisable(context.Context, *apipb.RemoteLocalDisableCommand) (*apipb.RemoteLocalStatusResult, error)
}

type localWebStatusView struct {
	Enabled   bool   `json:"enabled"`
	URL       string `json:"url,omitempty"`
	Address   string `json:"address,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

var openLocalWebBrowser = openBrowser

func newWebCommand(socket, logFile, configPath *string) *cobra.Command {
	var address string
	var noOpen bool
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "web",
		Short: "Open the local AnyTTY browser interface",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			response, err := callLocalWeb(cmd, socket, logFile, configPath, func(ctx context.Context, application localWebApplicationSession) (*apipb.RemoteLocalStatusResult, error) {
				return application.RemoteLocalEnable(ctx, &apipb.RemoteLocalEnableCommand{LocalWebAddress: strings.TrimSpace(address)})
			})
			if err != nil {
				return err
			}
			status := localWebStatusFromProto(response)
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "AnyTTY Web: %s\n", status.URL)
			if noOpen {
				return nil
			}
			if err := openLocalWebBrowser(status.URL); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Could not open the browser automatically: %v\n", err)
			}
			return nil
		},
	}
	command.Flags().StringVar(&address, "listen", localweb.DefaultAddress, "IPv4 loopback listen address")
	command.Flags().BoolVar(&noOpen, "no-open", false, "print the URL without opening a browser")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	command.AddCommand(newWebStatusCommand(socket, logFile, configPath), newWebStopCommand(socket, logFile, configPath))
	return command
}

func newWebStatusCommand(socket, logFile, configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "status", Short: "Show the local Web interface state", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		status, err := readLocalWebStatus(cmd, socket, logFile, configPath)
		if err != nil {
			return err
		}
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		}
		if !status.Enabled {
			fmt.Fprintln(cmd.OutOrStdout(), "AnyTTY Web is stopped.")
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "AnyTTY Web is running at %s\n", status.URL)
		return nil
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newWebStopCommand(socket, logFile, configPath *string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "stop", Aliases: []string{"disable", "off"}, Short: "Stop the local Web interface", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		status, err := readLocalWebStatus(cmd, socket, logFile, configPath)
		if err != nil {
			return err
		}
		if status.Enabled {
			response, callErr := callLocalWeb(cmd, socket, logFile, configPath, func(ctx context.Context, application localWebApplicationSession) (*apipb.RemoteLocalStatusResult, error) {
				return application.RemoteLocalDisable(ctx, &apipb.RemoteLocalDisableCommand{})
			})
			if callErr != nil {
				return callErr
			}
			status = localWebStatusFromProto(response)
		}
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "AnyTTY Web stopped.")
		return nil
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func readLocalWebStatus(cmd *cobra.Command, socket, logFile, configPath *string) (localWebStatusView, error) {
	daemon, _, err := daemonStatus(resolveV3Socket(*socket), *logFile, *configPath)
	if err != nil {
		return localWebStatusView{}, err
	}
	if daemon.State != "running" {
		return localWebStatusView{}, nil
	}
	response, err := callLocalWeb(cmd, socket, logFile, configPath, func(ctx context.Context, application localWebApplicationSession) (*apipb.RemoteLocalStatusResult, error) {
		return application.RemoteLocalStatus(ctx, &apipb.RemoteLocalStatusCommand{})
	})
	if err != nil {
		return localWebStatusView{}, err
	}
	return localWebStatusFromProto(response), nil
}

func callLocalWeb(cmd *cobra.Command, socket, logFile, configPath *string, call func(context.Context, localWebApplicationSession) (*apipb.RemoteLocalStatusResult, error)) (*apipb.RemoteLocalStatusResult, error) {
	ctx := cmd.Context()
	client, err := dialOrStartV3ClientWithConfig(resolveV3Socket(*socket), resolveV3LogFilePath(*logFile), *configPath, nil)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	application, err := newLocalApplicationSession(client)
	if err != nil {
		return nil, err
	}
	return call(ctx, application)
}

func localWebStatusFromProto(status *apipb.RemoteLocalStatusResult) localWebStatusView {
	if status == nil {
		return localWebStatusView{}
	}
	view := localWebStatusView{Enabled: status.GetEnabled(), URL: status.GetHttpUrl(), Address: status.GetLocalWebAddress()}
	if value := status.GetUpdatedAtUnixNano(); value > 0 {
		view.UpdatedAt = time.Unix(0, value).UTC().Format(time.RFC3339Nano)
	}
	return view
}

func openBrowser(url string) error {
	if strings.TrimSpace(url) == "" {
		return errors.New("local web URL is empty")
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	return command.Start()
}

var _ localWebApplicationSession = (*clientruntime.ApplicationSession)(nil)
