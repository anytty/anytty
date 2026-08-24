package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/localweb"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type localWebApplicationSession interface {
	RemoteLocalEnable(context.Context, *apipb.RemoteLocalEnableCommand) (*apipb.RemoteLocalStatusResult, error)
	RemoteLocalStatus(context.Context, *apipb.RemoteLocalStatusCommand) (*apipb.RemoteLocalStatusResult, error)
	RemoteLocalDisable(context.Context, *apipb.RemoteLocalDisableCommand) (*apipb.RemoteLocalStatusResult, error)
}

type localWebStatusView struct {
	Enabled           bool   `json:"enabled"`
	URL               string `json:"url,omitempty"`
	Address           string `json:"address,omitempty"`
	PasswordProtected bool   `json:"password_protected"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

var openLocalWebBrowser = openBrowser

func newWebCommand(socket, logFile, configPath *string) *cobra.Command {
	var address string
	var noOpen bool
	var jsonOutput bool
	var passwordProtected bool
	command := &cobra.Command{
		Use:   "web",
		Short: "Open the local AnyTTY browser interface",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var password []byte
			if passwordProtected {
				var err error
				password, err = promptLocalWebPassword(cmd)
				if err != nil {
					return err
				}
				defer clear(password)
			}
			response, err := callLocalWeb(cmd, socket, logFile, configPath, func(ctx context.Context, application localWebApplicationSession) (*apipb.RemoteLocalStatusResult, error) {
				return application.RemoteLocalEnable(ctx, &apipb.RemoteLocalEnableCommand{LocalWebAddress: strings.TrimSpace(address), LocalWebPassword: password})
			})
			if err != nil {
				return err
			}
			status := localWebStatusFromProto(response)
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "AnyTTY Web: %s\n", status.URL)
			fmt.Fprintf(cmd.OutOrStdout(), "Password protection: %s\n", passwordProtectionState(status.PasswordProtected))
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
	command.Flags().BoolVar(&passwordProtected, "password", false, "prompt for a Web access password")
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
		fmt.Fprintf(cmd.OutOrStdout(), "Password protection: %s\n", passwordProtectionState(status.PasswordProtected))
		return nil
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func passwordProtectionState(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
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
	view := localWebStatusView{Enabled: status.GetEnabled(), URL: status.GetHttpUrl(), Address: status.GetLocalWebAddress(), PasswordProtected: status.GetPasswordProtected()}
	if value := status.GetUpdatedAtUnixNano(); value > 0 {
		view.UpdatedAt = time.Unix(0, value).UTC().Format(time.RFC3339Nano)
	}
	return view
}

func promptLocalWebPassword(cmd *cobra.Command) ([]byte, error) {
	input, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(input.Fd())) {
		return nil, fmt.Errorf("--password requires interactive terminal input")
	}
	fmt.Fprint(cmd.ErrOrStderr(), "Web access password: ")
	password, err := term.ReadPassword(int(input.Fd()))
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return nil, fmt.Errorf("read Web access password: %w", err)
	}
	if err := localweb.ValidatePassword(password); err != nil {
		clear(password)
		return nil, err
	}
	fmt.Fprint(cmd.ErrOrStderr(), "Confirm Web access password: ")
	confirmation, err := term.ReadPassword(int(input.Fd()))
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		clear(password)
		return nil, fmt.Errorf("confirm Web access password: %w", err)
	}
	defer clear(confirmation)
	if !bytes.Equal(password, confirmation) {
		clear(password)
		return nil, fmt.Errorf("Web access passwords do not match")
	}
	return password, nil
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
