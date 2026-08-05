//go:build darwin || linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func daemonLifecycleSupported() bool { return true }

func daemonProcessIdentity(pid int) (string, error) {
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

func stopDaemonProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(syscall.SIGTERM)
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
