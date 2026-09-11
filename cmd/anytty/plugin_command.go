package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	localadapter "github.com/anytty/anytty/client/adapter/local"
	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	endpointdomain "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/plugin/host"
	"github.com/anytty/anytty/plugin/sdk"
	"github.com/anytty/anytty/plugins/agents"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func pluginRegistryPath() string {
	if path := os.Getenv("ANYTTY_PLUGIN_REGISTRY"); path != "" {
		return path
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "anytty", "plugins.yaml")
}
func newPluginCommand(runtime terminalCommandRuntime) *cobra.Command {
	command := &cobra.Command{Use: "plugin", Short: "Manage daemon and TUI workflow plugins"}
	var registryPath string
	command.PersistentFlags().StringVar(&registryPath, "registry", pluginRegistryPath(), "plugin installation registry")
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		registry, err := host.LoadRegistry(registryPath)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(registry)
	}}
	command.AddCommand(list)
	command.AddCommand(&cobra.Command{Use: "doctor", Short: "Validate installed packages without executing them", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		registry, err := host.LoadRegistry(registryPath)
		if err != nil {
			return err
		}
		var failures []error
		for _, installation := range registry.Plugins {
			manifest, err := host.ReadManifest(installation.Directory)
			if err == nil && manifest.ID != installation.ID {
				err = errors.New("registry and manifest IDs differ")
			}
			if err == nil {
				for _, component := range []host.Component{manifest.Daemon, manifest.TUI} {
					if len(component.Command) == 0 {
						continue
					}
					executable := component.Command[0]
					if strings.ContainsAny(executable, "/\\") {
						if !filepath.IsAbs(executable) {
							executable = filepath.Join(installation.Directory, executable)
						}
						info, statErr := os.Stat(executable)
						if statErr != nil {
							err = statErr
							break
						}
						if !info.Mode().IsRegular() {
							err = errors.New("plugin executable is not a regular file")
							break
						}
					} else if _, lookupErr := exec.LookPath(executable); lookupErr != nil {
						err = lookupErr
						break
					}
				}
			}
			if err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", installation.ID, err))
				fmt.Fprintf(cmd.OutOrStdout(), "FAIL %s: %v\n", installation.ID, err)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "OK %s (enabled=%t)\n", installation.ID, installation.Enabled)
			}
		}
		return errors.Join(failures...)
	}})
	var logMode string
	logs := &cobra.Command{Use: "logs ID", Short: "Read the most recent 64 KiB of a plugin component log", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		registry, err := host.LoadRegistry(registryPath)
		if err != nil {
			return err
		}
		installed := false
		for _, installation := range registry.Plugins {
			if installation.ID == args[0] {
				installed = true
			}
		}
		if !installed {
			return errors.New("plugin is not installed")
		}
		if logMode == "" || strings.ContainsAny(logMode, "/\\") || logMode == "." || logMode == ".." {
			return errors.New("invalid log component")
		}
		file, err := os.Open(filepath.Join(filepath.Dir(registryPath), "plugin-logs", args[0], logMode+".log"))
		if err != nil {
			return err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return err
		}
		if info.Size() > 64<<10 {
			if _, err = file.Seek(info.Size()-(64<<10), io.SeekStart); err != nil {
				return err
			}
		}
		_, err = io.Copy(cmd.OutOrStdout(), io.LimitReader(file, 64<<10))
		return err
	}}
	logs.Flags().StringVar(&logMode, "component", "daemon", "daemon or tui-INSTANCE_ID")
	command.AddCommand(logs)
	for _, name := range []string{"link", "install", "enable", "disable", "uninstall"} {
		command.AddCommand(&cobra.Command{Use: name + " ID_OR_DIRECTORY", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			err := host.UpdateRegistry(cmd.Context(), registryPath, func(registry *host.Registry) error {
				switch name {
				case "link":
					_, err := registry.Link(args[0])
					return err
				case "install":
					_, err := registry.Install(args[0], filepath.Join(filepath.Dir(registryPath), "plugins"))
					return err
				case "enable":
					return registry.Enable(args[0], true)
				case "disable":
					return registry.Enable(args[0], false)
				case "uninstall":
					return registry.Remove(args[0])
				}
				return nil
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Plugin %s saved. New daemon/TUI instances read this registry; running daemons are not restarted.\n", name)
			return nil
		}})
	}
	agentCommand := &cobra.Command{Use: "agents", Short: "Agent workbench integrations"}
	hook := &cobra.Command{Use: "hook codex|opencode", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		// Hook failures never block the Agent or change an approval decision.
		defer fmt.Fprintln(cmd.OutOrStdout(), "{}")
		if err := runAgentHook(cmd.Context(), cmd.InOrStdin(), args[0]); err != nil {
			// The OpenCode JS adapter consumes this status asynchronously and
			// retries; its event callback never fails or blocks the Agent loop.
			if args[0] == "opencode" {
				return err
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "AnyTTY Agent report:", err)
		}
		return nil
	}}
	agentCommand.AddCommand(hook)
	for _, operation := range []string{"install", "uninstall"} {
		var configDir string
		sub := &cobra.Command{Use: operation + " codex|opencode", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			directory := configDir
			if directory == "" {
				home, _ := os.UserHomeDir()
				if args[0] == "codex" {
					directory = filepath.Join(home, ".codex")
				} else {
					directory = filepath.Join(home, ".config", "opencode")
				}
			}
			executable, err := os.Executable()
			if err != nil {
				return err
			}
			switch args[0] {
			case "codex":
				if operation == "install" {
					err = agents.InstallCodex(directory, executable)
				} else {
					err = agents.UninstallCodex(directory)
				}
			case "opencode":
				if operation == "install" {
					err = agents.InstallOpenCode(directory, executable)
				} else {
					err = agents.UninstallOpenCode(directory)
				}
			default:
				return errors.New("expected codex or opencode")
			}
			if err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s integration: %s\n", operation, args[0], directory)
				if args[0] == "codex" && operation == "install" {
					fmt.Fprintln(cmd.OutOrStdout(), "Review and trust the new hooks in Codex before they can run.")
				}
			}
			return err
		}}
		sub.Flags().StringVar(&configDir, "config-dir", "", "explicit Agent configuration directory")
		agentCommand.AddCommand(sub)
	}
	command.AddCommand(agentCommand)
	return command
}

// This dial never starts or replaces a daemon. Hooks must be harmless when the
// owning daemon is absent, including when an older AnyTTY created the terminal.
func openPluginDaemon(ctx context.Context, socket string) (*protocoladapter.ApplicationClient, error) {
	registry := endpointdomain.DefaultRegistry()
	target, _ := registry.DefaultEndpoint()
	owner := clientruntime.NewSessionOwner()
	client, _, err := connectV3EndpointApplication(ctx, owner, target, endpointdomain.DefaultLocalRouteID, clientruntime.ConnectIntentInteractive, localadapter.Options{SocketOverride: socket, DefaultSocket: socket, ClientName: "anytty-plugin"}, nil)
	if err != nil {
		owner.Close()
		return nil, err
	}
	return client, nil
}
func pluginExecutor(application *clientruntime.ApplicationSession) sdk.Executor {
	return func(ctx context.Context, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
		result, err := application.Execute(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_Plugin{Plugin: command}})
		if err != nil {
			return nil, err
		}
		if result.GetError() != nil {
			return nil, fmt.Errorf("plugin API: %s", result.GetError().Message)
		}
		if result.GetPlugin() == nil {
			return nil, errors.New("daemon does not support plugins")
		}
		return result.GetPlugin(), nil
	}
}
func runAgentHook(ctx context.Context, input io.Reader, provider string) error {
	hookContext := agents.Context{TerminalID: os.Getenv("ANYTTY_TERMINAL_ID"), DaemonSocket: os.Getenv("ANYTTY_DAEMON_SOCKET")}
	if !hookContext.Available() {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	data, err := io.ReadAll(io.LimitReader(input, 1<<20+1))
	if err != nil {
		return err
	}
	if len(data) > 1<<20 {
		return errors.New("hook input too large")
	}
	var event *agents.Event
	switch provider {
	case "codex":
		event, err = agents.DecodeCodex(data, hookContext)
	case "opencode":
		event, err = agents.DecodeOpenCode(data, hookContext)
	default:
		return errors.New("unknown Agent provider")
	}
	if err != nil || event == nil {
		return err
	}
	if provider == "codex" {
		stateDir := os.Getenv("ANYTTY_PLUGIN_STATE_DIR")
		if stateDir == "" {
			stateDir = filepath.Join(filepath.Dir(pluginRegistryPath()), "plugin-state", agents.PluginID)
		}
		if err = agents.StampCodex(ctx, stateDir, event); err != nil {
			return err
		}
	}
	application, err := openPluginDaemon(ctx, hookContext.DaemonSocket)
	if err != nil {
		return err
	}
	defer application.Close()
	client := sdk.NewClient(pluginExecutor(application.ApplicationSession))
	reg, err := client.Register(ctx, &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{PluginId: agents.PluginID, PluginInstanceId: "hook-" + uuid.NewString()}})
	if err != nil {
		return err
	}
	defer client.Unregister(ctx)
	message := &apipb.PluginMessage{RequestId: uuid.NewString(), Destination: &apipb.PluginAddress{DaemonId: reg.Address.DaemonId, PluginId: agents.PluginID}, Body: &apipb.PluginMessage_AgentReport{AgentReport: event.Proto(reg.Address.DaemonId, time.Now())}}
	if err = client.Send(ctx, message); err != nil {
		return err
	}
	// Report is only complete after the daemon plugin persisted it and replied.
	batch, err := client.Receive(ctx, time.Second)
	if err != nil {
		return err
	}
	for _, response := range batch.Messages {
		if reply := response.GetReply(); reply != nil && reply.RequestId == message.RequestId {
			if reply.Error != nil {
				return &sdk.Error{Code: reply.Error.Code, Message: reply.Error.Message}
			}
			return nil
		}
	}
	return errors.New("Agent report acknowledgement not received")
}
