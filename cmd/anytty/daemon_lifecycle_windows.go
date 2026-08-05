//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows"
)

func daemonLifecycleSupported() bool { return true }

func daemonProcessIdentity(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("process %d is invalid", pid)
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil {
		return "", err
	}
	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil {
		return "", err
	}
	image := strings.ToLower(windows.UTF16ToString(buffer[:size]))
	return fmt.Sprintf("%08x%08x:%s", creation.HighDateTime, creation.LowDateTime, image), nil
}

func stopDaemonProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	// 中文说明：PID 已由创建时间与映像路径复验；Windows 无 SIGTERM，当前 lifecycle contract 使用精确进程终止。
	return process.Kill()
}

func startDetachedDaemon(socketPath, logPath, configPath string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{"--socket", socketPath, "--log-file", logPath}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	args = append(args, "daemon", "run")
	command := exec.Command(executable, args...)
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()
	output, err := openPrivateDaemonLog(logPath)
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
