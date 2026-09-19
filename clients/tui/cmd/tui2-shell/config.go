package main

import (
	"flag"
	"io"
	"time"

	"github.com/anytty/anytty/clients/tui/config"
)

// shellOptions is the parsed command line of the layout program.
type shellOptions struct {
	configPath   string
	attach       string
	printDefault bool
	version      bool
}

// parseOptions parses the shell flags. It never touches the config file.
func parseOptions(args []string) (shellOptions, error) {
	fs := flag.NewFlagSet("tui2-shell", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var opts shellOptions
	fs.StringVar(&opts.configPath, "config", "", "path to the tui2 config JSON (default $XDG_CONFIG_HOME/anytty/tui2.json)")
	fs.StringVar(&opts.attach, "attach", "", "terminal to bind at startup: <id> or <endpoint>:<id>; missing terminals fall back to the picker")
	fs.BoolVar(&opts.printDefault, "print-default-config", false, "print an example config JSON and exit")
	fs.BoolVar(&opts.version, "version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return shellOptions{}, err
	}
	return opts, nil
}

// loadConfig reads the config, degrading to the built-in defaults with a
// one-line warning instead of failing. The warning is shown as a status
// notice once the view is live.
func loadConfig(path string) (config.Config, string) {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Default(), "config error: " + err.Error()
	}
	return cfg, ""
}

// applyDefaults installs the built-in config defaults.
func (m *model) applyDefaults() {
	cfg := config.Default()
	if err := m.applyConfig(cfg); err != nil {
		panic(err) // defaults are valid by construction
	}
}

// endpointSpecs converts the config endpoints into the model's local spec
// slice (the model never imports the config package).
func endpointSpecs(endpoints []config.Endpoint) []endpointSpec {
	if len(endpoints) == 0 {
		return nil
	}
	out := make([]endpointSpec, 0, len(endpoints))
	for _, endpoint := range endpoints {
		out = append(out, endpointSpec{
			name:              endpoint.Name,
			label:             endpoint.Label,
			kind:              endpoint.KindName(),
			argv:              append([]string(nil), endpoint.Argv...),
			cwd:               endpoint.Cwd,
			env:               cloneEnv(endpoint.Env),
			socket:            endpoint.Socket,
			address:           endpoint.Address,
			connectMode:       endpoint.ConnectModeName(),
			signaling:         append([]string(nil), endpoint.SignalingAddresses...),
			iceTCP:            append([]string(nil), endpoint.ICETCPAddresses...),
			daemonDeviceID:    endpoint.DaemonDeviceID,
			daemonFingerprint: endpoint.DaemonFingerprint,
			credentialDir:     endpoint.CredentialDir,
			credentialRef:     endpoint.CredentialRef,
			cloudGateway:      endpoint.CloudGatewayAddress,
		})
	}
	return out
}

// applyConfig applies a parsed config to the model. The theme is resolved
// program-side into explicit styles (the host holds no palette); a
// keybinding collision returns an error and leaves the previous bindings
// untouched.
func (m *model) applyConfig(cfg config.Config) error {
	binds, err := newKeybindings(cfg.KeybindingOverrides())
	if err != nil {
		return err
	}
	m.theme = themeByName(cfg.Theme)
	m.icons = config.ResolveIcons(cfg.Icons)
	m.endpoints = endpointSpecs(cfg.Endpoints)
	m.binds = binds
	m.gap = cfg.GapValue()
	m.sidebar = cfg.SidebarEnabled()
	m.clockEnabled = cfg.ClockEnabled()
	m.clockLayout = cfg.ClockLayout()
	m.forwardWindow = cfg.ForwardWindow()
	m.autoAttachFirst = cfg.AutoAttachFirst()
	m.startupCwd = cfg.Startup.Cwd
	if m.now == nil {
		m.now = time.Now
	}
	return nil
}
