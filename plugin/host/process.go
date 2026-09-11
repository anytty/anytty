package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/plugin/sdk"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

type ProcessOptions struct {
	RoutedExecutor sdk.RoutedExecutor
	Installation   Installation
	Mode           string
	TUIInstanceID  string
	StateDir       string
	DaemonSocket   string
	Executable     string
	ExtraEnv       []string
	Executor       sdk.Executor
	Log            io.Writer
	OnExit         func(error)
}

// Run starts only this plugin's own process. Cancellation never signals a daemon
// or any process discovered by name; it terminates the child owned by exec.Cmd.
func Run(ctx context.Context, options ProcessOptions) error {
	manifest, err := ReadManifest(options.Installation.Directory)
	if err != nil {
		return err
	}
	if manifest.ID != options.Installation.ID {
		return errors.New("registry and plugin manifest identities differ")
	}
	component := manifest.Daemon
	if options.Mode == "tui" {
		component = manifest.TUI
	} else if options.Mode != "daemon" {
		return errors.New("invalid plugin component mode")
	}
	if len(component.Command) == 0 {
		return nil
	}
	if options.Executor == nil && options.RoutedExecutor == nil {
		return errors.New("daemon executor required")
	}
	if options.StateDir != "" {
		if err = os.MkdirAll(options.StateDir, 0700); err != nil {
			return err
		}
	}
	for attempt := 0; ; attempt++ {
		err = runProcess(ctx, manifest, component, options)
		if options.OnExit != nil {
			options.OnExit(err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err == nil || component.Restart != "on-failure" || attempt >= 3 {
			return err
		}
		timer := time.NewTimer(time.Duration(1<<attempt) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
func runProcess(ctx context.Context, manifest Manifest, component Component, options ProcessOptions) error {
	executable := component.Command[0]
	if strings.ContainsAny(executable, "/\\") && !filepath.IsAbs(executable) {
		executable = filepath.Join(options.Installation.Directory, executable)
	}
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(childCtx, executable, component.Command[1:]...)
	cmd.Dir = options.Installation.Directory
	cmd.Env = append(os.Environ(), options.ExtraEnv...)
	cmd.Env = append(cmd.Env, "ANYTTY_PLUGIN_ID="+manifest.ID, "ANYTTY_PLUGIN_MODE="+options.Mode, "ANYTTY_TUI_INSTANCE_ID="+options.TUIInstanceID, "ANYTTY_PLUGIN_STATE_DIR="+options.StateDir, "ANYTTY_DAEMON_SOCKET="+options.DaemonSocket, "ANYTTY_BIN_PATH="+options.Executable)
	cmd.Stderr = options.Log
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	bridgeDone := make(chan error, 1)
	var leasesMu sync.Mutex
	ownedLeases := map[string]string{}
	executeRaw := func(ctx context.Context, endpointID string, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
		if options.RoutedExecutor != nil {
			return options.RoutedExecutor(ctx, endpointID, command)
		}
		return options.Executor(ctx, command)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		leasesMu.Lock()
		defer leasesMu.Unlock()
		for key, endpoint := range ownedLeases {
			_, _ = executeRaw(cleanup, endpoint, &apipb.PluginCommand{Command: &apipb.PluginCommand_Unregister{Unregister: &apipb.PluginUnregisterRequest{SourceLease: []byte(key)}}})
		}
	}()
	execute := func(ctx context.Context, endpointID string, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
		if command == nil {
			return nil, errors.New("plugin command required")
		}
		if err := authorizeCommand(manifest, options.Mode, command); err != nil {
			return nil, err
		}
		if register := command.GetRegister(); register != nil {
			if register.Address == nil || register.Address.PluginId != manifest.ID || register.DaemonService != (options.Mode == "daemon") || register.Address.TuiInstanceId != options.TUIInstanceID || register.Address.PluginInstanceId == "host" {
				return nil, errors.New("plugin registration does not match launched component")
			}
		} else {
			var lease []byte
			switch {
			case command.GetSend() != nil:
				lease = command.GetSend().SourceLease
			case command.GetReceive() != nil:
				lease = command.GetReceive().SourceLease
			case command.GetState() != nil:
				lease = command.GetState().SourceLease
			case command.GetUnregister() != nil:
				lease = command.GetUnregister().SourceLease
			}
			leasesMu.Lock()
			owner, ok := ownedLeases[string(lease)]
			leasesMu.Unlock()
			if !ok || owner != endpointID {
				return nil, errors.New("plugin lease does not belong to this component and endpoint")
			}
		}
		result, err := executeRaw(ctx, endpointID, proto.Clone(command).(*apipb.PluginCommand))
		if err == nil && result.GetRegistration() != nil {
			leasesMu.Lock()
			ownedLeases[string(result.GetRegistration().SourceLease)] = endpointID
			leasesMu.Unlock()
		}
		return result, err
	}
	go func() { bridgeDone <- sdk.ServeStdioRouted(childCtx, stdout, stdin, execute); cancel() }()
	err = cmd.Wait()
	cancel()
	stdin.Close()
	stdout.Close()
	bridgeErr := <-bridgeDone
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("plugin %s %s: %w", manifest.ID, options.Mode, err)
	}
	if bridgeErr != nil && !errors.Is(bridgeErr, io.EOF) && !errors.Is(bridgeErr, os.ErrClosed) {
		return bridgeErr
	}
	return nil
}
