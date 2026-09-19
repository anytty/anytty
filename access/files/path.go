// Package files 是 access 本地文件服务。Phase 2 起 file.* 命令在 access 终结，
// 操作 access 主机文件系统。
//
// 本文件实现 §11.4 的统一路径解析：绝对/相对/`~`/`.`/`..` 归一化、根目录
// 约束与符号链接逃逸防护。列表、预览、上传、下载、改名、删除必须共用同一个
// Resolver，避免每个操作各自解释路径。
package files

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	// ErrPathInvalid 表示路径为空、无法归一化或既不是绝对路径也没有 BaseDir。
	ErrPathInvalid = errors.New("file path is invalid")
	// ErrPathOutsideRoots 表示归一化后的路径逃逸出允许的 file roots。
	ErrPathOutsideRoots = errors.New("file path escapes configured roots")
)

// ResolverConfig 配置路径解析策略。
type ResolverConfig struct {
	// Roots 是允许访问的根目录集合；空集合表示不限制（legacy 行为），
	// 只做归一化，不做符号链接逃逸检查。
	Roots []string
	// BaseDir 是相对路径的解析基准；空表示拒绝相对路径。
	BaseDir string
	// Home 是 `~` 展开目标；空时使用 os.UserHomeDir。
	Home string
	// CaseInsensitive 在大小写不敏感平台（Windows/macOS）上做根目录比较；
	// 默认按 GOOS 推断。
	CaseInsensitive *bool
}

// Resolver 把客户端路径解析为 root-safe 的绝对路径。
type Resolver struct {
	roots           []string
	baseDir         string
	home            string
	caseInsensitive bool
}

// NewResolver 创建路径解析器。root 自身也会做符号链接解析，保证比较基准一致。
func NewResolver(config ResolverConfig) (*Resolver, error) {
	resolver := &Resolver{
		baseDir:         strings.TrimSpace(config.BaseDir),
		home:            strings.TrimSpace(config.Home),
		caseInsensitive: caseInsensitivePlatform(),
	}
	if config.CaseInsensitive != nil {
		resolver.caseInsensitive = *config.CaseInsensitive
	}
	if resolver.baseDir != "" {
		absolute, err := filepath.Abs(resolver.baseDir)
		if err != nil {
			return nil, fmt.Errorf("%w: base dir: %v", ErrPathInvalid, err)
		}
		resolver.baseDir = filepath.Clean(absolute)
	}
	roots, err := normalizeRoots(config.Roots)
	if err != nil {
		return nil, err
	}
	resolver.roots = roots
	return resolver, nil
}

// Resolve 归一化 path 并校验它落在允许根目录内。
// 它解析已有前缀的符号链接；对尚不存在的写目标，解析其最近存在祖先，
// 从而阻止通过符号链接目录逃逸。
func (resolver *Resolver) Resolve(path string) (string, error) {
	if resolver == nil {
		return "", ErrPathInvalid
	}
	expanded, err := resolver.expand(path)
	if err != nil {
		return "", err
	}
	absolute := expanded
	if !filepath.IsAbs(absolute) {
		if resolver.baseDir == "" {
			return "", fmt.Errorf("%w: relative path %q requires a base directory", ErrPathInvalid, path)
		}
		absolute = filepath.Join(resolver.baseDir, absolute)
	}
	clean := filepath.Clean(absolute)
	if len(resolver.roots) == 0 {
		return clean, nil
	}
	resolved, err := resolveExistingPrefix(clean)
	if err != nil {
		return "", err
	}
	if !resolver.withinRoots(resolved) {
		return "", fmt.Errorf("%w: %s", ErrPathOutsideRoots, clean)
	}
	return resolved, nil
}

// ResolveParent 归一化 path 并解析其父目录的符号链接，但保留最后一个组件。
// stat/删除/改名等操作必须作用于目录项本身（symlink 不能被跟随掉），
// 同时父目录逃逸仍会被拒绝。
func (resolver *Resolver) ResolveParent(path string) (string, error) {
	if resolver == nil {
		return "", ErrPathInvalid
	}
	expanded, err := resolver.expand(path)
	if err != nil {
		return "", err
	}
	absolute := expanded
	if !filepath.IsAbs(absolute) {
		if resolver.baseDir == "" {
			return "", fmt.Errorf("%w: relative path %q requires a base directory", ErrPathInvalid, path)
		}
		absolute = filepath.Join(resolver.baseDir, absolute)
	}
	clean := filepath.Clean(absolute)
	if len(resolver.roots) == 0 {
		return clean, nil
	}
	parent := filepath.Dir(clean)
	if parent == clean {
		resolved, err := resolveExistingPrefix(clean)
		if err != nil {
			return "", err
		}
		if !resolver.withinRoots(resolved) {
			return "", fmt.Errorf("%w: %s", ErrPathOutsideRoots, clean)
		}
		return resolved, nil
	}
	resolvedParent, err := resolveExistingPrefix(parent)
	if err != nil {
		return "", err
	}
	resolved := filepath.Join(resolvedParent, filepath.Base(clean))
	if !resolver.withinRoots(resolved) {
		return "", fmt.Errorf("%w: %s", ErrPathOutsideRoots, clean)
	}
	return resolved, nil
}

// Roots 返回归一化后的允许根目录副本（诊断用）。
func (resolver *Resolver) Roots() []string {
	if resolver == nil {
		return nil
	}
	return append([]string(nil), resolver.roots...)
}

func (resolver *Resolver) expand(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("%w: empty path", ErrPathInvalid)
	}
	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") || strings.HasPrefix(trimmed, `~\`) {
		home := resolver.home
		if home == "" {
			resolved, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("%w: expand home: %v", ErrPathInvalid, err)
			}
			home = resolved
		}
		if trimmed == "~" {
			return home, nil
		}
		return filepath.Join(home, trimmed[2:]), nil
	}
	return trimmed, nil
}

func (resolver *Resolver) withinRoots(path string) bool {
	candidate := normalizeForCompare(path, resolver.caseInsensitive)
	for _, root := range resolver.roots {
		normalizedRoot := normalizeForCompare(root, resolver.caseInsensitive)
		if candidate == normalizedRoot {
			return true
		}
		prefix := normalizedRoot
		if !strings.HasSuffix(prefix, string(filepath.Separator)) {
			prefix += string(filepath.Separator)
		}
		if strings.HasPrefix(candidate, prefix) {
			return true
		}
	}
	return false
}

// resolveExistingPrefix 逐段解析 path 的符号链接：每个已存在组件都跟随链接，
// 尚不存在的尾部拼接到已解析祖先上。与 filepath.EvalSymlinks 不同，它把断链
// （target 不存在）也解析到 target 路径，从而在写入前拒绝逃逸出 roots 的断链。
func resolveExistingPrefix(path string) (string, error) {
	return resolvePath(filepath.Clean(path), 0)
}

func resolvePath(path string, depth int) (string, error) {
	const maxSymlinkDepth = 40
	if depth > maxSymlinkDepth {
		return "", fmt.Errorf("%w: too many symlink levels at %q", ErrPathInvalid, path)
	}
	volume := filepath.VolumeName(path)
	rest := strings.TrimPrefix(path, volume)
	components := strings.Split(strings.TrimPrefix(rest, string(filepath.Separator)), string(filepath.Separator))
	current := volume + string(filepath.Separator)
	for index, component := range components {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return "", fmt.Errorf("%w: resolve %q: %v", ErrPathInvalid, current, err)
			}
			for _, tail := range components[index+1:] {
				current = filepath.Join(current, tail)
			}
			return current, nil
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := os.Readlink(current)
		if err != nil {
			return "", fmt.Errorf("%w: readlink %q: %v", ErrPathInvalid, current, err)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(current), target)
		}
		if remainder := filepath.Join(components[index+1:]...); remainder != "" {
			target = filepath.Join(target, remainder)
		}
		return resolvePath(filepath.Clean(target), depth+1)
	}
	return current, nil
}

func normalizeRoots(roots []string) ([]string, error) {
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		switch {
		case trimmed == "~":
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("%w: expand root: %v", ErrPathInvalid, err)
			}
			trimmed = home
		case strings.HasPrefix(trimmed, "~/") || strings.HasPrefix(trimmed, `~\`):
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("%w: expand root: %v", ErrPathInvalid, err)
			}
			trimmed = filepath.Join(home, trimmed[2:])
		}
		absolute, err := filepath.Abs(trimmed)
		if err != nil {
			return nil, fmt.Errorf("%w: root %q: %v", ErrPathInvalid, root, err)
		}
		clean := filepath.Clean(absolute)
		if resolved, err := filepath.EvalSymlinks(clean); err == nil {
			clean = resolved
		}
		out = append(out, clean)
	}
	return out, nil
}

func normalizeForCompare(path string, caseInsensitive bool) string {
	cleaned := filepath.Clean(path)
	if caseInsensitive {
		return strings.ToLower(cleaned)
	}
	return cleaned
}

// caseInsensitivePlatform 报告当前平台文件系统通常是否大小写不敏感。
func caseInsensitivePlatform() bool {
	switch runtime.GOOS {
	case "windows", "darwin":
		return true
	default:
		return false
	}
}
