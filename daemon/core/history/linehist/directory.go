package linehist

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// PrepareDirectory 在 daemon 获得单实例锁后统一落实存储策略。旧格式没有
// 兼容路径：旧正文与 sidecar 直接删除；当前块文件按新的物理上限裁剪。
func PrepareDirectory(dir string, options CompressedLineFileOptions) error {
	if strings.TrimSpace(dir) == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var result error
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, searchIndexObsoleteV1Suffix) || strings.HasSuffix(name, searchIndexObsoleteV2Suffix) || strings.HasSuffix(name, searchIndexObsoleteV3Suffix) {
			if removeErr := os.Remove(filepath.Join(dir, name)); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				result = errors.Join(result, removeErr)
			}
			continue
		}
		if strings.HasSuffix(name, searchIndexSuffix) {
			mainPath := strings.TrimSuffix(filepath.Join(dir, name), searchIndexSuffix)
			if _, err := os.Stat(mainPath); errors.Is(err, os.ErrNotExist) {
				if removeErr := os.Remove(filepath.Join(dir, name)); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
					result = errors.Join(result, removeErr)
				}
			}
			continue
		}
		if obsoleteHistoryFileName(name) {
			if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
				result = errors.Join(result, err)
			}
			continue
		}
		const suffix = ".logical-lines.bin"
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		escapedID := strings.TrimSuffix(name, suffix)
		terminalID, err := url.PathUnescape(escapedID)
		if err != nil || strings.TrimSpace(terminalID) == "" || url.PathEscape(terminalID) != escapedID {
			if removeErr := os.Remove(filepath.Join(dir, name)); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				result = errors.Join(result, removeErr)
			}
			if removeErr := os.Remove(searchIndexPath(filepath.Join(dir, name))); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				result = errors.Join(result, removeErr)
			}
			continue
		}
		file, err := OpenCompressedLineFile(dir, terminalID, options)
		if err != nil {
			result = errors.Join(result, err)
			continue
		}
		if err := file.Close(); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

// DeleteTerminalHistory 删除一个 terminal 的全部 history 物理文件。
func DeleteTerminalHistory(dir string, terminalID string) (int, error) {
	dir = strings.TrimSpace(dir)
	terminalID = strings.TrimSpace(terminalID)
	if dir == "" || terminalID == "" {
		return 0, os.ErrInvalid
	}
	currentPath := filepath.Join(dir, url.PathEscape(terminalID)+".logical-lines.bin")
	base := strings.TrimSuffix(currentPath, ".logical-lines.bin")
	paths := []string{
		currentPath,
		searchIndexPath(currentPath),
		currentPath + searchIndexObsoleteV1Suffix,
		currentPath + searchIndexObsoleteV2Suffix,
		currentPath + searchIndexObsoleteV3Suffix,
		currentPath + ".idx",
		base + ".history-lines.bin",
		base + ".screen-rows.bin",
	}
	matches, _ := filepath.Glob(currentPath + ".rows.*.idx")
	paths = append(paths, matches...)
	return removeHistoryPaths(paths)
}

// DeleteAllHistory 删除目录内所有 terminal history 文件。
func DeleteAllHistory(dir string) (int, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return 0, os.ErrInvalid
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".logical-lines.bin") || strings.HasSuffix(name, searchIndexSuffix) || strings.HasSuffix(name, searchIndexObsoleteV1Suffix) || strings.HasSuffix(name, searchIndexObsoleteV2Suffix) || strings.HasSuffix(name, searchIndexObsoleteV3Suffix) || obsoleteHistoryFileName(name) {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	return removeHistoryPaths(paths)
}

// DeleteObsoleteCompactHistory 删除早期 core-v2 history 的 .compact 文件。
// 它不递归，也不删除目录中的未知文件。
func DeleteObsoleteCompactHistory(dir string) (int, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return 0, os.ErrInvalid
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	paths := make([]string, 0, len(entries))
	hasUnknown := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".compact") {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		} else {
			hasUnknown = true
		}
	}
	removed, removeErr := removeHistoryPaths(paths)
	if removeErr == nil && !hasUnknown {
		if err := os.Remove(dir); err != nil && !errors.Is(err, os.ErrNotExist) {
			removeErr = err
		}
	}
	return removed, removeErr
}

func removeHistoryPaths(paths []string) (int, error) {
	removed := 0
	var result error
	for _, path := range paths {
		if err := os.Remove(path); err == nil {
			removed++
		} else if !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, err)
		}
	}
	return removed, result
}

func obsoleteHistoryFileName(name string) bool {
	return strings.HasSuffix(name, ".history-lines.bin") ||
		strings.HasSuffix(name, ".screen-rows.bin") ||
		strings.HasSuffix(name, ".logical-lines.bin.idx") ||
		(strings.Contains(name, ".logical-lines.bin.rows.") && strings.HasSuffix(name, ".idx")) ||
		strings.HasPrefix(name, ".anytty-history-compact-")
}
