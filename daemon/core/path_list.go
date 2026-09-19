package core

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const defaultPathListDirsLimit = 100

func listPathDirectories(prefix string, limit int) (PathDirectories, error) {
	if strings.TrimSpace(prefix) == "" {
		return PathDirectories{}, nil
	}
	baseDisplay, baseResolved, fragment, ok := pathCompletionBase(prefix)
	if !ok {
		return PathDirectories{}, nil
	}
	out := PathDirectories{BasePath: baseResolved}
	entries, err := os.ReadDir(baseResolved)
	if err != nil {
		// 中文说明：目录补全的 domain owner 是当前 daemon 机器文件系统；
		// base 不存在或无权限是 prompt 空态，不是 transport/endpoint lifecycle 失败。
		out.Missing = true
		return out, nil
	}
	if limit <= 0 {
		limit = defaultPathListDirsLimit
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
	fragmentLower := strings.ToLower(fragment)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(fragment, ".") {
			continue
		}
		if fragmentLower != "" && !strings.HasPrefix(strings.ToLower(name), fragmentLower) {
			continue
		}
		if len(out.Entries) >= limit {
			out.Truncated = true
			break
		}
		candidate := name + string(filepath.Separator)
		if baseDisplay != "" {
			candidate = baseDisplay + candidate
		}
		out.Entries = append(out.Entries, PathDirectoryEntry{Name: name, Path: candidate})
	}
	return out, nil
}

func pathDefaults() TerminalDefaults {
	return TerminalDefaults{
		DefaultCommand: defaultPathCommand(),
		DefaultCWD:     defaultPathCWD(),
		Platform:       runtime.GOOS,
	}
}

func defaultPathCommand() []string {
	shell := strings.TrimSpace(os.Getenv("SHELL"))
	if shell != "" {
		return []string{shell}
	}
	if shell = strings.TrimSpace(currentAccountShell()); shell != "" {
		return []string{shell}
	}
	// 中文说明：默认 shell 的 truth 在 daemon 进程所在机器；账号信息也不可用时
	// 才退回该机器上的最小 POSIX shell，不读取 TUI/client 进程环境。
	return []string{"/bin/sh"}
}

func defaultPathCWD() string {
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		return strings.TrimSpace(home)
	}
	cwd, err := os.Getwd()
	if err == nil {
		return strings.TrimSpace(cwd)
	}
	return ""
}

func pathCompletionBase(prefix string) (string, string, string, bool) {
	home, _ := os.UserHomeDir()
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", "", false
	}
	switch {
	case prefix == "~":
		if strings.TrimSpace(home) == "" {
			return "", "", "", false
		}
		return "~" + string(filepath.Separator), home, "", true
	case strings.HasPrefix(prefix, "~/") || strings.HasPrefix(prefix, "~\\"):
		if strings.TrimSpace(home) == "" {
			return "", "", "", false
		}
		rest := filepath.FromSlash(prefix[2:])
		base, fragment := filepath.Split(rest)
		return "~" + string(filepath.Separator) + base, filepath.Join(home, base), fragment, true
	}
	base, fragment := filepath.Split(filepath.FromSlash(prefix))
	if filepath.IsAbs(filepath.FromSlash(prefix)) {
		return base, filepath.Clean(base), fragment, true
	}
	return base, filepath.Join(cwd, base), fragment, true
}
