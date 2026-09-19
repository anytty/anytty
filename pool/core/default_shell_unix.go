//go:build !windows

package core

import (
	"bufio"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
)

func currentAccountShell() string {
	current, err := user.Current()
	if err != nil {
		return ""
	}
	if shell := shellFromPasswd(current.Uid); shell != "" {
		return shell
	}
	if runtime.GOOS != "darwin" || strings.TrimSpace(current.Username) == "" {
		return ""
	}
	out, err := exec.Command("dscl", ".", "-read", "/Users/"+current.Username, "UserShell").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(out)), "UserShell:"))
}

func shellFromPasswd(uid string) string {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) == 7 && fields[2] == uid {
			return strings.TrimSpace(fields[6])
		}
	}
	return ""
}
