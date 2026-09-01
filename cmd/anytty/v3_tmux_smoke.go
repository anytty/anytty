package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anytty/anytty/tui/render"
)

const (
	v3TmuxDaemonReadyTimeout  = 10 * time.Second
	v3TmuxSurfaceReadyTimeout = 10 * time.Second
)

type v3TmuxSmokeResult struct {
	Session     string
	ArtifactDir string
	ANSIPath    string
	PlainPath   string
	Captured    string
	SentInput   string
}

type v3TmuxTerminalSmokeResult struct {
	Session      string
	TerminalID   string
	ArtifactDir  string
	ANSIPath     string
	PlainPath    string
	DaemonLog    string
	SocketPath   string
	TimelinePath string
	Captured     string
	SentInput    string
}

type v3TmuxResizeSmokeResult struct {
	Session      string
	TerminalID   string
	ArtifactDir  string
	ANSIPath     string
	PlainPath    string
	DaemonLog    string
	SocketPath   string
	TimelinePath string
	Captured     string
	BeforeSize   string
	AfterSize    string
	WindowSize   string
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
	Captured     string
	ANSICaptured string
}

type v3TmuxEmojiDotsSmokeResult struct {
	Session      string
	TerminalID   string
	ArtifactDir  string
	ANSIPath     string
	PlainPath    string
	DaemonLog    string
	SocketPath   string
	TimelinePath string
	Captured     string
	BeforeSize   string
	AfterSize    string
	DotsVisible  bool
}

type v3TmuxVisualCompareResult struct {
	Session             string
	ArtifactDir         string
	CurrentANSIPath     string
	CurrentPlainPath    string
	TargetPath          string
	DiffPath            string
	StylePath           string
	StyleDiffPath       string
	CurrentStyleMapPath string
	TargetStyleMapPath  string
	StyleMapDiffPath    string
	SummaryPath         string
	Mismatches          int
	StyleMismatches     int
	StyleMapMismatches  int
}

type v3TmuxStabilitySmokeResult struct {
	ArtifactDir  string
	TimelinePath string
	Rounds       int
	Artifacts    []string
}

func runV3TmuxSmoke(ctx context.Context) (v3TmuxSmokeResult, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return v3TmuxSmokeResult{}, fmt.Errorf("tmux smoke requires tmux in PATH: %w", err)
	}
	artifactDir, err := os.MkdirTemp("", "anytty-v3-tmux-smoke-*")
	if err != nil {
		return v3TmuxSmokeResult{}, err
	}
	session := fmt.Sprintf("anytty-v3-smoke-%d", time.Now().UnixNano())
	target := session + ":0.0"
	cleanup := func() {
		_ = runTmuxCommand(context.Background(), "kill-session", "-t", session)
	}
	defer cleanup()

	script := "printf 'anytty tmux harness ready\\n'; " +
		"IFS= read -r line; " +
		"printf 'anytty tmux input:%s\\n' \"$line\"; " +
		"sleep 30"
	if err := runTmuxCommand(ctx, "new-session", "-d", "-x", "100", "-y", "30", "-s", session, "/bin/sh", "-lc", script); err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxSmokeResult{}, err
	}
	if err := waitForTmuxCapture(ctx, target, "anytty tmux harness ready", 2*time.Second); err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxSmokeResult{}, err
	}
	sent := "hello-from-tmux"
	if err := runTmuxCommand(ctx, "send-keys", "-t", target, sent, "Enter"); err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxSmokeResult{}, err
	}
	if err := waitForTmuxCapture(ctx, target, "anytty tmux input:"+sent, 2*time.Second); err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxSmokeResult{}, err
	}
	ansi, err := captureTmuxPane(ctx, target, true)
	if err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxSmokeResult{}, err
	}
	plain, err := captureTmuxPane(ctx, target, false)
	if err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxSmokeResult{}, err
	}
	ansiPath := filepath.Join(artifactDir, "capture.ansi")
	plainPath := filepath.Join(artifactDir, "capture.txt")
	if err := os.WriteFile(ansiPath, []byte(ansi), 0o600); err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxSmokeResult{}, err
	}
	if err := os.WriteFile(plainPath, []byte(plain), 0o600); err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxSmokeResult{}, err
	}
	return v3TmuxSmokeResult{
		Session:     session,
		ArtifactDir: artifactDir,
		ANSIPath:    ansiPath,
		PlainPath:   plainPath,
		Captured:    plain,
		SentInput:   sent,
	}, nil
}

func runV3TmuxTerminalSmoke(ctx context.Context, anyttyBin string) (v3TmuxTerminalSmokeResult, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return v3TmuxTerminalSmokeResult{}, fmt.Errorf("tmux terminal smoke requires tmux in PATH: %w", err)
	}
	if anyttyBin == "" {
		return v3TmuxTerminalSmokeResult{}, fmt.Errorf("tmux terminal smoke requires a anytty binary path")
	}
	artifactDir, err := os.MkdirTemp("", "anytty-v3-tmux-terminal-*")
	if err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	configHome, stateHome, harnessEnv, err := v3TmuxHarnessEnvironment(artifactDir)
	if err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxTerminalSmokeResult{}, err
	}
	removeArtifacts := true
	defer func() {
		if removeArtifacts {
			_ = os.RemoveAll(artifactDir)
		}
	}()

	socketPath := filepath.Join(artifactDir, "core.sock")
	daemonLog := filepath.Join(artifactDir, "daemon.log")
	timelinePath := filepath.Join(artifactDir, "timeline.txt")
	session := fmt.Sprintf("anytty-v3-terminal-%d", time.Now().UnixNano())
	target := session + ":0.0"
	appendTimeline := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...) + "\n"
		_ = appendFile(timelinePath, line)
	}

	daemonStdout, err := os.Create(filepath.Join(artifactDir, "daemon.stdout"))
	if err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	defer daemonStdout.Close()
	daemonStderr, err := os.Create(filepath.Join(artifactDir, "daemon.stderr"))
	if err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	defer daemonStderr.Close()
	daemonCtx, stopDaemonCtx := context.WithCancel(ctx)
	defer stopDaemonCtx()
	daemonCmd := exec.CommandContext(daemonCtx, anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "daemon")
	daemonCmd.Env = harnessEnv
	daemonCmd.Stdout = daemonStdout
	daemonCmd.Stderr = daemonStderr
	if err := daemonCmd.Start(); err != nil {
		return v3TmuxTerminalSmokeResult{}, fmt.Errorf("start core-v2 daemon: %w", err)
	}
	defer func() {
		stopDaemonCtx()
		if daemonCmd.Process != nil {
			_ = daemonCmd.Process.Kill()
		}
		_ = daemonCmd.Wait()
	}()
	appendTimeline("daemon start socket=%s log=%s", socketPath, daemonLog)
	if err := waitForSocket(socketPath, v3TmuxDaemonReadyTimeout, func() error {
		client, dialErr := dialV3Client(socketPath)
		if dialErr != nil {
			return dialErr
		}
		return client.Close()
	}); err != nil {
		return v3TmuxTerminalSmokeResult{}, tmuxDaemonReadyError(err, daemonStdout, daemonStderr)
	}
	appendTimeline("daemon ready")

	terminalIDPath := filepath.Join(artifactDir, "terminal.id")
	scriptPath := filepath.Join(artifactDir, "tmux-terminal-smoke.sh")
	terminalScript := "printf 'anytty-pty-ready\\n'; while IFS= read -r line; do printf 'anytty-pty-echo:%s\\n' \"$line\"; done"
	script := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"export TERM=xterm-256color",
		"export ANYTTY=0",
		"export XDG_CONFIG_HOME=" + shellQuote(configHome),
		"export XDG_STATE_HOME=" + shellQuote(stateHome),
		"anytty_bin=" + shellQuote(anyttyBin),
		"socket=" + shellQuote(socketPath),
		"log_file=" + shellQuote(daemonLog),
		"terminal_id_path=" + shellQuote(terminalIDPath),
		"ls_before_path=" + shellQuote(filepath.Join(artifactDir, "ls.before.txt")),
		"printf 'anytty tmux terminal harness ready\\n'",
		"id=\"$($anytty_bin --socket \"$socket\" --log-file \"$log_file\" v3 new --name tmux-e2e -- /bin/sh -c " + shellQuote(terminalScript) + ")\"",
		"printf '%s\\n' \"$id\" > \"$terminal_id_path\"",
		"printf 'anytty terminal id:%s\\n' \"$id\"",
		"$anytty_bin --socket \"$socket\" --log-file \"$log_file\" v3 ls > \"$ls_before_path\"",
		"exec $anytty_bin --socket \"$socket\" --log-file \"$log_file\" v3 attach \"$id\"",
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	appendTimeline("tmux script=%s", scriptPath)

	cleanupTmux := func() {
		_ = runTmuxCommand(context.Background(), "kill-session", "-t", session)
	}
	defer cleanupTmux()
	if err := runTmuxCommand(ctx, "new-session", "-d", "-x", "100", "-y", "30", "-s", session, "/bin/sh", scriptPath); err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	appendTimeline("tmux session=%s", session)
	terminalIDBytes, err := waitForFileContent(ctx, terminalIDPath, v3TmuxSurfaceReadyTimeout)
	if err != nil {
		return v3TmuxTerminalSmokeResult{}, fmt.Errorf("read terminal id artifact: %w", err)
	}
	terminalID := strings.TrimSpace(string(terminalIDBytes))
	appendTimeline("terminal created id=%s", terminalID)
	if err := waitForTmuxCapture(ctx, target, "anytty-pty-ready", v3TmuxSurfaceReadyTimeout); err != nil {
		removeArtifacts = false
		return v3TmuxTerminalSmokeResult{}, fmt.Errorf("%w; artifacts=%s", err, artifactDir)
	}
	appendTimeline("attach rendered initial live surface")

	sent := "tmux-live-input"
	if err := runTmuxCommand(ctx, "send-keys", "-t", target, sent, "Enter"); err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	appendTimeline("tmux send-keys input=%s", sent)
	if err := waitForTmuxCapture(ctx, target, "anytty-pty-echo:"+sent, v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	appendTimeline("live surface echoed input")

	ansi, err := captureTmuxPane(ctx, target, true)
	if err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	plain, err := captureTmuxPane(ctx, target, false)
	if err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	ansiPath := filepath.Join(artifactDir, "capture.ansi")
	plainPath := filepath.Join(artifactDir, "capture.txt")
	if err := os.WriteFile(ansiPath, []byte(ansi), 0o600); err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	if err := os.WriteFile(plainPath, []byte(plain), 0o600); err != nil {
		return v3TmuxTerminalSmokeResult{}, err
	}
	appendTimeline("captured ansi=%s plain=%s", ansiPath, plainPath)

	_ = runAnyTTYCommand(context.Background(), anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "v3", "kill", terminalID)
	_ = runAnyTTYCommand(context.Background(), anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "v3", "rm", terminalID)
	appendTimeline("terminal cleanup id=%s", terminalID)
	removeArtifacts = false
	return v3TmuxTerminalSmokeResult{
		Session:      session,
		TerminalID:   terminalID,
		ArtifactDir:  artifactDir,
		ANSIPath:     ansiPath,
		PlainPath:    plainPath,
		DaemonLog:    daemonLog,
		SocketPath:   socketPath,
		TimelinePath: timelinePath,
		Captured:     plain,
		SentInput:    sent,
	}, nil
}

func runV3TmuxResizeSmoke(ctx context.Context, anyttyBin string) (v3TmuxResizeSmokeResult, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return v3TmuxResizeSmokeResult{}, fmt.Errorf("tmux resize smoke requires tmux in PATH: %w", err)
	}
	if anyttyBin == "" {
		return v3TmuxResizeSmokeResult{}, fmt.Errorf("tmux resize smoke requires a anytty binary path")
	}
	artifactDir, err := os.MkdirTemp("", "anytty-v3-tmux-resize-*")
	if err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	configHome, stateHome, harnessEnv, err := v3TmuxHarnessEnvironment(artifactDir)
	if err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxResizeSmokeResult{}, err
	}
	removeArtifacts := true
	defer func() {
		if removeArtifacts {
			_ = os.RemoveAll(artifactDir)
		}
	}()

	socketPath := filepath.Join(artifactDir, "core.sock")
	daemonLog := filepath.Join(artifactDir, "daemon.log")
	timelinePath := filepath.Join(artifactDir, "timeline.txt")
	session := fmt.Sprintf("anytty-v3-resize-%d", time.Now().UnixNano())
	target := session + ":0.0"
	appendTimeline := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...) + "\n"
		_ = appendFile(timelinePath, line)
	}

	daemonCtx, stopDaemonCtx := context.WithCancel(ctx)
	defer stopDaemonCtx()
	daemonCmd := exec.CommandContext(daemonCtx, anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "daemon")
	daemonCmd.Env = harnessEnv
	daemonCmd.Stdout = io.Discard
	daemonCmd.Stderr = io.Discard
	if err := daemonCmd.Start(); err != nil {
		return v3TmuxResizeSmokeResult{}, fmt.Errorf("start core-v2 daemon: %w", err)
	}
	defer func() {
		stopDaemonCtx()
		if daemonCmd.Process != nil {
			_ = daemonCmd.Process.Kill()
		}
		_ = daemonCmd.Wait()
	}()
	appendTimeline("daemon start socket=%s log=%s", socketPath, daemonLog)
	if err := waitForSocket(socketPath, v3TmuxDaemonReadyTimeout, func() error {
		client, dialErr := dialV3Client(socketPath)
		if dialErr != nil {
			return dialErr
		}
		return client.Close()
	}); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	appendTimeline("daemon ready")

	terminalIDPath := filepath.Join(artifactDir, "terminal.id")
	scriptPath := filepath.Join(artifactDir, "tmux-resize-smoke.sh")
	terminalScript := "printf 'anytty-pty-ready\\n'\n" +
		"while IFS= read -r line; do\n" +
		"  case \"$line\" in\n" +
		"    size-*) set -- $(stty size); printf 'anytty-pty-size:%s:%sx%s\\n' \"$line\" \"$2\" \"$1\" ;;\n" +
		"    *) printf 'anytty-pty-echo:%s\\n' \"$line\" ;;\n" +
		"  esac\n" +
		"done\n"
	script := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"export TERM=xterm-256color",
		"export ANYTTY=0",
		"export XDG_CONFIG_HOME=" + shellQuote(configHome),
		"export XDG_STATE_HOME=" + shellQuote(stateHome),
		"anytty_bin=" + shellQuote(anyttyBin),
		"socket=" + shellQuote(socketPath),
		"log_file=" + shellQuote(daemonLog),
		"terminal_id_path=" + shellQuote(terminalIDPath),
		"printf 'anytty tmux resize harness ready\\n'",
		"id=\"$($anytty_bin --socket \"$socket\" --log-file \"$log_file\" v3 new --name tmux-resize -- /bin/sh -c " + shellQuote(terminalScript) + ")\"",
		"printf '%s\\n' \"$id\" > \"$terminal_id_path\"",
		"printf 'anytty resize terminal id:%s\\n' \"$id\"",
		"exec $anytty_bin --socket \"$socket\" --log-file \"$log_file\" v3 attach \"$id\"",
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	appendTimeline("tmux script=%s", scriptPath)

	cleanupTmux := func() {
		_ = runTmuxCommand(context.Background(), "kill-session", "-t", session)
	}
	defer cleanupTmux()
	if err := runTmuxCommand(ctx, "new-session", "-d", "-x", "100", "-y", "30", "-s", session, "/bin/sh", scriptPath); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	appendTimeline("tmux session=%s initial=100x30", session)
	terminalIDBytes, err := waitForFileContent(ctx, terminalIDPath, v3TmuxSurfaceReadyTimeout)
	if err != nil {
		return v3TmuxResizeSmokeResult{}, fmt.Errorf("read terminal id artifact: %w", err)
	}
	terminalID := strings.TrimSpace(string(terminalIDBytes))
	appendTimeline("terminal created id=%s", terminalID)
	if err := waitForTmuxCapture(ctx, target, "anytty-pty-ready", v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}

	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "size-before", "Enter"); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	beforeMarker := "anytty-pty-size:size-before:"
	if err := waitForTmuxCapture(ctx, target, beforeMarker+"98x26", v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	beforeCapture, _ := captureTmuxPane(ctx, target, false)
	beforeSize := lastMarkerSuffix(beforeCapture, beforeMarker)
	appendTimeline("before size=%s", beforeSize)

	windowSize := "120x40"
	if err := runTmuxCommand(ctx, "resize-window", "-t", session, "-x", "120", "-y", "40"); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	appendTimeline("tmux resize-window=%s", windowSize)
	time.Sleep(250 * time.Millisecond)
	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "size-after", "Enter"); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	afterMarker := "anytty-pty-size:size-after:"
	if err := waitForTmuxCapture(ctx, target, afterMarker+"118x36", v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	afterCapture, _ := captureTmuxPane(ctx, target, false)
	afterSize := lastMarkerSuffix(afterCapture, afterMarker)
	appendTimeline("after size=%s", afterSize)

	ansi, err := captureTmuxPane(ctx, target, true)
	if err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	plain, err := captureTmuxPane(ctx, target, false)
	if err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	ansiPath := filepath.Join(artifactDir, "capture.ansi")
	plainPath := filepath.Join(artifactDir, "capture.txt")
	if err := os.WriteFile(ansiPath, []byte(ansi), 0o600); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	if err := os.WriteFile(plainPath, []byte(plain), 0o600); err != nil {
		return v3TmuxResizeSmokeResult{}, err
	}
	appendTimeline("captured ansi=%s plain=%s", ansiPath, plainPath)

	_ = runAnyTTYCommand(context.Background(), anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "v3", "kill", terminalID)
	_ = runAnyTTYCommand(context.Background(), anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "v3", "rm", terminalID)
	appendTimeline("terminal cleanup id=%s", terminalID)
	removeArtifacts = false
	return v3TmuxResizeSmokeResult{
		Session:      session,
		TerminalID:   terminalID,
		ArtifactDir:  artifactDir,
		ANSIPath:     ansiPath,
		PlainPath:    plainPath,
		DaemonLog:    daemonLog,
		SocketPath:   socketPath,
		TimelinePath: timelinePath,
		Captured:     plain,
		BeforeSize:   beforeSize,
		AfterSize:    afterSize,
		WindowSize:   windowSize,
	}, nil
}

func runV3TmuxANSISmoke(ctx context.Context, anyttyBin string) (v3TmuxANSISmokeResult, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return v3TmuxANSISmokeResult{}, fmt.Errorf("tmux ansi smoke requires tmux in PATH: %w", err)
	}
	if anyttyBin == "" {
		return v3TmuxANSISmokeResult{}, fmt.Errorf("tmux ansi smoke requires a anytty binary path")
	}
	artifactDir, err := os.MkdirTemp("", "anytty-v3-tmux-ansi-*")
	if err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	configHome, stateHome, harnessEnv, err := v3TmuxHarnessEnvironment(artifactDir)
	if err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxANSISmokeResult{}, err
	}
	removeArtifacts := true
	defer func() {
		if removeArtifacts {
			_ = os.RemoveAll(artifactDir)
		}
	}()

	socketPath := filepath.Join(artifactDir, "core.sock")
	daemonLog := filepath.Join(artifactDir, "daemon.log")
	timelinePath := filepath.Join(artifactDir, "timeline.txt")
	session := fmt.Sprintf("anytty-v3-ansi-%d", time.Now().UnixNano())
	target := session + ":0.0"
	appendTimeline := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...) + "\n"
		_ = appendFile(timelinePath, line)
	}

	daemonCtx, stopDaemonCtx := context.WithCancel(ctx)
	defer stopDaemonCtx()
	daemonCmd := exec.CommandContext(daemonCtx, anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "daemon")
	daemonCmd.Env = harnessEnv
	daemonCmd.Stdout = io.Discard
	daemonCmd.Stderr = io.Discard
	if err := daemonCmd.Start(); err != nil {
		return v3TmuxANSISmokeResult{}, fmt.Errorf("start core-v2 daemon: %w", err)
	}
	defer func() {
		stopDaemonCtx()
		if daemonCmd.Process != nil {
			_ = daemonCmd.Process.Kill()
		}
		_ = daemonCmd.Wait()
	}()
	appendTimeline("daemon start socket=%s log=%s", socketPath, daemonLog)
	if err := waitForSocket(socketPath, v3TmuxDaemonReadyTimeout, func() error {
		client, dialErr := dialV3Client(socketPath)
		if dialErr != nil {
			return dialErr
		}
		return client.Close()
	}); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	appendTimeline("daemon ready")

	terminalIDPath := filepath.Join(artifactDir, "terminal.id")
	scriptPath := filepath.Join(artifactDir, "tmux-ansi-smoke.sh")
	terminalScript := "printf 'anytty-ansi-ready\\n'\n" +
		"printf '\\033[31mANSI16_RED\\033[0m \\033[1;34mANSI16_BLUE_BOLD\\033[0m\\n'\n" +
		"printf '\\033[38;5;202mANSI256_ORANGE\\033[0m \\033[48;5;24mANSI256_BG\\033[0m\\n'\n" +
		"printf '\\033[38;2;12;200;155mTRUECOLOR_MINT\\033[0m\\n'\n" +
		"printf 'CR_START\\rCR_REPLACED\\n'\n" +
		"printf '\\033[?1049hALT_SCREEN_MARK\\033[?1049lPRIMARY_AFTER_ALT\\n'\n" +
		"while IFS= read -r line; do printf 'anytty-ansi-echo:%s\\n' \"$line\"; done\n"
	script := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"export TERM=xterm-256color",
		"export ANYTTY=0",
		"export XDG_CONFIG_HOME=" + shellQuote(configHome),
		"export XDG_STATE_HOME=" + shellQuote(stateHome),
		"anytty_bin=" + shellQuote(anyttyBin),
		"socket=" + shellQuote(socketPath),
		"log_file=" + shellQuote(daemonLog),
		"terminal_id_path=" + shellQuote(terminalIDPath),
		"printf 'anytty tmux ansi harness ready\\n'",
		"id=\"$($anytty_bin --socket \"$socket\" --log-file \"$log_file\" v3 new --name tmux-ansi -- /bin/sh -c " + shellQuote(terminalScript) + ")\"",
		"printf '%s\\n' \"$id\" > \"$terminal_id_path\"",
		"printf 'anytty ansi terminal id:%s\\n' \"$id\"",
		"exec $anytty_bin --socket \"$socket\" --log-file \"$log_file\" v3 attach \"$id\"",
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	appendTimeline("tmux script=%s", scriptPath)

	cleanupTmux := func() {
		_ = runTmuxCommand(context.Background(), "kill-session", "-t", session)
	}
	defer cleanupTmux()
	if err := runTmuxCommand(ctx, "new-session", "-d", "-x", "100", "-y", "30", "-s", session, "/bin/sh", scriptPath); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	appendTimeline("tmux session=%s", session)
	terminalIDBytes, err := waitForFileContent(ctx, terminalIDPath, v3TmuxSurfaceReadyTimeout)
	if err != nil {
		return v3TmuxANSISmokeResult{}, fmt.Errorf("read terminal id artifact: %w", err)
	}
	terminalID := strings.TrimSpace(string(terminalIDBytes))
	appendTimeline("terminal created id=%s", terminalID)
	for _, marker := range []string{"anytty-ansi-ready", "ANSI16_RED", "ANSI256_ORANGE", "TRUECOLOR_MINT", "CR_REPLACED", "PRIMARY_AFTER_ALT"} {
		if err := waitForTmuxCapture(ctx, target, marker, v3TmuxSurfaceReadyTimeout); err != nil {
			return v3TmuxANSISmokeResult{}, err
		}
		appendTimeline("marker rendered=%s", marker)
	}

	ansi, err := captureTmuxPane(ctx, target, true)
	if err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	plain, err := captureTmuxPane(ctx, target, false)
	if err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	ansiPath := filepath.Join(artifactDir, "capture.ansi")
	plainPath := filepath.Join(artifactDir, "capture.txt")
	if err := os.WriteFile(ansiPath, []byte(ansi), 0o600); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	if err := os.WriteFile(plainPath, []byte(plain), 0o600); err != nil {
		return v3TmuxANSISmokeResult{}, err
	}
	appendTimeline("captured ansi=%s plain=%s", ansiPath, plainPath)

	_ = runAnyTTYCommand(context.Background(), anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "v3", "kill", terminalID)
	_ = runAnyTTYCommand(context.Background(), anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "v3", "rm", terminalID)
	appendTimeline("terminal cleanup id=%s", terminalID)
	removeArtifacts = false
	return v3TmuxANSISmokeResult{
		Session:      session,
		TerminalID:   terminalID,
		ArtifactDir:  artifactDir,
		ANSIPath:     ansiPath,
		PlainPath:    plainPath,
		DaemonLog:    daemonLog,
		SocketPath:   socketPath,
		TimelinePath: timelinePath,
		Captured:     plain,
		ANSICaptured: ansi,
	}, nil
}

func runV3TmuxEmojiDotsSmoke(ctx context.Context, anyttyBin string) (v3TmuxEmojiDotsSmokeResult, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, fmt.Errorf("tmux emoji dots smoke requires tmux in PATH: %w", err)
	}
	if anyttyBin == "" {
		return v3TmuxEmojiDotsSmokeResult{}, fmt.Errorf("tmux emoji dots smoke requires a anytty binary path")
	}
	artifactDir, err := os.MkdirTemp("", "anytty-v3-tmux-emoji-dots-*")
	if err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	configHome, stateHome, harnessEnv, err := v3TmuxHarnessEnvironment(artifactDir)
	if err != nil {
		_ = os.RemoveAll(artifactDir)
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	removeArtifacts := true
	defer func() {
		if removeArtifacts {
			_ = os.RemoveAll(artifactDir)
		}
	}()

	socketPath := filepath.Join(artifactDir, "core.sock")
	daemonLog := filepath.Join(artifactDir, "daemon.log")
	timelinePath := filepath.Join(artifactDir, "timeline.txt")
	session := fmt.Sprintf("anytty-v3-emoji-dots-%d", time.Now().UnixNano())
	target := session + ":0.0"
	appendTimeline := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...) + "\n"
		_ = appendFile(timelinePath, line)
	}

	daemonCtx, stopDaemonCtx := context.WithCancel(ctx)
	defer stopDaemonCtx()
	daemonCmd := exec.CommandContext(daemonCtx, anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "daemon")
	daemonCmd.Env = harnessEnv
	daemonCmd.Stdout = io.Discard
	daemonCmd.Stderr = io.Discard
	if err := daemonCmd.Start(); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, fmt.Errorf("start core-v2 daemon: %w", err)
	}
	defer func() {
		stopDaemonCtx()
		if daemonCmd.Process != nil {
			_ = daemonCmd.Process.Kill()
		}
		_ = daemonCmd.Wait()
	}()
	appendTimeline("daemon start socket=%s log=%s", socketPath, daemonLog)
	if err := waitForSocket(socketPath, v3TmuxDaemonReadyTimeout, func() error {
		client, dialErr := dialV3Client(socketPath)
		if dialErr != nil {
			return dialErr
		}
		return client.Close()
	}); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	appendTimeline("daemon ready")

	terminalIDPath := filepath.Join(artifactDir, "terminal.id")
	scriptPath := filepath.Join(artifactDir, "tmux-emoji-dots-smoke.sh")
	terminalScript := "printf 'anytty-pty-ready\\n'\n" +
		"while IFS= read -r line; do\n" +
		"  case \"$line\" in\n" +
		"    size-*) set -- $(stty size); printf 'anytty-pty-size:%s:%sx%s\\n' \"$line\" \"$2\" \"$1\" ;;\n" +
		"    *) printf 'anytty-pty-echo:%s\\n' \"$line\" ;;\n" +
		"  esac\n" +
		"done\n"
	script := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"export TERM=xterm-256color",
		"export ANYTTY=0",
		"export XDG_CONFIG_HOME=" + shellQuote(configHome),
		"export XDG_STATE_HOME=" + shellQuote(stateHome),
		"anytty_bin=" + shellQuote(anyttyBin),
		"socket=" + shellQuote(socketPath),
		"log_file=" + shellQuote(daemonLog),
		"terminal_id_path=" + shellQuote(terminalIDPath),
		"printf 'anytty tmux emoji dots harness ready\\n'",
		"id=\"$($anytty_bin --socket \"$socket\" --log-file \"$log_file\" v3 new --name tmux-emoji-dots -- /bin/sh -c " + shellQuote(terminalScript) + ")\"",
		"printf '%s\\n' \"$id\" > \"$terminal_id_path\"",
		"printf 'anytty emoji terminal id:%s\\n' \"$id\"",
		"exec $anytty_bin --socket \"$socket\" --log-file \"$log_file\" v3 attach \"$id\"",
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	appendTimeline("tmux script=%s", scriptPath)

	cleanupTmux := func() {
		_ = runTmuxCommand(context.Background(), "kill-session", "-t", session)
	}
	defer cleanupTmux()
	if err := runTmuxCommand(ctx, "new-session", "-d", "-x", "180", "-y", "32", "-s", session, "/bin/sh", scriptPath); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	appendTimeline("tmux session=%s initial=180x32", session)
	terminalIDBytes, err := waitForFileContent(ctx, terminalIDPath, v3TmuxSurfaceReadyTimeout)
	if err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, fmt.Errorf("read terminal id artifact: %w", err)
	}
	terminalID := strings.TrimSpace(string(terminalIDBytes))
	appendTimeline("terminal created id=%s", terminalID)
	if err := waitForTmuxCapture(ctx, target, "anytty-pty-ready", v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}

	// 中文说明：这里固定复现用户报告的真实链路：
	// 先 split 成左右 pane，再把右 pane attach 到同一个 terminal，
	// 回到左 pane 明确持有 owner，然后把左 owner 缩窄，让右 follower 展示 extent dots。
	for _, action := range []struct {
		keys  []string
		label string
		wait  time.Duration
	}{
		{keys: []string{"C-p"}, label: "enter pane mode", wait: 150 * time.Millisecond},
		{keys: []string{"%"}, label: "split right", wait: 800 * time.Millisecond},
		{keys: []string{"C-f"}, label: "open picker", wait: 300 * time.Millisecond},
	} {
		if err := runTmuxCommand(ctx, append([]string{"send-keys", "-t", target}, action.keys...)...); err != nil {
			return v3TmuxEmojiDotsSmokeResult{}, err
		}
		appendTimeline("send-keys %s keys=%q", action.label, action.keys)
		time.Sleep(action.wait)
	}
	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "-l", terminalID); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	appendTimeline("picker query terminal=%s", terminalID)
	time.Sleep(150 * time.Millisecond)
	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "Enter"); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	appendTimeline("attach follower by terminal query")
	time.Sleep(900 * time.Millisecond)
	for _, action := range []struct {
		keys  []string
		label string
		wait  time.Duration
	}{
		{keys: []string{"C-p"}, label: "enter pane mode again", wait: 150 * time.Millisecond},
		{keys: []string{"h"}, label: "focus left owner pane", wait: 150 * time.Millisecond},
		{keys: []string{"a"}, label: "reassert left owner", wait: 800 * time.Millisecond},
	} {
		if err := runTmuxCommand(ctx, append([]string{"send-keys", "-t", target}, action.keys...)...); err != nil {
			return v3TmuxEmojiDotsSmokeResult{}, err
		}
		appendTimeline("send-keys %s keys=%q", action.label, action.keys)
		time.Sleep(action.wait)
	}
	if err := waitForTmuxCapture(ctx, target, "owner", v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	if err := waitForTmuxCapture(ctx, target, "follow", v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	appendTimeline("owner and follower chrome visible")

	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "-l", "size-before"); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "Enter"); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	beforeMarker := "anytty-pty-size:size-before:"
	if err := waitForTmuxCapture(ctx, target, beforeMarker, v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	beforeCapture, _ := captureTmuxPane(ctx, target, false)
	beforeSize := lastMarkerSuffix(beforeCapture, beforeMarker)
	appendTimeline("before size=%s", beforeSize)

	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "C-r"); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	appendTimeline("enter resize mode")
	time.Sleep(150 * time.Millisecond)
	for step := 0; step < 16; step++ {
		if err := runTmuxCommand(ctx, "send-keys", "-t", target, "h"); err != nil {
			return v3TmuxEmojiDotsSmokeResult{}, err
		}
		appendTimeline("resize left owner narrower step=%d", step+1)
		time.Sleep(100 * time.Millisecond)
	}
	appendTimeline("wait for resize mode timeout")
	time.Sleep(1300 * time.Millisecond)
	_ = runTmuxCommand(ctx, "send-keys", "-t", target, "Escape")
	time.Sleep(150 * time.Millisecond)

	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "-l", "size-after"); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "Enter"); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	afterMarker := "anytty-pty-size:size-after:"
	if err := waitForTmuxCapture(ctx, target, afterMarker, v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	afterCapture, _ := captureTmuxPane(ctx, target, false)
	afterSize := lastMarkerSuffix(afterCapture, afterMarker)
	appendTimeline("after size=%s", afterSize)

	// 中文说明：连续 FE0F emoji 是这次回归的核心触发器；数量要足够长，
	// 才能稳定覆盖左/右 pane 同 terminal、宽 follower 已经展示 dots 的真实场景。
	emojiBurst := strings.TrimSpace(strings.Repeat("♻️ ", 24))
	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "-l", emojiBurst); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	appendTimeline("send emoji burst len=%d", utf8.RuneCountInString(emojiBurst))
	time.Sleep(200 * time.Millisecond)
	if err := runTmuxCommand(ctx, "send-keys", "-t", target, "Enter"); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	appendTimeline("submit emoji burst")
	if err := waitForTmuxCapture(ctx, target, "anytty-pty-echo:", v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}

	ansi, err := captureTmuxPane(ctx, target, true)
	if err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	plain, err := captureTmuxPane(ctx, target, false)
	if err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	ansiPath := filepath.Join(artifactDir, "capture.ansi")
	plainPath := filepath.Join(artifactDir, "capture.txt")
	if err := os.WriteFile(ansiPath, []byte(ansi), 0o600); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	if err := os.WriteFile(plainPath, []byte(plain), 0o600); err != nil {
		return v3TmuxEmojiDotsSmokeResult{}, err
	}
	appendTimeline("captured ansi=%s plain=%s", ansiPath, plainPath)

	dotsVisible := strings.Contains(plain, "····")
	if !dotsVisible {
		return v3TmuxEmojiDotsSmokeResult{}, fmt.Errorf("emoji dots smoke expected follower dots in capture, got %q", plain)
	}
	if !strings.Contains(plain, "anytty-pty-echo:") || !strings.Contains(plain, "follow") || !strings.Contains(plain, "owner") {
		return v3TmuxEmojiDotsSmokeResult{}, fmt.Errorf("emoji dots smoke missing owner/follower/echo markers in capture")
	}

	_ = runAnyTTYCommand(context.Background(), anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "v3", "kill", terminalID)
	_ = runAnyTTYCommand(context.Background(), anyttyBin, "--socket", socketPath, "--log-file", daemonLog, "v3", "rm", terminalID)
	appendTimeline("terminal cleanup id=%s", terminalID)
	removeArtifacts = false
	return v3TmuxEmojiDotsSmokeResult{
		Session:      session,
		TerminalID:   terminalID,
		ArtifactDir:  artifactDir,
		ANSIPath:     ansiPath,
		PlainPath:    plainPath,
		DaemonLog:    daemonLog,
		SocketPath:   socketPath,
		TimelinePath: timelinePath,
		Captured:     plain,
		BeforeSize:   beforeSize,
		AfterSize:    afterSize,
		DotsVisible:  dotsVisible,
	}, nil
}

func runV3TmuxVisualCompare(ctx context.Context, anyttyBin string) (v3TmuxVisualCompareResult, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return v3TmuxVisualCompareResult{}, fmt.Errorf("tmux visual compare requires tmux in PATH: %w", err)
	}
	if anyttyBin == "" {
		return v3TmuxVisualCompareResult{}, fmt.Errorf("tmux visual compare requires a anytty binary path")
	}
	artifactDir, err := os.MkdirTemp("", "anytty-v3-tmux-visual-*")
	if err != nil {
		return v3TmuxVisualCompareResult{}, err
	}
	removeArtifacts := true
	defer func() {
		if removeArtifacts {
			_ = os.RemoveAll(artifactDir)
		}
	}()

	session := fmt.Sprintf("anytty-v3-visual-%d", time.Now().UnixNano())
	target := session + ":0.0"
	scriptPath := filepath.Join(artifactDir, "tmux-visual-compare.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"export TERM=xterm-256color",
		"anytty_bin=" + shellQuote(anyttyBin),
		// 用 ANSI repaint 输出单帧，tmux 抓到的是屏幕状态，不是逐行日志。
		"$anytty_bin v3 visual-snapshot --ansi",
		"sleep 30",
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return v3TmuxVisualCompareResult{}, err
	}
	cleanupTmux := func() {
		_ = runTmuxCommand(context.Background(), "kill-session", "-t", session)
	}
	defer cleanupTmux()
	if err := runTmuxCommand(ctx, "new-session", "-d", "-x", "140", "-y", "40", "-s", session, "/bin/sh", scriptPath); err != nil {
		return v3TmuxVisualCompareResult{}, err
	}
	glyphs := render.DefaultPaneChromeGlyphs()
	if err := waitForTmuxCapture(ctx, target, "["+glyphs.Zoom+"]─["+glyphs.Close+"]", v3TmuxSurfaceReadyTimeout); err != nil {
		return v3TmuxVisualCompareResult{}, err
	}
	currentANSI, err := captureTmuxPane(ctx, target, true)
	if err != nil {
		return v3TmuxVisualCompareResult{}, err
	}
	currentPlain, err := captureTmuxPane(ctx, target, false)
	if err != nil {
		return v3TmuxVisualCompareResult{}, err
	}
	currentANSIPath := filepath.Join(artifactDir, "current.ansi")
	currentPlainPath := filepath.Join(artifactDir, "current.txt")
	targetPath := filepath.Join(artifactDir, "target.txt")
	diffPath := filepath.Join(artifactDir, "diff.txt")
	stylePath := filepath.Join(artifactDir, "style.txt")
	styleDiffPath := filepath.Join(artifactDir, "style.diff.txt")
	currentStyleMapPath := filepath.Join(artifactDir, "current.stylemap.txt")
	targetStyleMapPath := filepath.Join(artifactDir, "target.stylemap.txt")
	styleMapDiffPath := filepath.Join(artifactDir, "stylemap.diff.txt")
	summaryPath := filepath.Join(artifactDir, "summary.txt")
	targetPlain := v3VisualTargetPlain()
	diffText, mismatches := diffVisualPlain(targetPlain, currentPlain, 140, 40)
	styleText, styleDiffText, styleMismatches := visualANSIStyleReport(currentANSI)
	currentStyleMap := visualANSIStyleMap(currentANSI, 140, 40)
	targetStyleMap := visualANSIStyleMap(currentANSI, 140, 40)
	styleMapDiffText, styleMapMismatches := diffVisualStyleMap(targetStyleMap, currentStyleMap)
	currentStyleMapText := formatVisualStyleMap("current tmux ANSI style map", currentStyleMap)
	targetStyleMapText := formatVisualStyleMap("target semantic style map", targetStyleMap)
	summary := strings.Join([]string{
		"anytty v3 tmux visual compare",
		"source: tuiv2 single-line bar + object chrome contract",
		fmt.Sprintf("viewport: %dx%d", 140, 40),
		fmt.Sprintf("mismatches: %d", mismatches),
		fmt.Sprintf("style_mismatches: %d", styleMismatches),
		fmt.Sprintf("stylemap_mismatches: %d", styleMapMismatches),
		"current_plain: " + currentPlainPath,
		"current_ansi: " + currentANSIPath,
		"target: " + targetPath,
		"diff: " + diffPath,
		"style: " + stylePath,
		"style_diff: " + styleDiffPath,
		"current_stylemap: " + currentStyleMapPath,
		"target_stylemap: " + targetStyleMapPath,
		"stylemap_diff: " + styleMapDiffPath,
		"",
	}, "\n")
	for _, file := range []struct {
		path string
		body string
	}{
		{currentANSIPath, currentANSI},
		{currentPlainPath, currentPlain},
		{targetPath, targetPlain},
		{diffPath, diffText},
		{stylePath, styleText},
		{styleDiffPath, styleDiffText},
		{currentStyleMapPath, currentStyleMapText},
		{targetStyleMapPath, targetStyleMapText},
		{styleMapDiffPath, styleMapDiffText},
		{summaryPath, summary},
	} {
		if err := os.WriteFile(file.path, []byte(file.body), 0o600); err != nil {
			return v3TmuxVisualCompareResult{}, err
		}
	}
	removeArtifacts = false
	return v3TmuxVisualCompareResult{
		Session:             session,
		ArtifactDir:         artifactDir,
		CurrentANSIPath:     currentANSIPath,
		CurrentPlainPath:    currentPlainPath,
		TargetPath:          targetPath,
		DiffPath:            diffPath,
		StylePath:           stylePath,
		StyleDiffPath:       styleDiffPath,
		CurrentStyleMapPath: currentStyleMapPath,
		TargetStyleMapPath:  targetStyleMapPath,
		StyleMapDiffPath:    styleMapDiffPath,
		SummaryPath:         summaryPath,
		Mismatches:          mismatches,
		StyleMismatches:     styleMismatches,
		StyleMapMismatches:  styleMapMismatches,
	}, nil
}

func v3VisualTargetPlain() string {
	lines := []string{
		" WS main ▎ 1 main ×   2 logs ×  + ",
		"┌─[□] shell ────────────────────────────────────────────────────────── ●  x1 ◆ owner ─[↗]─[│]─[─]─[×]──┐─[□] logs ───── ●  x1 ◇ follow─[×]─┐",
		"│anytty git:core-tui-v3-migration  go v1.26.0                                      ····················│ visual review baseline       ·····│",
		"│> make test                                                                       ····················│ target visual mismatch       ·····│",
		"│ok   tui/render                                                                   ····················│ emoji 🚀 and 中文            ·····│",
		"│>                                                                                 ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│ ┌───────────[◎]─[▾]─[↗]─[×]─┐·····│",
		"│                                                                                  ····················│ │        unconnected        │·····│",
		"│                                                                                  ····················│ │                           │·····│",
		"│                                                                                  ····················│ │      Attach existing      │·····│",
		"│                                                                                  ····················│ │        New terminal       │·····│",
		"│                                                                                  ····················│ │      Terminal Manager     │·····│",
		"│                                                                                  ····················│ │           Close           │·····│",
		"│                                                                                  ····················│ └───────────────────────────┘·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│                                                                                  ····················│                              ·····│",
		"│······································································································│···································│",
		"│······································································································│···································│",
		"└──────────────────────────────────────────────────────────────────────────────────────────────────────┘───────────────────────────────────┘",
		"[Ctrl+P] PANE • [Ctrl+R] RESIZE • [Ctrl+G] GLOBAL • [Ctrl+O] FLOAT • [Ctrl+T] TAB • [PgUp] COPY                 ws:main float:1 terminals:1",
	}
	return normalizeVisualText(strings.Join(lines, "\n"), 140, 40)
}

func diffVisualPlain(target string, current string, width int, height int) (string, int) {
	targetLines := normalizeVisualLines(target, width, height)
	currentLines := normalizeVisualLines(current, width, height)
	var builder strings.Builder
	builder.WriteString("tmux visual diff: target vs current\n")
	builder.WriteString("source target: tuiv2 single-line bar + object chrome contract\n\n")
	mismatches := 0
	for i := 0; i < height; i++ {
		if targetLines[i] == currentLines[i] {
			continue
		}
		mismatches++
		fmt.Fprintf(&builder, "@@ row %02d @@\n", i+1)
		fmt.Fprintf(&builder, "- %s\n", targetLines[i])
		fmt.Fprintf(&builder, "+ %s\n", currentLines[i])
	}
	if mismatches == 0 {
		builder.WriteString("no row mismatches\n")
	}
	return builder.String(), mismatches
}

func normalizeVisualText(text string, width int, height int) string {
	return strings.Join(normalizeVisualLines(text, width, height), "\n") + "\n"
}

func normalizeVisualLines(text string, width int, height int) []string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	out := make([]string, height)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			out[i] = fitVisualLine(lines[i], width)
			continue
		}
		out[i] = strings.Repeat(" ", width)
	}
	return out
}

func fitVisualLine(line string, width int) string {
	line = strings.TrimRight(line, "\r")
	displayWidth := render.DisplayWidth(line)
	if displayWidth > width {
		return render.TruncateCells(line, width)
	}
	if displayWidth < width {
		return line + strings.Repeat(" ", width-displayWidth)
	}
	return line
}

type visualStyleExpectation struct {
	Name      string
	Row       int
	Col       int
	Glyph     string
	MustHave  []string
	MustAvoid []string
}

type visualANSICell struct {
	Text string
	SGR  []string
}

type visualStyleClass struct {
	Code string
	Name string
}

var (
	visualStylePlain       = visualStyleClass{Code: "P", Name: "plain"}
	visualStyleStatus      = visualStyleClass{Code: "S", Name: "status"}
	visualStyleAccent      = visualStyleClass{Code: "A", Name: "accent"}
	visualStyleMuted       = visualStyleClass{Code: "M", Name: "muted"}
	visualStyleWarn        = visualStyleClass{Code: "W", Name: "warning"}
	visualStyleSuccess     = visualStyleClass{Code: "G", Name: "success"}
	visualStyleTransparent = visualStyleClass{Code: ".", Name: "transparent"}
	visualStyleUnknown     = visualStyleClass{Code: "?", Name: "unknown"}
)

func visualANSIStyleReport(ansiText string) (string, string, int) {
	lines := strings.Split(strings.TrimRight(ansiText, "\n"), "\n")
	grid := visualANSICellGrid(lines)
	expectations := visualStyleExpectations()
	var report strings.Builder
	var diff strings.Builder
	report.WriteString("tmux visual ANSI style contract\n")
	report.WriteString("source target: semantic SGR probes for visual-audit-current\n\n")
	diff.WriteString("tmux visual ANSI style diff\n")
	diff.WriteString("source target: semantic SGR probes for visual-audit-current\n\n")
	mismatches := 0
	for _, expectation := range expectations {
		cell := visualCellAt(grid, expectation.Row, expectation.Col)
		ok := true
		reasons := []string{}
		if expectation.Glyph != "" && cell.Text != expectation.Glyph {
			ok = false
			reasons = append(reasons, fmt.Sprintf("glyph got %q want %q", cell.Text, expectation.Glyph))
		}
		for _, required := range expectation.MustHave {
			if !visualSGRContains(cell.SGR, required) {
				ok = false
				reasons = append(reasons, "missing "+required)
			}
		}
		for _, forbidden := range expectation.MustAvoid {
			if visualSGRContains(cell.SGR, forbidden) {
				ok = false
				reasons = append(reasons, "unexpected "+forbidden)
			}
		}
		status := "ok"
		if !ok {
			status = "mismatch"
			mismatches++
			fmt.Fprintf(&diff, "@@ %s row=%d col=%d @@\n", expectation.Name, expectation.Row, expectation.Col)
			fmt.Fprintf(&diff, "glyph=%q sgr=%s\n", cell.Text, strings.Join(cell.SGR, " "))
			fmt.Fprintf(&diff, "reason=%s\n", strings.Join(reasons, "; "))
		}
		fmt.Fprintf(&report, "%s row=%d col=%d glyph=%q status=%s sgr=%s\n", expectation.Name, expectation.Row, expectation.Col, cell.Text, status, strings.Join(cell.SGR, " "))
	}
	if mismatches == 0 {
		diff.WriteString("no style mismatches\n")
	}
	return report.String(), diff.String(), mismatches
}

func visualANSIStyleMap(ansiText string, width int, height int) [][]visualStyleClass {
	lines := strings.Split(strings.TrimRight(ansiText, "\n"), "\n")
	grid := visualANSICellGrid(lines)
	styleMap := make([][]visualStyleClass, height)
	for row := 0; row < height; row++ {
		styleMap[row] = make([]visualStyleClass, width)
		for col := 0; col < width; col++ {
			cell := visualCellAt(grid, row+1, col+1)
			styleMap[row][col] = visualClassFromANSICell(cell)
		}
	}
	return styleMap
}

func visualClassFromANSICell(cell visualANSICell) visualStyleClass {
	if len(cell.SGR) == 0 {
		if strings.TrimSpace(cell.Text) != "" {
			return visualStylePlain
		}
		return visualStyleTransparent
	}
	return visualClassFromSGR(cell.SGR)
}

func visualClassFromSGR(sgr []string) visualStyleClass {
	switch {
	case visualSGRContains(sgr, "38;2;169;112;255"):
		return visualStyleAccent
	case visualSGRContains(sgr, "38;2;240;196;92"):
		return visualStyleWarn
	case visualSGRContains(sgr, "38;2;138;223;122"):
		return visualStyleSuccess
	case visualSGRContains(sgr, "38;2;119;113;127") || visualSGRContains(sgr, "38;2;130;113;155") ||
		visualSGRContains(sgr, "38;2;171;111;119"):
		return visualStyleMuted
	case visualSGRContains(sgr, "38;2;255;107;107") || visualSGRContains(sgr, "38;2;194;110;116"):
		return visualStyleWarn
	case visualSGRContains(sgr, "38;2;231;226;239") || visualSGRContains(sgr, "38;2;222;219;230") ||
		visualSGRContains(sgr, "38;2;8;8;13") || visualSGRContains(sgr, "48;2;8;8;13") ||
		visualSGRContains(sgr, "48;2;231;226;239") || visualSGRContains(sgr, "48;2;42;34;59") ||
		visualSGRContains(sgr, "48;2;60;46;85"):
		return visualStyleStatus
	case visualSGRContains(sgr, "38;2;122;184;255"):
		return visualStyleStatus
	case visualSGRContains(sgr, "38;2;250;138;102") || visualSGRContains(sgr, "38;2;194;141;198") ||
		visualSGRContains(sgr, "38;2;148;144;255") || visualSGRContains(sgr, "38;2;132;209;169"):
		return visualStyleUnknown
	case len(sgr) == 0:
		return visualStyleTransparent
	default:
		return visualStyleUnknown
	}
}

func formatVisualStyleMap(title string, styleMap [][]visualStyleClass) string {
	var builder strings.Builder
	builder.WriteString(title)
	builder.WriteString("\nlegend: S=status A=accent M=muted W=warning G=success P=plain .=transparent ?=unknown\n\n")
	for row, line := range styleMap {
		fmt.Fprintf(&builder, "%02d ", row+1)
		for _, class := range line {
			builder.WriteString(class.Code)
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func diffVisualStyleMap(target [][]visualStyleClass, current [][]visualStyleClass) (string, int) {
	var builder strings.Builder
	builder.WriteString("tmux visual style map diff\n")
	builder.WriteString("source target: fixed 140x40 semantic style regions\n\n")
	mismatches := 0
	height := len(target)
	for row := 0; row < height; row++ {
		targetLine := target[row]
		currentLine := []visualStyleClass{}
		if row < len(current) {
			currentLine = current[row]
		}
		if visualStyleLineCodes(targetLine) == visualStyleLineCodes(currentLine) {
			continue
		}
		mismatches++
		fmt.Fprintf(&builder, "@@ row %02d @@\n", row+1)
		fmt.Fprintf(&builder, "- %s\n", visualStyleLineCodes(targetLine))
		fmt.Fprintf(&builder, "+ %s\n", visualStyleLineCodes(currentLine))
		fmt.Fprintf(&builder, "  %s\n", visualStyleMismatchColumns(targetLine, currentLine))
	}
	if mismatches == 0 {
		builder.WriteString("no style map mismatches\n")
	}
	return builder.String(), mismatches
}

func visualStyleLineCodes(line []visualStyleClass) string {
	var builder strings.Builder
	for _, class := range line {
		if class.Code == "" {
			builder.WriteString(visualStyleUnknown.Code)
			continue
		}
		builder.WriteString(class.Code)
	}
	return builder.String()
}

func visualStyleMismatchColumns(target []visualStyleClass, current []visualStyleClass) string {
	limit := len(target)
	if len(current) > limit {
		limit = len(current)
	}
	cols := []string{}
	for i := 0; i < limit; i++ {
		var targetCode, currentCode string
		if i < len(target) {
			targetCode = target[i].Code
		}
		if i < len(current) {
			currentCode = current[i].Code
		}
		if targetCode != currentCode {
			cols = append(cols, fmt.Sprintf("%d:%s/%s", i+1, targetCode, currentCode))
		}
	}
	if len(cols) > 16 {
		cols = append(cols[:16], fmt.Sprintf("...(+%d)", len(cols)-16))
	}
	return "cols " + strings.Join(cols, " ")
}

func visualStyleExpectations() []visualStyleExpectation {
	return []visualStyleExpectation{
		{Name: "header-workspace-bg", Row: 1, Col: 1, Glyph: " ", MustHave: []string{"1", "38;2;212;192;244", "48;2;60;46;85"}},
		{Name: "active-tab-marker", Row: 1, Col: 10, Glyph: "▎", MustHave: []string{"1", "38;2;169;112;255", "48;2;42;34;59"}},
		{Name: "inactive-tab-muted", Row: 1, Col: 23, Glyph: "2", MustHave: []string{"38;2;181;163;209", "48;2;8;8;13"}, MustAvoid: []string{"2;38;2;119;113;127"}},
		{Name: "pane-action-accent", Row: 2, Col: 88, Glyph: "↗", MustHave: []string{"1", "38;2;169;112;255"}},
		{Name: "active-pane-right-border-accent", Row: 2, Col: 104, Glyph: "┐", MustHave: []string{"1", "38;2;169;112;255"}},
		{Name: "active-pane-content-border-accent", Row: 3, Col: 104, Glyph: "│", MustHave: []string{"1", "38;2;169;112;255"}},
		{Name: "floating-border-accent", Row: 8, Col: 106, Glyph: "┌", MustHave: []string{"1", "38;2;169;112;255"}},
		{Name: "floating-inner-accent", Row: 10, Col: 106, Glyph: "│", MustHave: []string{"1", "38;2;169;112;255"}},
		{Name: "right-pane-border-muted", Row: 10, Col: 140, Glyph: "│", MustHave: []string{"2", "38;2;184;177;196"}, MustAvoid: []string{"38;2;169;112;255"}},
		{Name: "footer-no-bg", Row: 40, Col: 1, Glyph: "[", MustHave: []string{"38;2;169;112;255"}, MustAvoid: []string{"48;2;8;8;13"}},
		{Name: "footer-float-accent", Row: 40, Col: 121, Glyph: "f", MustHave: []string{"1", "38;2;169;112;255"}, MustAvoid: []string{"48;2;8;8;13"}},
	}
}

func visualANSICellGrid(lines []string) [][]visualANSICell {
	grid := make([][]visualANSICell, len(lines))
	for row, line := range lines {
		grid[row] = visualANSILineCells(line)
	}
	return grid
}

func visualANSILineCells(line string) []visualANSICell {
	out := []visualANSICell{}
	activeSGR := []string{}
	for i := 0; i < len(line); {
		if line[i] == '\x1b' {
			params, end, ok := parseSGRSequence(line, i)
			if ok {
				activeSGR = updateActiveSGR(activeSGR, params)
				i = end
				continue
			}
		}
		r, size := nextRune(line, i)
		text := string(r)
		width := render.DisplayWidth(text)
		if width <= 0 {
			width = 1
		}
		out = append(out, visualANSICell{Text: text, SGR: append([]string(nil), activeSGR...)})
		for col := 1; col < width; col++ {
			out = append(out, visualANSICell{Text: "", SGR: append([]string(nil), activeSGR...)})
		}
		i += size
	}
	return out
}

func parseSGRSequence(value string, start int) ([]string, int, bool) {
	if start+2 >= len(value) || value[start] != '\x1b' || value[start+1] != '[' {
		return nil, start, false
	}
	end := start + 2
	for end < len(value) && value[end] != 'm' {
		end++
	}
	if end >= len(value) {
		return nil, start, false
	}
	raw := value[start+2 : end]
	if raw == "" {
		raw = "0"
	}
	return strings.Split(raw, ";"), end + 1, true
}

func updateActiveSGR(active []string, params []string) []string {
	next := append([]string(nil), active...)
	for i := 0; i < len(params); i++ {
		param := params[i]
		switch param {
		case "", "0":
			next = nil
		case "1", "2":
			next = visualReplaceSGRPrefix(next, param, 1)
		case "22":
			next = visualRemoveSGR(next, "1", "2")
		case "38", "48":
			if i+4 < len(params) && params[i+1] == "2" {
				sequence := strings.Join(params[i:i+5], ";")
				next = visualReplaceSGRPrefix(next, param+";2", 5)
				next = append(next, sequence)
				i += 4
			} else {
				next = append(next, param)
			}
		case "39":
			next = visualRemoveSGRPrefix(next, "38;")
		case "49":
			next = visualRemoveSGRPrefix(next, "48;")
		default:
			next = append(next, param)
		}
	}
	return next
}

func visualReplaceSGRPrefix(values []string, prefix string, width int) []string {
	_ = width
	values = visualRemoveSGRPrefix(values, prefix)
	return append(values, prefix)
}

func visualRemoveSGR(values []string, targets ...string) []string {
	out := values[:0]
	for _, value := range values {
		remove := false
		for _, target := range targets {
			if value == target {
				remove = true
				break
			}
		}
		if !remove {
			out = append(out, value)
		}
	}
	return out
}

func visualRemoveSGRPrefix(values []string, prefix string) []string {
	out := values[:0]
	for _, value := range values {
		if !strings.HasPrefix(value, prefix) {
			out = append(out, value)
		}
	}
	return out
}

func nextRune(value string, start int) (rune, int) {
	r, size := utf8.DecodeRuneInString(value[start:])
	if r == utf8.RuneError && size == 0 {
		return rune(value[start]), 1
	}
	return r, size
}

func visualCellAt(grid [][]visualANSICell, row int, col int) visualANSICell {
	row--
	col--
	if row < 0 || row >= len(grid) || col < 0 || col >= len(grid[row]) {
		return visualANSICell{}
	}
	return grid[row][col]
}

func visualSGRContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected || strings.Contains(value, expected) {
			return true
		}
	}
	return false
}

func v3TmuxHarnessEnvironment(artifactDir string) (string, string, []string, error) {
	configHome := filepath.Join(artifactDir, "xdg-config")
	stateHome := filepath.Join(artifactDir, "xdg-state")
	for _, path := range []string{configHome, stateHome} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return "", "", nil, fmt.Errorf("create tmux harness XDG directory %q: %w", path, err)
		}
	}
	environment := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "XDG_CONFIG_HOME=") || strings.HasPrefix(entry, "XDG_STATE_HOME=") {
			continue
		}
		environment = append(environment, entry)
	}
	environment = append(environment, "XDG_CONFIG_HOME="+configHome, "XDG_STATE_HOME="+stateHome)
	return configHome, stateHome, environment, nil
}

func runV3TmuxStabilitySmoke(ctx context.Context, anyttyBin string, rounds int) (v3TmuxStabilitySmokeResult, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return v3TmuxStabilitySmokeResult{}, fmt.Errorf("tmux stability smoke requires tmux in PATH: %w", err)
	}
	if anyttyBin == "" {
		return v3TmuxStabilitySmokeResult{}, fmt.Errorf("tmux stability smoke requires a anytty binary path")
	}
	if rounds <= 0 {
		rounds = 1
	}
	artifactDir, err := os.MkdirTemp("", "anytty-v3-tmux-stability-*")
	if err != nil {
		return v3TmuxStabilitySmokeResult{}, err
	}
	removeArtifacts := true
	defer func() {
		if removeArtifacts {
			_ = os.RemoveAll(artifactDir)
		}
	}()
	timelinePath := filepath.Join(artifactDir, "timeline.txt")
	appendTimeline := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...) + "\n"
		_ = appendFile(timelinePath, line)
	}
	var artifacts []string
	for round := 1; round <= rounds; round++ {
		appendTimeline("round %d terminal smoke start", round)
		terminal, err := runV3TmuxTerminalSmoke(ctx, anyttyBin)
		if err != nil {
			return v3TmuxStabilitySmokeResult{}, fmt.Errorf("round %d terminal smoke: %w", round, err)
		}
		artifacts = append(artifacts, terminal.ArtifactDir)
		appendTimeline("round %d terminal smoke ok artifact=%s", round, terminal.ArtifactDir)

		appendTimeline("round %d resize smoke start", round)
		resize, err := runV3TmuxResizeSmoke(ctx, anyttyBin)
		if err != nil {
			return v3TmuxStabilitySmokeResult{}, fmt.Errorf("round %d resize smoke: %w", round, err)
		}
		artifacts = append(artifacts, resize.ArtifactDir)
		appendTimeline("round %d resize smoke ok artifact=%s before=%s after=%s", round, resize.ArtifactDir, resize.BeforeSize, resize.AfterSize)

		appendTimeline("round %d ansi smoke start", round)
		ansi, err := runV3TmuxANSISmoke(ctx, anyttyBin)
		if err != nil {
			return v3TmuxStabilitySmokeResult{}, fmt.Errorf("round %d ansi smoke: %w", round, err)
		}
		artifacts = append(artifacts, ansi.ArtifactDir)
		appendTimeline("round %d ansi smoke ok artifact=%s", round, ansi.ArtifactDir)
	}
	manifest := strings.Join(append([]string{"tmux stability artifacts:"}, artifacts...), "\n") + "\n"
	if err := os.WriteFile(filepath.Join(artifactDir, "artifacts.txt"), []byte(manifest), 0o600); err != nil {
		return v3TmuxStabilitySmokeResult{}, err
	}
	removeArtifacts = false
	return v3TmuxStabilitySmokeResult{
		ArtifactDir:  artifactDir,
		TimelinePath: timelinePath,
		Rounds:       rounds,
		Artifacts:    artifacts,
	}, nil
}

func runTmuxCommand(ctx context.Context, args ...string) error {
	output, err := exec.CommandContext(ctx, "tmux", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runAnyTTYCommand(ctx context.Context, anyttyBin string, args ...string) error {
	output, err := exec.CommandContext(ctx, anyttyBin, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("anytty %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func captureTmuxPane(ctx context.Context, target string, ansi bool) (string, error) {
	args := []string{"capture-pane", "-t", target, "-p"}
	if ansi {
		args = []string{"capture-pane", "-t", target, "-e", "-p"}
	}
	output, err := exec.CommandContext(ctx, "tmux", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tmux %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func waitForTmuxCapture(ctx context.Context, target string, marker string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		captured, err := captureTmuxPane(ctx, target, false)
		if err == nil {
			last = captured
			if strings.Contains(captured, marker) {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return fmt.Errorf("timed out waiting for tmux marker %q; last capture:\n%s", marker, last)
}

func waitForFileContent(ctx context.Context, path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil && len(strings.TrimSpace(string(data))) > 0 {
			return data, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return nil, fmt.Errorf("timed out waiting for file %s: %w", path, lastErr)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func tmuxDaemonReadyError(startupErr error, stdoutFile *os.File, stderrFile *os.File) error {
	readOutput := func(file *os.File) string {
		if file == nil {
			return ""
		}
		_ = file.Sync()
		data, err := os.ReadFile(file.Name())
		if err != nil {
			return "<read failed: " + err.Error() + ">"
		}
		return strings.TrimSpace(string(data))
	}
	// 冷启动超时必须保留子进程证据，不得把 daemon 退出伪装成单纯 socket 超时。
	return fmt.Errorf(
		"%w; daemon stdout=%q stderr=%q",
		startupErr,
		readOutput(stdoutFile),
		readOutput(stderrFile),
	)
}

func appendFile(path string, text string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.WriteString(file, text)
	return err
}

func lastMarkerSuffix(text string, marker string) string {
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if index := strings.Index(lines[i], marker); index >= 0 {
			fields := strings.Fields(lines[i][index+len(marker):])
			if len(fields) > 0 {
				return fields[0]
			}
			return ""
		}
	}
	return ""
}
