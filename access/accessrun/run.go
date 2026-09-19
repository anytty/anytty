// Package accessrun 是 anytty-access / `anytty access run` 的可复用组合入口。
//
// 它拥有 access owner 的运行时装配：identity/AccessStore、control RPC、pairing、
// Cloud、relay、Direct 与 access protocol server（终端路由 + 文件 + 转发）。
// 二进制与 CLI 生命周期都调用同一 Run，避免两份组合真值。
package accessrun

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/anytty/anytty/access/direct"
	"github.com/anytty/anytty/access/files"
	"github.com/anytty/anytty/access/gateway"
	daemonprovider "github.com/anytty/anytty/access/provider/daemon"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	accessruntime "github.com/anytty/anytty/access/runtime"
	accessserver "github.com/anytty/anytty/access/server"
	"github.com/anytty/anytty/access/sessions"
	remotev2daemon "github.com/anytty/anytty/daemon/remote"
	"github.com/anytty/anytty/shared/userdirs"
)

// Options 是一次 access 运行的全部输入。
type Options struct {
	// Socket 是 canonical 客户端 socket 基址（也是 pairing/control/direct 记录基址）。
	Socket string
	// AccessSocket 是 access protocol server 的绑定路径；空时取默认。
	AccessSocket string
	// ProviderSocket 是 daemon terminal provider socket；空时取默认。
	ProviderSocket string
	// Listeners 是字节透明 relay 监听。
	Listeners []gateway.ListenerSpec
	// Allow 是 relay TCP peer allow 列表。
	Allow []string
	// PairToken 是可选 relay pairing token。
	PairToken []byte
	// PairTokenFrom 是 token 来源（诊断）。
	PairTokenFrom string
	// Route 是 Direct signaling/ICE-TCP 监听地址。
	Route string
	// LogFile 是 access 日志路径；空时用默认。
	LogFile string
	// FileRoots 是 access 文件服务允许的根目录；空表示任意绝对路径（legacy）。
	FileRoots []string
	// TransferDir 是断点续传记录目录；空时用默认。
	TransferDir string
	// Logger 可选；nil 时按 LogFile 打开文件日志。
	Logger *slog.Logger
}

// DefaultAccessSocket 返回 access protocol server 的默认绑定路径。
// Phase 3 flip 后 access 就是 canonical 客户端入口。
func DefaultAccessSocket(socket string) string {
	return strings.TrimSpace(socket)
}

// DefaultProviderSocket 返回 daemon terminal provider socket 的默认路径。
func DefaultProviderSocket(socket string) string {
	return ProviderSocketPath(socket)
}

// ProviderSocketPath 返回 canonical 客户端 socket 对应的 daemon provider socket 路径。
func ProviderSocketPath(socket string) string {
	return strings.TrimSpace(socket) + ".provider"
}

// AccessSocketPath 返回 access 客户端监听路径（Phase 3 后与 canonical 相同）。
func DefaultLogFile() string {
	return filepath.Join(userdirs.StateHome(), "anytty", "anytty-access.log")
}

// DirectConfigured 报告本次运行是否启用 Direct listener：显式 --route 或环境变量配置。
func (opts Options) DirectConfigured() bool {
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

// Resolve 返回补全默认值后的选项。
func (opts Options) Resolve() Options {
	if strings.TrimSpace(opts.AccessSocket) == "" {
		opts.AccessSocket = DefaultAccessSocket(opts.Socket)
	}
	if strings.TrimSpace(opts.ProviderSocket) == "" {
		opts.ProviderSocket = DefaultProviderSocket(opts.Socket)
	}
	return opts
}

// Run 装配并运行 access，直到 ctx 取消或进程退出。
func Run(ctx context.Context, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	opts = opts.Resolve()
	if strings.TrimSpace(opts.Socket) == "" {
		return errors.New("access: --socket is required")
	}
	logger := opts.Logger
	closeLog := func() {}
	logPath := ""
	if logger == nil {
		var err error
		logger, closeLog, logPath, err = OpenLogger(opts.LogFile)
		if err != nil {
			return err
		}
	} else {
		logPath = strings.TrimSpace(opts.LogFile)
	}
	defer closeLog()

	accessRuntime, err := accessruntime.Load(opts.Socket)
	if err != nil {
		logger.Error("access identity/store load failed", "error", err)
		return fmt.Errorf("access identity/store load: %w", err)
	}
	defer func() { _ = accessRuntime.Close() }()

	sessionRegistry := sessions.NewRegistry()
	defer sessionRegistry.CloseAll()

	// Phase 4：鉴权/配对/Cloud 命令由 access/server 直答；不再有 daemon→access
	// 的反向 control RPC，也没有 daemonpipe 字节管道。
	accessService := accessruntime.Service{
		DeviceIdentity: accessRuntime.Identity, Store: accessRuntime.Store,
		DefaultLabel: func() string {
			return accessruntime.DefaultPairingLabelFromEnrollment(accessruntime.EnrollmentRecordPath())
		},
		Sessions: sessionRegistry,
	}
	cloudControl := &accessruntime.CloudControl{}
	accessServer, err := accessserver.New(accessserver.Config{
		Socket:  opts.AccessSocket,
		Tracker: sessionRegistry,
		Logger:  logger,
		Files: files.Config{
			Resolver:    files.ResolverConfig{Roots: opts.FileRoots},
			TransferDir: opts.TransferDir,
		},
		Auth: &accessserver.AuthServices{Access: accessService, Remote: cloudControl},
		Provider: func(dialCtx context.Context) (terminalprovider.Provider, error) {
			return daemonprovider.DialTerminal(dialCtx, opts.ProviderSocket)
		},
	})
	if err != nil {
		logger.Error("access protocol server failed", "error", err)
		return fmt.Errorf("access protocol server: %w", err)
	}
	defer func() { _ = accessServer.Close() }()
	go func() {
		if serveErr := accessServer.Serve(ctx); serveErr != nil && ctx.Err() == nil {
			logger.Error("access protocol server stopped", "error", serveErr)
		}
	}()
	cloudControl.ConfigureLocalWeb(accessServer)

	closePairing, err := accessRuntime.StartPairingListener(ctx, logger)
	if err != nil {
		logger.Error("access pairing listener failed", "error", err)
		return fmt.Errorf("access pairing listener: %w", err)
	}
	defer closePairing()

	closeCloud, err := accessruntime.StartCloud(ctx, accessServer, accessRuntime, logger, cloudControl)
	if err != nil {
		logger.Error("access Cloud runtime failed", "error", err)
		return fmt.Errorf("access Cloud runtime: %w", err)
	}
	defer closeCloud()

	var relay *gateway.Gateway
	if len(opts.Listeners) > 0 {
		relay, err = gateway.New(gateway.Config{
			Provider:  daemonprovider.New(opts.AccessSocket),
			Listeners: opts.Listeners,
			Allow:     MustParseNetworks(opts.Allow, logger),
			PairToken: opts.PairToken,
			Logger:    logger,
		})
		if err != nil {
			logger.Error("access startup failed", "error", err)
			return fmt.Errorf("access relay startup: %w", err)
		}
	}
	var directServer *direct.Server
	if opts.DirectConfigured() {
		acceptor := remotev2daemon.SessionAcceptor{
			Core: accessServer, Identity: accessRuntime.Identity, AccessStore: accessRuntime.Store,
		}
		directServer, err = direct.Start(ctx, direct.Options{
			Identity:        accessRuntime.Identity,
			Handler:         acceptor,
			SignalingListen: opts.Route,
			ICETCPListen:    opts.Route,
			RecordPath:      direct.RecordPath(opts.Socket),
			Logger:          logger,
		})
		if err != nil {
			logger.Error("access Direct startup failed", "error", err)
			return fmt.Errorf("access Direct startup: %w", err)
		}
		defer directServer.Close()
		signaling, ice := directServer.Addresses()
		logger.Info("access Direct enabled", "route", opts.Route, "signaling", signaling, "ice_tcp", ice)
	}
	logger.Info("access starting",
		"daemon_socket", opts.ProviderSocket,
		"access_socket", opts.AccessSocket,
		"file_roots", len(opts.FileRoots),
		"listeners", len(opts.Listeners),
		"direct", opts.DirectConfigured(),
		"allow", len(opts.Allow),
		"pair_token", opts.PairTokenFrom,
		"log_file", logPath,
	)
	if relay != nil {
		if err := relay.Serve(ctx); err != nil {
			logger.Error("access stopped with error", "error", err)
			return err
		}
		return nil
	}
	<-ctx.Done()
	return nil
}

// OpenLogger 打开 owner-only access 日志文件；空路径用默认。
func OpenLogger(explicit string) (*slog.Logger, func(), string, error) {
	path := strings.TrimSpace(explicit)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("ANYTTY_ACCESS_LOG_FILE"))
	}
	if path == "" {
		path = DefaultLogFile()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, nil, path, fmt.Errorf("create log directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, path, fmt.Errorf("open log file %q: %w", path, err)
	}
	logger := slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo})).With("pid", os.Getpid())
	return logger, func() { _ = file.Close() }, path, nil
}

// MustParseNetworks 解析已验证的 allow 列表；解析失败只记日志，不 panic。
func MustParseNetworks(values []string, logger *slog.Logger) []*net.IPNet {
	networks, err := gateway.ParseNetworks(values)
	if err != nil && logger != nil {
		logger.Error("access allow list parse failed", "error", err)
	}
	return networks
}

// ReadTokenFile 读取 owner-only pair token 文件。
func ReadTokenFile(path string) ([]byte, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pair token file %q: %w", path, err)
	}
	token := []byte(strings.TrimSpace(string(payload)))
	if len(token) == 0 {
		return nil, fmt.Errorf("pair token file %q is empty", path)
	}
	return token, nil
}
