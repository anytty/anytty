package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/anytty/anytty/plugin/host"
	"github.com/anytty/anytty/plugin/sdk"
	clientruntimeadapter "github.com/anytty/anytty/tui/adapter/clientruntime"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

// Runs alongside the daemon being started by this command. It never replaces or
// signals an existing daemon, and all API calls use its explicit socket.
func runDaemonPlugins(ctx context.Context, socket string, logger *slog.Logger) {
	// The registry is deliberately watched by the daemon. Mutations made by the
	// CLI therefore affect only the corresponding child process; the daemon and
	// all unrelated plugins remain alive. A short poll also works across filesystems
	// where fsnotify cannot reliably observe atomic rename updates.
	type running struct {
		cancel    context.CancelFunc
		directory string
	}
	children := map[string]running{}
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	reconcile := func() {
		registry, err := host.LoadRegistry(pluginRegistryPath())
		if err != nil {
			logger.Error("load daemon plugins", "error", err)
			return
		}
		desired := map[string]host.Installation{}
		for _, in := range registry.Plugins {
			if in.Enabled {
				desired[in.ID] = in
			}
		}
		for id, child := range children {
			if _, ok := desired[id]; !ok {
				child.cancel()
				delete(children, id)
			}
		}
		for id, in := range desired {
			if child, ok := children[id]; ok {
				if child.directory != in.Directory {
					child.cancel()
					delete(children, id)
				} else {
					continue
				}
			}
			if _, ok := children[id]; ok {
				continue
			}
			manifest, err := host.ReadManifest(in.Directory)
			if err != nil || len(manifest.Daemon.Command) == 0 {
				continue
			}
			childCtx, cancel := context.WithCancel(ctx)
			children[id] = running{cancel: cancel, directory: in.Directory}
			go runOneDaemonPlugin(childCtx, socket, logger, in)
		}
	}
	reconcile()
	for {
		select {
		case <-ctx.Done():
			for _, c := range children {
				c.cancel()
			}
			return
		case <-ticker.C:
			reconcile()
		}
	}
}

func runOneDaemonPlugin(ctx context.Context, socket string, logger *slog.Logger, installation host.Installation) {
	for ctx.Err() == nil {
		dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		application, err := openPluginDaemon(dialCtx, socket)
		cancel()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}
		executable, _ := os.Executable()
		stateDir := filepath.Join(filepath.Dir(pluginRegistryPath()), "plugin-state", installation.ID, "daemon")
		logFile, err := openPluginLog(installation.ID, "daemon")
		if err != nil {
			application.Close()
			return
		}
		err = host.Run(ctx, host.ProcessOptions{Installation: installation, Mode: "daemon", StateDir: stateDir, DaemonSocket: socket, Executable: executable, Executor: pluginExecutor(application.ApplicationSession), Log: logFile})
		logFile.Close()
		application.Close()
		if err != nil && ctx.Err() == nil {
			logger.Error("daemon plugin exited", "plugin", installation.ID, "error", err)
		}
		return
	}
}

func openPluginLog(id, mode string) (*os.File, error) {
	directory := filepath.Join(filepath.Dir(pluginRegistryPath()), "plugin-logs", id)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(directory, mode+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}
func makeTUIPluginService(router *clientruntimeadapter.EndpointApplicationRouter, instanceID string, endpoints []state.EndpointID, logger *slog.Logger) port.PluginService {
	registry, err := host.LoadRegistry(pluginRegistryPath())
	if err != nil {
		if logger != nil {
			logger.Error("load TUI plugins", "error", err)
		}
		return nil
	}
	launches := []clientruntimeadapter.PluginLaunch{}
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i] < endpoints[j] })
	for _, installation := range registry.Plugins {
		if !installation.Enabled {
			continue
		}
		manifest, err := host.ReadManifest(installation.Directory)
		if err != nil || len(manifest.TUI.Command) == 0 {
			continue
		}
		launches = append(launches, clientruntimeadapter.PluginLaunch{ID: installation.ID, RunWithExit: func(ctx context.Context, executor sdk.RoutedExecutor, resolvedEndpoints []state.EndpointID, notifyExit func(error)) error {
			executable, _ := os.Executable()
			logFile, err := openPluginLog(installation.ID, "tui-"+instanceID)
			if err != nil {
				return err
			}
			defer logFile.Close()
			endpointJSON, _ := json.Marshal(resolvedEndpoints)
			stateDir := filepath.Join(filepath.Dir(pluginRegistryPath()), "plugin-state", installation.ID, "tui")
			return host.Run(ctx, host.ProcessOptions{Installation: installation, Mode: "tui", TUIInstanceID: instanceID, StateDir: stateDir, Executable: executable, RoutedExecutor: executor, Log: logFile, ExtraEnv: []string{"ANYTTY_PLUGIN_ENDPOINTS=" + string(endpointJSON)}, OnExit: func(err error) {
				notifyExit(err)
				if err != nil && ctx.Err() == nil && logger != nil {
					logger.Warn("TUI plugin exited", "plugin", installation.ID, "error", fmt.Sprint(err))
				}
			}})
		}})
	}
	if len(launches) == 0 {
		return nil
	}
	return clientruntimeadapter.NewPluginService(router, clientruntimeadapter.PluginOptions{TUIInstanceID: instanceID, Endpoints: endpoints, Plugins: launches})
}
