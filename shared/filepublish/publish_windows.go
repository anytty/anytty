//go:build windows

// Package filepublish 提供原子文件发布后的平台持久化边界。
package filepublish

import "golang.org/x/sys/windows"

// Rename 使用 Windows write-through rename 原子替换目标文件。
// MOVEFILE_WRITE_THROUGH 是 Windows 的目录项提交边界；失败时不会假装发布成功。
func Rename(temporaryPath, targetPath string) error {
	from, err := windows.UTF16PtrFromString(temporaryPath)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(targetPath)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

// SyncDirectory 在 Windows 上无需再次打开目录。
// Rename 已通过 MOVEFILE_WRITE_THROUGH 完成提交，而目录句柄 Sync 在 Windows 会返回访问拒绝。
func SyncDirectory(string) error { return nil }
