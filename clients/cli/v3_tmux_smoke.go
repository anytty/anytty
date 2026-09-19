package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	endpointdomain "github.com/anytty/anytty/access/engine/endpoint"
)

// v3 tmux 黑盒冒烟：在隔离 XDG/socket/registry 下运行 anytty 默认入口，
// anytty 会自动拉起本地栈并启动新 TUI（tui2 宿主 + tui2-shell 布局程序）。
// 断言基于新 TUI 的稳定标记（picker / term-1 / footer / 终端回显），不再依赖
// 旧 TUI 的像素级 golden。

const (
	v3TmuxSmokeTimeout = 20 * time.Second
	v3TmuxQuitTimeout  = 15 * time.Second
)

type v3TmuxSmokeResult struct {
	Session      string
	TerminalID   string
	SentInput    string
	ArtifactDir  string
	ANSIPath     string
	PlainPath    string
	DaemonLog    string
	SocketPath   string
	TimelinePath string
}

type v3TmuxResizeSmokeResult struct {
	Session      string
	TerminalID   string
	BeforeSize   string
	AfterSize    string
	ArtifactDir  string
	ANSIPath     string
	PlainPath    string
	DaemonLog    string
	SocketPath   string
	TimelinePath string
}

type v3TmuxANSISmokeResult struct {
	Session      string
	TerminalID   string
	ArtifactDir  string
	ANSIPath     string
	PlainPath    string
	DaemonLog    string
	SocketPath   string
	TimelinePath string
}

type v3TmuxStabilitySmokeResult struct {
	Rounds       int
	Artifacts    []string
	ArtifactDir  string
	TimelinePath string
}

type v3TmuxHarness struct {
	session      string
	artifactDir  string
	socketPath   string
	daemonLog    string
	harnessEnv   []string
	anyttyBin    string
	timelinePath string
}

// runV3TmuxSmoke 覆盖最小默认路径：cold start 打开 picker、Enter 创建并绑定
// terminal、输入回显、Ctrl-Q 确认退出。
func runV3TmuxSmoke(ctx context.Context, anyttyBin string) (v3TmuxSmokeResult, error) {
	harness, err := newV3TmuxHarness(ctx, anyttyBin, "smoke")
	if err != nil {
		return v3TmuxSmokeResult{}, err
	}
	defer harness.close()
	if err := harness.waitForCapture(ctx, `select a terminal`); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.sendKeys(ctx, "Enter"); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.waitForCapture(ctx, `bound local:`); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.sendLiteral(ctx, "echo tui2-smoke-ok"); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.sendKeys(ctx, "Enter"); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.waitForCapture(ctx, `tui2-smoke-ok`); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	ansiPath, plainPath, err := harness.captureArtifacts(ctx)
	if err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.quit(ctx); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	return v3TmuxSmokeResult{
		Session: harness.session, TerminalID: "term-1", SentInput: "echo tui2-smoke-ok",
		ArtifactDir: harness.artifactDir, ANSIPath: ansiPath, PlainPath: plainPath,
		DaemonLog: harness.daemonLog, SocketPath: harness.socketPath, TimelinePath: harness.timelinePath,
	}, nil
}

// runV3TmuxTerminalSmoke 与 plain smoke 等价，但以显式 `anytty attach` 入口
// 运行，验证 CLI attach 传递的 -attach 目标被新 TUI 绑定。
func runV3TmuxTerminalSmoke(ctx context.Context, anyttyBin string) (v3TmuxSmokeResult, error) {
	harness, err := newV3TmuxHarnessWithArgs(ctx, anyttyBin, "terminal-smoke", []string{"attach", "term-1"})
	if err != nil {
		return v3TmuxSmokeResult{}, err
	}
	defer harness.close()
	// attach 目标尚不存在时 picker 兜底；Enter 创建 term-1。
	if err := harness.waitForCapture(ctx, `select a terminal`); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.sendKeys(ctx, "Enter"); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.waitForCapture(ctx, `bound local:`); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.sendLiteral(ctx, "echo attach-smoke-ok"); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.sendKeys(ctx, "Enter"); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.waitForCapture(ctx, `attach-smoke-ok`); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	ansiPath, plainPath, err := harness.captureArtifacts(ctx)
	if err != nil {
		return v3TmuxSmokeResult{}, err
	}
	if err := harness.quit(ctx); err != nil {
		return v3TmuxSmokeResult{}, err
	}
	return v3TmuxSmokeResult{
		Session: harness.session, TerminalID: "term-1", SentInput: "echo attach-smoke-ok",
		ArtifactDir: harness.artifactDir, ANSIPath: ansiPath, PlainPath: plainPath,
		DaemonLog: harness.daemonLog, SocketPath: harness.socketPath, TimelinePath: harness.timelinePath,
	}, nil
}

// runV3TmuxResizeSmoke 在 tmux 中放大窗口，并断言 PTY 通过 `stty size` 观察到
// 同向尺寸变化。
func runV3TmuxResizeSmoke(ctx context.Context, anyttyBin string) (v3TmuxResizeSmokeResult, error) {
	harness, err := newV3TmuxHarnessWithSize(ctx, anyttyBin, "resize-smoke", 80, 24)
	if err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	defer harness.close()
	if err := harness.createTerminal(ctx); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	before, err := harness.terminalSize(ctx)
	if err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	if err := runTmuxCommand(ctx, "resize-window", "-t", harness.session, "-x", "110", "-y", "36"); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	after, err := harness.waitForLargerTerminalSize(ctx, before)
	if err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	ansiPath, plainPath, err := harness.captureArtifacts(ctx)
	if err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	if err := harness.quit(ctx); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	return v3TmuxResizeSmokeResult{
		Session: harness.session, TerminalID: "term-1", BeforeSize: before, AfterSize: after,
		ArtifactDir: harness.artifactDir, ANSIPath: ansiPath, PlainPath: plainPath,
		DaemonLog: harness.daemonLog, SocketPath: harness.socketPath, TimelinePath: harness.timelinePath,
	}, nil
}

// runV3TmuxANSISmoke 断言 ANSI SGR 与 Unicode 文本都能穿过新 TUI 落到 pane capture。
func runV3TmuxANSISmoke(ctx context.Context, anyttyBin string) (v3TmuxANSISmokeResult, error) {
	harness, err := newV3TmuxHarness(ctx, anyttyBin, "ansi-smoke")
	if err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	defer harness.close()
	if err := harness.createTerminal(ctx); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	if err := harness.sendLiteral(ctx, `printf '\033[31mANSI-RED\033[0m\n'`); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	if err := harness.sendKeys(ctx, "Enter"); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	if err := harness.waitForCapture(ctx, `ANSI-RED`); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	if err := harness.sendLiteral(ctx, "echo UNICODE-你好-😀"); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	if err := harness.sendKeys(ctx, "Enter"); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	if err := harness.waitForCapture(ctx, `UNICODE-你好-😀`); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	ansiPath, plainPath, err := harness.captureArtifacts(ctx)
	if err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	if err := harness.quit(ctx); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	ansiCapture, err := os.ReadFile(ansiPath)
	if err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	if !strings.Contains(string(ansiCapture), "[3") && !strings.Contains(string(ansiCapture), "[31") {
		return v3TmuxANSISmokeResult{}, fmt.Errorf("ANSI capture lost SGR sequences")
	}
	return v3TmuxANSISmokeResult{
		Session: harness.session, TerminalID: "term-1",
		ArtifactDir: harness.artifactDir, ANSIPath: ansiPath, PlainPath: plainPath,
		DaemonLog: harness.daemonLog, SocketPath: harness.socketPath, TimelinePath: harness.timelinePath,
	}, nil
}

// runV3TmuxStabilitySmoke 连续多轮启动/退出，验证每轮都能到达 picker 并干净退出。
func runV3TmuxStabilitySmoke(ctx context.Context, anyttyBin string, rounds int) (v3TmuxStabilitySmokeResult, error) {
	if rounds <= 0 {
		rounds = 1
	}
	artifactDir, err := os.MkdirTemp("", "anytty-tmux-stability-*")
	if err != nil {
		return v3TmuxStabilitySmokeResult{}, err
	}
	timelinePath := filepath.Join(artifactDir, "timeline.txt")
	var artifacts []string
	for round := 0; round < rounds; round++ {
		harness, err := newV3TmuxHarness(ctx, anyttyBin, fmt.Sprintf("stability-%d", round))
		if err != nil {
			return v3TmuxStabilitySmokeResult{}, err
		}
		if err := harness.waitForCapture(ctx, `select a terminal`); err != nil {
			harness.close()
			return v3TmuxStabilitySmokeResult{}, fmt.Errorf("round %d: %w", round, err)
		}
		ansiPath, plainPath, err := harness.captureArtifacts(ctx)
		if err != nil {
			harness.close()
			return v3TmuxStabilitySmokeResult{}, err
		}
		artifacts = append(artifacts, ansiPath, plainPath)
		if err := harness.quit(ctx); err != nil {
			harness.close()
			return v3TmuxStabilitySmokeResult{}, fmt.Errorf("round %d: %w", round, err)
		}
		_ = appendFile(timelinePath, fmt.Sprintf("round %d ok session=%s\n", round, harness.session))
		harness.close()
	}
	return v3TmuxStabilitySmokeResult{Rounds: rounds, Artifacts: artifacts, ArtifactDir: artifactDir, TimelinePath: timelinePath}, nil
}

// newV3TmuxHarness 建立隔离的 tmux 会话并启动 anytty 默认入口。
func newV3TmuxHarness(ctx context.Context, anyttyBin, tag string) (*v3TmuxHarness, error) {
	return newV3TmuxHarnessWithArgs(ctx, anyttyBin, tag, nil)
}

func newV3TmuxHarnessWithArgs(ctx context.Context, anyttyBin, tag string, args []string) (*v3TmuxHarness, error) {
	return newV3TmuxHarnessWithSizeAndArgs(ctx, anyttyBin, tag, 100, 30, args)
}

func newV3TmuxHarnessWithSize(ctx context.Context, anyttyBin, tag string, cols, rows int) (*v3TmuxHarness, error) {
	return newV3TmuxHarnessWithSizeAndArgs(ctx, anyttyBin, tag, cols, rows, nil)
}

func newV3TmuxHarnessWithSizeAndArgs(ctx context.Context, anyttyBin, tag string, cols, rows int, args []string) (*v3TmuxHarness, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return nil, fmt.Errorf("tmux is not installed: %w", err)
	}
	baseDir, err := os.MkdirTemp("", "anytty-tmux-"+tag+"-*")
	if err != nil {
		return nil, err
	}
	configHome := filepath.Join(baseDir, "config")
	stateHome := filepath.Join(baseDir, "state")
	runtimeDir := filepath.Join(baseDir, "run")
	for _, dir := range []string{configHome, stateHome, runtimeDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
	}
	socketPath := filepath.Join(runtimeDir, "anytty-v2-wire7.sock")
	if err := writeV3TmuxEndpointRegistry(configHome, socketPath); err != nil {
		return nil, err
	}
	session := fmt.Sprintf("anytty-%s-%d", tag, os.Getpid())
	_ = runTmuxCommand(ctx, "kill-session", "-t", session)
	tui2Bin := strings.TrimSpace(os.Getenv("TUI2_BIN"))
	tui2Shell := strings.TrimSpace(os.Getenv("TUI2_SHELL"))
	envPairs := []string{
		"XDG_CONFIG_HOME=" + configHome,
		"XDG_STATE_HOME=" + stateHome,
		"XDG_RUNTIME_DIR=" + runtimeDir,
		"ANYTTY_ALLOW_NESTED=1",
		"TUI2_ROUTES=local-unix",
	}
	if tui2Bin != "" {
		envPairs = append(envPairs, "TUI2_BIN="+tui2Bin)
	}
	if tui2Shell != "" {
		envPairs = append(envPairs, "TUI2_SHELL="+tui2Shell)
	}
	logPath := filepath.Join(baseDir, "anytty.log")
	commandParts := []string{"env"}
	commandParts = append(commandParts, envPairs...)
	commandParts = append(commandParts, shellQuote(anyttyBin), "--socket", shellQuote(socketPath), "--log-file", shellQuote(logPath))
	for _, arg := range args {
		commandParts = append(commandParts, shellQuote(arg))
	}
	paneCommand := fmt.Sprintf("bash -c '%s; echo ANYTTY-EXIT:$?; sleep 5'", strings.Join(commandParts, " "))
	if err := runTmuxCommand(ctx, "new-session", "-d", "-s", session, "-x", strconv.Itoa(cols), "-y", strconv.Itoa(rows), paneCommand); err != nil {
		return nil, err
	}
	return &v3TmuxHarness{
		session: session, artifactDir: baseDir, socketPath: socketPath, daemonLog: logPath,
		anyttyBin: anyttyBin, timelinePath: filepath.Join(baseDir, "timeline.txt"),
	}, nil
}

func writeV3TmuxEndpointRegistry(configHome, socketPath string) error {
	registry := endpointdomain.DefaultRegistry()
	endpoint, ok := registry.Endpoints[endpointdomain.DefaultEndpointID]
	if !ok {
		return fmt.Errorf("default endpoint registry is empty")
	}
	route, ok := endpoint.Routes[endpointdomain.DefaultLocalRouteID]
	if !ok {
		return fmt.Errorf("default endpoint registry has no local route")
	}
	route.Socket = socketPath
	endpoint.Routes[endpointdomain.DefaultLocalRouteID] = route
	registry.Endpoints[endpointdomain.DefaultEndpointID] = endpoint
	payload, err := endpointdomain.Encode(registry)
	if err != nil {
		return err
	}
	dir := filepath.Join(configHome, "anytty")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, endpointdomain.DefaultFileName), payload, 0o600)
}

func (harness *v3TmuxHarness) close() {
	if harness == nil || harness.session == "" {
		return
	}
	_ = runTmuxCommand(context.Background(), "kill-session", "-t", harness.session)
}

// createTerminal 走 picker 冷启动：等待 picker，Enter 创建并绑定 term-1。
func (harness *v3TmuxHarness) createTerminal(ctx context.Context) error {
	if err := harness.waitForCapture(ctx, `select a terminal`); err != nil {
		return err
	}
	if err := harness.sendKeys(ctx, "Enter"); err != nil {
		return err
	}
	return harness.waitForCapture(ctx, `bound local:`)
}

// quit 通过 host 确认框退出：Ctrl-Q 后 Enter。
func (harness *v3TmuxHarness) quit(ctx context.Context) error {
	if err := harness.sendKeys(ctx, "C-q"); err != nil {
		return err
	}
	if err := harness.waitForCapture(ctx, `Quit tui2`); err != nil {
		return err
	}
	if err := harness.sendKeys(ctx, "Enter"); err != nil {
		return err
	}
	return harness.waitForCapture(ctx, `ANYTTY-EXIT:0`)
}

func (harness *v3TmuxHarness) sendKeys(ctx context.Context, keys ...string) error {
	args := append([]string{"send-keys", "-t", harness.session}, keys...)
	return runTmuxCommand(ctx, args...)
}

func (harness *v3TmuxHarness) sendLiteral(ctx context.Context, text string) error {
	return runTmuxCommand(ctx, "send-keys", "-t", harness.session, "-l", text)
}

func (harness *v3TmuxHarness) capture(ctx context.Context, ansi bool) (string, error) {
	args := []string{"capture-pane", "-p", "-t", harness.session}
	if ansi {
		args = append(args, "-e")
	}
	command := exec.CommandContext(ctx, "tmux", args...)
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("tmux capture-pane: %w", err)
	}
	return string(output), nil
}

func (harness *v3TmuxHarness) waitForCapture(ctx context.Context, marker string) error {
	return harness.waitForCapturePattern(ctx, regexp.QuoteMeta(marker))
}

func (harness *v3TmuxHarness) waitForCapturePattern(ctx context.Context, pattern string) error {
	deadline := time.Now().Add(v3TmuxSmokeTimeout)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		capture, err := harness.capture(ctx, false)
		if err == nil && regexp.MustCompile(pattern).MatchString(capture) {
			_ = appendFile(harness.timelinePath, fmt.Sprintf("capture %s\n", pattern))
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	capture, _ := harness.capture(context.Background(), false)
	return fmt.Errorf("timed out waiting for %q in tmux capture; last capture:\n%s", pattern, capture)
}

func (harness *v3TmuxHarness) captureArtifacts(ctx context.Context) (string, string, error) {
	ansPath := filepath.Join(harness.artifactDir, "capture-ansi.txt")
	plainPath := filepath.Join(harness.artifactDir, "capture-plain.txt")
	ansi, err := harness.capture(ctx, true)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(ansPath, []byte(ansi), 0o600); err != nil {
		return "", "", err
	}
	plain, err := harness.capture(ctx, false)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(plainPath, []byte(plain), 0o600); err != nil {
		return "", "", err
	}
	return ansPath, plainPath, nil
}

var v3TmuxSizePattern = regexp.MustCompile(`(\d+)\s+(\d+)`)

// terminalSize 在终端里运行 `stty size` 并解析 "rows cols"。
func (harness *v3TmuxHarness) terminalSize(ctx context.Context) (string, error) {
	if err := harness.sendLiteral(ctx, "stty size"); err != nil {
		return "", err
	}
	if err := harness.sendKeys(ctx, "Enter"); err != nil {
		return "", err
	}
	deadline := time.Now().Add(v3TmuxSmokeTimeout)
	for time.Now().Before(deadline) {
		capture, err := harness.capture(ctx, false)
		if err == nil {
			if size, ok := lastV3TmuxSize(capture); ok {
				return size, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", fmt.Errorf("timed out reading stty size from the TUI terminal")
}

func (harness *v3TmuxHarness) waitForLargerTerminalSize(ctx context.Context, before string) (string, error) {
	beforeRows, beforeCols := v3TmuxSizeValues(before)
	deadline := time.Now().Add(v3TmuxSmokeTimeout)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		after, err := harness.terminalSize(ctx)
		if err == nil {
			rows, cols := v3TmuxSizeValues(after)
			if rows > beforeRows && cols > beforeCols {
				return after, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", fmt.Errorf("terminal size did not grow after resize (before=%s)", before)
}

func lastV3TmuxSize(capture string) (string, bool) {
	matches := v3TmuxSizePattern.FindAllString(capture, -1)
	// picker/footer 里也有数字；取最后一行看起来像尺寸的样本，且要求 > 0。
	for index := len(matches) - 1; index >= 0; index-- {
		rows, cols := v3TmuxSizeValues(matches[index])
		if rows > 0 && cols > 0 {
			return matches[index], true
		}
	}
	return "", false
}

func v3TmuxSizeValues(size string) (int, int) {
	match := v3TmuxSizePattern.FindStringSubmatch(size)
	if len(match) != 3 {
		return 0, 0
	}
	rows, _ := strconv.Atoi(match[1])
	cols, _ := strconv.Atoi(match[2])
	return rows, cols
}

func runTmuxCommand(ctx context.Context, args ...string) error {
	command := exec.CommandContext(ctx, "tmux", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux %s: %w\n%s", strings.Join(args, " "), err, output)
	}
	return nil
}

func appendFile(path string, text string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(text)
	return err
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if !strings.ContainsAny(value, " \t\"'\\$`") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
