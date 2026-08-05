//go:build windows

package main

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func configureDetachedCommand(command *exec.Cmd) {
	// 中文说明：Windows current-user daemon 必须脱离调用方控制台，并拥有独立进程组，避免关闭 CLI 时被连带终止。
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
		HideWindow:    true,
	}
}
