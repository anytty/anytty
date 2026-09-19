// Command tui2 is the v2 host entry for the local single-machine case: it
// owns the TTY (alt screen, raw mode, SIGWINCH), starts a layout program and
// runs one runtime session over it, with the PTY-backed terminal handler.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/term"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	"github.com/anytty/anytty/clients/tui/endpoint"
)

// version is the host version reported by -version.
const version = "0.3.0"

func main() {
	shellPath := flag.String("shell", "", "layout program to launch: a path or a command with arguments (e.g. \"python3 tui2/examples/python-shell/shell.py\"); default: look for tui2-shell in PATH or next to this binary")
	endpointsPath := flag.String("endpoints", "", "explicit CLI-owned endpoints.yaml path list to read (os path separator); read-only, explicit files win by endpoint name over $TUI2_ENDPOINTS and the default registry")
	logFilePath := flag.String("log-file", "", "host and shared-client log file (default: $TUI2_LOG_FILE or $XDG_STATE_HOME/anytty/tui2.log); diagnostics never go to the terminal")
	routesValue := flag.String("routes", "", "comma-separated shared route kinds to enable: local-unix,ssh-webrtc-tcp,direct-webrtc-tcp,managed-webrtc or all (default: local-unix + credential-backed ssh; also $TUI2_ROUTES)")
	devFlag := flag.Bool("dev", false, "development mode: write the decoded frame log (default $XDG_STATE_HOME/anytty/tui2-dev.log), capture layout program stderr and surface crashes/view_rejected/protocol errors as notices")
	protocolLog := flag.String("protocol-log", "", "decode host<->program frames into readable JSON lines at <file> (one timestamped record per frame per direction); implies -dev logging")
	watchPaths := stringListFlag{}
	flag.Var(&watchPaths, "watch", "file or directory to poll for changes (repeatable, comma-separated); a change restarts the layout program keeping the last good frame")
	reloadOnSave := flag.Bool("reload-on-save", false, "watch the -shell program file/directory arguments and restart the layout program on every save")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("tui2", version)
		return
	}
	// The standard logger is redirected before any host work: the shared
	// client layer (client/runtime, client/adapter/*, shared/netpath,
	// shared/connecttrace) logs through it and must never write over the alt
	// screen. Failures here still happen before the host owns the terminal.
	logPath, err := setupLogging(*logFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui2:", err)
		os.Exit(2)
	}
	routes, err := endpoint.ResolveRouteKinds(*routesValue)
	if err != nil {
		log.Printf("tui2 config error: %v", err)
		fmt.Fprintln(os.Stderr, "tui2:", err)
		closeLogging()
		os.Exit(2)
	}
	log.Printf("tui2 start version=%s pid=%d log=%s routes=%s", version, os.Getpid(), logPath, endpoint.RouteKindsLabel(routes))
	devCfg := devConfig{
		Enabled:      *devFlag,
		ProtocolLog:  *protocolLog,
		Watch:        watchPaths,
		ReloadOnSave: *reloadOnSave,
	}
	code := run(*shellPath, *endpointsPath, routes, devCfg)
	closeLogging()
	os.Exit(code)
}

func run(shellPath string, endpointsPath string, routes []clientendpoint.RouteKind, devCfg devConfig) int {
	// Validate the layout program before taking over the terminal, so a bad
	// -shell command fails with a clear error even in a pipe and with no
	// screen damage.
	argv, err := resolveShell(shellPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui2:", err)
		return 1
	}
	dev, err := newDevMode(devCfg, argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui2:", err)
		return 1
	}
	if dev != nil {
		defer dev.close()
	}
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		fmt.Fprintln(os.Stderr, "tui2: stdin and stdout must be a terminal")
		return 1
	}
	state, err := term.MakeRaw(os.Stdin.Fd())
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui2: raw mode:", err)
		return 1
	}
	defer func() { _ = term.Restore(os.Stdin.Fd(), state) }()

	cols, rows, err := term.GetSize(os.Stdin.Fd())
	if err != nil || cols <= 0 || rows <= 0 {
		cols, rows = 80, 24
	}
	// The -endpoints flag wins over TUI2_ENDPOINTS; both are read-only and
	// explicit files are merged before the default client/endpoint registry.
	explicit := strings.TrimSpace(endpointsPath)
	if explicit == "" {
		explicit = strings.TrimSpace(os.Getenv(endpoint.RegistryEnvVar))
	}
	registryPaths := endpoint.RegistryPaths(explicit)
	host := NewHost(Options{Shell: argv, In: os.Stdin, Out: os.Stdout, Cols: cols, Rows: rows, LoadSharedRegistry: true, RegistryPaths: registryPaths, Routes: routes, Dev: dev})

	stopResize := watchResize(host, os.Stdin)
	defer stopResize()

	if err := host.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui2:", err)
		return 1
	}
	return 0
}

// stringListFlag collects repeatable, comma-separated flag values so both
// "-watch a -watch b" and "-watch a,b" work.
type stringListFlag []string

func (f *stringListFlag) String() string { return strings.Join(*f, ",") }

func (f *stringListFlag) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			*f = append(*f, trimmed)
		}
	}
	return nil
}

// resolveShell finds the layout program. The explicit -shell argument is
// either a literal executable path (backward compatible) or a command plus
// arguments ("python3 tui2/examples/python-shell/shell.py"), so layout
// programs in any language can be launched. Without -shell the host looks
// for tui2-shell in PATH, then next to the host binary.
func resolveShell(shellArg string) ([]string, error) {
	if shellArg == "" {
		return defaultShell()
	}
	// A literal path wins over command splitting, so paths containing spaces
	// keep working exactly as before.
	if _, err := os.Stat(shellArg); err == nil {
		return []string{shellArg}, nil
	}
	argv, err := splitCommand(shellArg)
	if err != nil {
		return nil, fmt.Errorf("shell %q: %w", shellArg, err)
	}
	if len(argv) == 0 {
		return nil, errors.New("shell command is empty")
	}
	if err := checkShellCommand(argv); err != nil {
		return nil, err
	}
	return argv, nil
}

func defaultShell() ([]string, error) {
	if found, err := exec.LookPath("tui2-shell"); err == nil {
		return []string{found}, nil
	}
	if exe, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(exe), "tui2-shell")
		if _, err := os.Stat(sibling); err == nil {
			return []string{sibling}, nil
		}
	}
	return nil, errors.New("tui2-shell not found; build tui2/cmd/tui2-shell or pass -shell <path|command>")
}

// checkShellCommand validates argv[0] before the host takes over the
// terminal: an explicit path must exist, a bare command must be on PATH.
// The error names the exact command so the user can fix the flag.
func checkShellCommand(argv []string) error {
	name := argv[0]
	if strings.ContainsRune(name, os.PathSeparator) {
		if _, err := os.Stat(name); err != nil {
			return fmt.Errorf("shell %q: %w", name, err)
		}
		return nil
	}
	if _, err := exec.LookPath(name); err != nil {
		if len(argv) == 1 {
			return fmt.Errorf("shell %q: %w", name, err)
		}
		return fmt.Errorf("shell command %q not found in PATH: %w", name, err)
	}
	return nil
}

// splitCommand splits a -shell argument into argv, honoring single quotes,
// double quotes and backslash escapes, so both a plain path and
// "python3 path/to/shell.py --flag" work. It never invokes a shell.
func splitCommand(s string) ([]string, error) {
	var argv []string
	var current strings.Builder
	inWord := false
	quote := rune(0)
	escaped := false
	for _, r := range s {
		switch {
		case escaped:
			current.WriteRune(r)
			inWord = true
			escaped = false
		case r == '\\' && quote != '\'':
			escaped = true
			inWord = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == ' ' || r == '\t' || r == '\n':
			if inWord {
				argv = append(argv, current.String())
				current.Reset()
				inWord = false
			}
		default:
			current.WriteRune(r)
			inWord = true
		}
	}
	if escaped {
		return nil, errors.New("trailing backslash")
	}
	if quote != 0 {
		return nil, errors.New("unbalanced quote")
	}
	if inWord {
		argv = append(argv, current.String())
	}
	return argv, nil
}
