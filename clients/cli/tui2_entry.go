package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	endpointdomain "github.com/anytty/anytty/access/engine/endpoint"
)

// tui2EntryConfig 是 `anytty` / `anytty attach` 启动新 TUI（tui2 宿主 +
// tui2-shell 布局程序）的输入。
type tui2EntryConfig struct {
	EndpointID string
	TerminalID string
	// SocketOverride 是显式 --socket（已解析）；只有它才生成临时 registry
	// override，避免覆盖用户 registry 里的 default endpoint 其它 route。
	SocketOverride string
	// LocalSocket 是用于 auto-start 的 canonical socket；空时用默认路径。
	LocalSocket string
	LogFile     string
	ConfigPath  string
}

var (
	runTUI2 = runTUI2Runtime
	// tui2Executable 允许测试替换宿主定位逻辑。
	tui2Executable = locateTUI2Executable
	tui2ShellPath  = locateTUI2ShellExecutable
)

// runTUI2Runtime 拉起本地栈（必要时），然后以前台子进程运行 tui2。
// 新 TUI 读取 CLI/TUI 共享 endpoint registry；显式 --socket 通过临时
// registry override 注入，保持旧的直连语义。
func runTUI2Runtime(ctx context.Context, cfg tui2EntryConfig) error {
	_ = ctx
	logPath := resolveV3LogFilePath(cfg.LogFile)
	localSocket := strings.TrimSpace(cfg.LocalSocket)
	if localSocket == "" {
		localSocket = resolveV3Socket("")
	}
	if err := ensureTUI2LocalStack(localSocket, logPath, cfg.ConfigPath); err != nil {
		return err
	}
	hostPath, err := tui2Executable()
	if err != nil {
		return err
	}
	args := []string{"-log-file", logPath}
	if shell := tui2ShellPath(hostPath); shell != "" {
		command := quoteTUI2Arg(shell)
		if path := strings.TrimSpace(cfg.ConfigPath); path != "" {
			command += " --config " + quoteTUI2Arg(path)
		}
		if terminalID := strings.TrimSpace(cfg.TerminalID); terminalID != "" {
			target := terminalID
			if endpointID := strings.TrimSpace(cfg.EndpointID); endpointID != "" {
				target = endpointID + ":" + terminalID
			}
			command += " --attach " + quoteTUI2Arg(target)
		}
		args = append(args, "-shell", command)
	}
	if override := strings.TrimSpace(cfg.SocketOverride); override != "" {
		registryPath, cleanup, err := writeTUI2SocketRegistry(resolveV3Socket(override))
		if err != nil {
			return err
		}
		defer cleanup()
		args = append(args, "-endpoints", registryPath)
	}
	return runTUI2Process(hostPath, args, defaultTUI2Routes())
}

// ensureTUI2LocalStack 在 canonical socket 不可用时自动拉起 pool+access；
// 这与旧 CLI TUI 的 auto-start 行为一致。
func ensureTUI2LocalStack(socketPath, logFile, configPath string) error {
	if strings.TrimSpace(socketPath) == "" {
		return nil
	}
	if err := probeV3Socket(socketPath); err == nil {
		return nil
	}
	if err := startV3LocalStackForConfig(socketPath, logFile, configPath); err != nil {
		return fmt.Errorf("start local access stack: %w", err)
	}
	return waitForSocket(socketPath, 5*time.Second, func() error {
		return probeV3Socket(socketPath)
	})
}

// writeTUI2SocketRegistry 把显式 --socket 写成一个只含默认 local endpoint 的
// 临时 registry。tui2 把它作为更高优先级的显式 registry 读取，因此同名
// "local" endpoint 的 local-unix route 指向该 socket。
func writeTUI2SocketRegistry(socketPath string) (string, func(), error) {
	registry := endpointdomain.DefaultRegistry()
	endpoint, ok := registry.Endpoints[endpointdomain.DefaultEndpointID]
	if !ok {
		return "", func() {}, errors.New("default endpoint registry is empty")
	}
	route, ok := endpoint.Routes[endpointdomain.DefaultLocalRouteID]
	if !ok {
		return "", func() {}, errors.New("default endpoint registry has no local route")
	}
	route.Socket = socketPath
	endpoint.Routes[endpointdomain.DefaultLocalRouteID] = route
	registry.Endpoints[endpointdomain.DefaultEndpointID] = endpoint
	payload, err := endpointdomain.Encode(registry)
	if err != nil {
		return "", func() {}, err
	}
	file, err := os.CreateTemp("", "anytty-tui2-endpoints-*.yaml")
	if err != nil {
		return "", func() {}, err
	}
	path := file.Name()
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", func() {}, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", func() {}, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func defaultTUI2Routes() string {
	if value := strings.TrimSpace(os.Getenv("TUI2_ROUTES")); value != "" {
		return value
	}
	// 旧 TUI 入口是本地单机路径；remote route 仍可由用户显式开启。
	return "local-unix"
}

// locateTUI2Executable 查找 tui2 宿主：TUI2_BIN > 与 anytty 同目录 > PATH。
func locateTUI2Executable() (string, error) {
	if override := strings.TrimSpace(os.Getenv("TUI2_BIN")); override != "" {
		return override, nil
	}
	if exe, err := osExecutable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "tui2")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}
	if found, err := exec.LookPath("tui2"); err == nil {
		return found, nil
	}
	return "", errors.New("tui2 host not found: install it next to anytty or set TUI2_BIN")
}

// locateTUI2ShellExecutable 查找布局程序：TUI2_SHELL > 与 tui2 同目录 >
// 与 anytty 同目录 > PATH。空返回时宿主使用自己的默认查找。
func locateTUI2ShellExecutable(hostPath string) string {
	if override := strings.TrimSpace(os.Getenv("TUI2_SHELL")); override != "" {
		return override
	}
	candidates := []string{filepath.Join(filepath.Dir(hostPath), "tui2-shell")}
	if exe, err := osExecutable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "tui2-shell"))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if found, err := exec.LookPath("tui2-shell"); err == nil {
		return found
	}
	return ""
}

func runTUI2Process(hostPath string, args []string, routes string) error {
	command := exec.Command(hostPath, args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Env = append(os.Environ(), "TUI2_ROUTES="+routes)
	if err := command.Start(); err != nil {
		return err
	}
	forwarded := make(chan os.Signal, 1)
	signal.Notify(forwarded, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(forwarded)
	done := make(chan struct{})
	go func() {
		select {
		case sig := <-forwarded:
			_ = command.Process.Signal(sig)
		case <-done:
		}
	}()
	err := command.Wait()
	close(done)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("tui2 exited with status %d", exitErr.ExitCode())
		}
		return err
	}
	return nil
}

// quoteTUI2Arg 给 -shell 命令串里的单个参数加引号，支持含空格路径。
func quoteTUI2Arg(value string) string {
	if !strings.ContainsAny(value, " \t\"'\\") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
