// Command anytty-access is the access gateway entry: an independent process
// that exposes the local daemon over operator-chosen listeners (tcp/unix)
// while staying byte-transparent to the access wire.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/anytty/anytty/access/accessrun"
	"github.com/anytty/anytty/access/direct"
	"github.com/anytty/anytty/access/gateway"
)

// version is replaced for release builds with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// options is the merged flag/config view of one run.
type options struct {
	Socket         string
	AccessSocket   string
	ProviderSocket string
	FileRoots      []string
	TransferDir    string
	Listeners      []gateway.ListenerSpec
	Allow          []string
	PairToken      []byte
	PairTokenFrom  string
	ConfigPath     string
	LogFile        string
	Route          string
	ShowVersion    bool
}

// fileConfig mirrors the optional --config JSON document. Flags override its
// values; the file exists so operators can keep listener and token paths out
// of the process list.
type fileConfig struct {
	Socket         string   `json:"socket"`
	AccessSocket   string   `json:"access_socket"`
	ProviderSocket string   `json:"provider_socket"`
	FileRoots      []string `json:"file_roots"`
	TransferDir    string   `json:"transfer_dir"`
	Listen         []string `json:"listen"`
	Allow          []string `json:"allow"`
	PairTokenFile  string   `json:"pair_token_file"`
	Route          string   `json:"route"`
}

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseOptions(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "anytty-access: %v\n", err)
		return 2
	}
	if opts.ShowVersion {
		fmt.Fprintf(stdout, "anytty-access %s\n", version)
		return 0
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := accessrun.Run(ctx, accessrun.Options{
		Socket:         opts.Socket,
		AccessSocket:   opts.AccessSocket,
		ProviderSocket: opts.ProviderSocket,
		Listeners:      opts.Listeners,
		Allow:          opts.Allow,
		PairToken:      opts.PairToken,
		PairTokenFrom:  opts.PairTokenFrom,
		Route:          opts.Route,
		LogFile:        opts.LogFile,
		FileRoots:      opts.FileRoots,
		TransferDir:    opts.TransferDir,
	}); err != nil {
		fmt.Fprintf(stderr, "anytty-access: %v\n", err)
		return 1
	}
	return 0
}

// parseOptions merges flags and the optional JSON config file. Explicit flags
// win over file values.
func parseOptions(args []string) (options, error) {
	flags := flag.NewFlagSet("anytty-access", flag.ContinueOnError)
	var listenValues stringList
	var allowValues stringList
	var fileRootValues stringList
	var configPath string
	var pairToken string
	var pairTokenFile string
	var logFile string
	var opts options
	flags.Var(&listenValues, "listen", "listener spec, repeatable: tcp:HOST:PORT or unix:/path")
	flags.Var(&allowValues, "allow", "TCP peer allow list entry (IP or CIDR), repeatable")
	flags.Var(&fileRootValues, "file-root", "allowed file root, repeatable; empty allows any absolute path")
	flags.StringVar(&opts.Socket, "socket", "", "canonical client socket base path")
	flags.StringVar(&opts.AccessSocket, "access-socket", "", "access client protocol listener path (default: <socket>)")
	flags.StringVar(&opts.ProviderSocket, "provider-socket", "", "daemon terminal provider socket path (default: <socket>.provider)")
	flags.StringVar(&opts.TransferDir, "transfer-dir", "", "durable file-transfer resume record directory (default: state dir)")
	flags.StringVar(&configPath, "config", "", "JSON config file path")
	flags.StringVar(&pairToken, "pair-token", "", "optional pair token (prefer --pair-token-file; argv is visible in process lists)")
	flags.StringVar(&pairTokenFile, "pair-token-file", "", "optional file holding the pair token")
	flags.StringVar(&logFile, "log-file", "", "log file path (default: $ANYTTY_ACCESS_LOG_FILE or XDG state dir)")
	flags.StringVar(&opts.Route, "route", "", "Direct listen HOST:PORT; wildcard hosts enable paired-device LAN discovery")
	flags.BoolVar(&opts.ShowVersion, "version", false, "print version and exit")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() > 0 {
		return options{}, fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	opts.ConfigPath = strings.TrimSpace(configPath)
	opts.LogFile = strings.TrimSpace(logFile)

	var file fileConfig
	if opts.ConfigPath != "" {
		loaded, err := loadFileConfig(opts.ConfigPath)
		if err != nil {
			return options{}, err
		}
		file = loaded
	}
	if opts.Socket == "" {
		opts.Socket = strings.TrimSpace(file.Socket)
	}
	if opts.AccessSocket == "" {
		opts.AccessSocket = strings.TrimSpace(file.AccessSocket)
	}
	if opts.ProviderSocket == "" {
		opts.ProviderSocket = strings.TrimSpace(file.ProviderSocket)
	}
	if len(fileRootValues.values) > 0 {
		opts.FileRoots = fileRootValues.values
	} else {
		opts.FileRoots = file.FileRoots
	}
	if opts.TransferDir == "" {
		opts.TransferDir = strings.TrimSpace(file.TransferDir)
	}
	opts.Route = strings.TrimSpace(opts.Route)
	if opts.Route == "" {
		opts.Route = strings.TrimSpace(file.Route)
	}
	listenSpecs := listenValues.values
	if len(listenSpecs) == 0 {
		listenSpecs = file.Listen
	}
	for _, value := range listenSpecs {
		spec, err := gateway.ParseListenerSpec(value)
		if err != nil {
			return options{}, err
		}
		opts.Listeners = append(opts.Listeners, spec)
	}
	if len(allowValues.values) > 0 {
		opts.Allow = allowValues.values
	} else {
		opts.Allow = file.Allow
	}
	if _, err := gateway.ParseNetworks(opts.Allow); err != nil {
		return options{}, err
	}

	tokenSource := strings.TrimSpace(pairTokenFile)
	tokenValue := pairToken
	if tokenSource == "" {
		tokenSource = strings.TrimSpace(file.PairTokenFile)
	}
	switch {
	case tokenSource != "":
		token, err := accessrun.ReadTokenFile(tokenSource)
		if err != nil {
			return options{}, err
		}
		opts.PairToken = token
		opts.PairTokenFrom = tokenSource
	case tokenValue != "":
		opts.PairToken = []byte(tokenValue)
		opts.PairTokenFrom = "command line"
	}
	if !opts.ShowVersion {
		if strings.TrimSpace(opts.Socket) == "" {
			return options{}, errors.New("--socket (or config socket) is required")
		}
		if opts.Route != "" {
			if err := direct.ValidateListenAddress(opts.Route); err != nil {
				return options{}, err
			}
		}
		if len(opts.Listeners) == 0 && !opts.directConfigured() {
			return options{}, errors.New("at least one --listen or --route is required")
		}
	}
	return opts, nil
}

// directConfigured 报告本次运行是否启用 Direct listener：显式 --route 或环境变量配置。
func (opts options) directConfigured() bool {
	if strings.TrimSpace(opts.Route) != "" {
		return true
	}
	for _, key := range []string{"ANYTTY_DIRECT_LISTEN", "ANYTTY_DIRECT_SIGNALING_LISTEN", "ANYTTY_DIRECT_ICE_TCP_LISTEN"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}

func loadFileConfig(path string) (fileConfig, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config %q: %w", path, err)
	}
	var config fileConfig
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return fileConfig{}, fmt.Errorf("decode config %q: %w", path, err)
	}
	return config, nil
}

// stringList collects repeatable string flags in order.
type stringList struct {
	values []string
}

func (list *stringList) String() string { return strings.Join(list.values, ",") }

func (list *stringList) Set(value string) error {
	list.values = append(list.values, value)
	return nil
}
