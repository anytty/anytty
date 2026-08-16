//go:build !unix && !windows

// Package securefs 统一私有状态文件的平台权限边界。
package securefs

import (
	"errors"
	"os"
)

// SecureDirectory 在未专门适配的平台保留最小 owner-only mode 请求。
func SecureDirectory(path string) error { return os.Chmod(path, 0o700) }

// OpenOrCreatePrivateDirectory is unsupported where handle ownership cannot
// be established.
func OpenOrCreatePrivateDirectory(string) (*os.File, error) {
	return nil, errors.New("private directory handle validation is unsupported")
}

// ValidatePrivateDirectoryHandle fails closed where directory ownership is unavailable.
func ValidatePrivateDirectoryHandle(*os.File) error {
	return errors.New("private directory handle validation is unsupported")
}

// SecureDirectoryHandle fails closed where handle permissions are unsupported.
func SecureDirectoryHandle(*os.File) error {
	return errors.New("private directory handle permissions are unsupported")
}

// SecureFile 在未专门适配的平台保留最小 owner-only mode 请求。
func SecureFile(path string) error { return os.Chmod(path, 0o600) }

// SecureFileHandle fails closed where handle permissions are unsupported.
func SecureFileHandle(*os.File) error {
	return errors.New("private file handle permissions are unsupported")
}

// IsPrivateFile 在无法取得 owner 真值的平台拒绝宣称文件私有。
func IsPrivateFile(string, os.FileInfo) bool { return false }

// ValidatePrivateFileHandle fails closed where handle ownership cannot be established.
func ValidatePrivateFileHandle(*os.File) error {
	return errors.New("private file handle validation is unsupported")
}
