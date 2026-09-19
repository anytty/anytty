//go:build darwin || linux

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func poolLifecycleSupported() bool { return true }

func poolProcessIdentity(pid int) (string, error) {
	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "lstart=", "-o", "command=").Output()
	if err != nil {
		return "", err
	}
	identity := strings.TrimSpace(string(output))
	if identity == "" {
		return "", fmt.Errorf("process %d is not running", pid)
	}
	return identity, nil
}

func stopPoolProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(syscall.SIGTERM)
}

func startDetachedPool(socketPath, logPath, configPath string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{"--socket", socketPath, "--log-file", logPath}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	args = append(args, "pool", "run")
	command := exec.Command(executable, args...)
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()
	output, err := openPrivatePoolLog(logPath)
	if err != nil {
		return err
	}
	defer output.Close()
	command.Stdin, command.Stdout, command.Stderr = devNull, output, output
	configureDetachedCommand(command)
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}

// startDetachedAccess 启动 access 子进程并返回其 PID；日志写入 accessLogPath。
func startDetachedAccess(socketPath, accessLogPath string) (int, error) {
	executable, err := os.Executable()
	if err != nil {
		return 0, err
	}
	command := exec.Command(executable, accessRunArgs(socketPath, accessLogPath)...)
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, err
	}
	defer devNull.Close()
	output, err := openPrivatePoolLog(accessLogPath)
	if err != nil {
		return 0, err
	}
	defer output.Close()
	command.Stdin, command.Stdout, command.Stderr = devNull, output, output
	command.Env = os.Environ()
	configureDetachedCommand(command)
	if err := command.Start(); err != nil {
		return 0, err
	}
	pid := command.Process.Pid
	return pid, command.Process.Release()
}
