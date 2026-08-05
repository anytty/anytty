//go:build unix

// Package securefs 统一私有状态文件的平台权限边界。
package securefs

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// SecureDirectory 把私有状态目录限制为当前 Unix 账号可访问。
// 调用方仍拥有目录内容真值；权限设置失败必须停止读取或写入秘密。
func SecureDirectory(path string) error { return os.Chmod(path, 0o700) }

// OpenOrCreatePrivateDirectory opens the final path without following a
// symlink and rejects an existing directory that is not owner-only.
func OpenOrCreatePrivateDirectory(path string) (*os.File, error) {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return nil, err
	}
	return openPrivateDirectory(path)
}

func openPrivateDirectory(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	directory := os.NewFile(uintptr(fd), path)
	if directory == nil {
		_ = unix.Close(fd)
		return nil, errors.New("create private directory file handle")
	}
	if err := ValidatePrivateDirectoryHandle(directory); err != nil {
		_ = directory.Close()
		return nil, err
	}
	return directory, nil
}

// SecureDirectoryHandle restricts an already-open directory without resolving
// its path again.
func SecureDirectoryHandle(directory *os.File) error {
	if directory == nil {
		return errors.New("private directory handle is required")
	}
	return directory.Chmod(0o700)
}

// ValidatePrivateDirectoryHandle verifies owner and mode on an already-open directory.
func ValidatePrivateDirectoryHandle(directory *os.File) error {
	if directory == nil {
		return errors.New("private directory handle is required")
	}
	info, err := directory.Stat()
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("private directory owner is unavailable")
	}
	if !info.IsDir() || info.Mode().Perm() != 0o700 || int(stat.Uid) != os.Geteuid() {
		return fmt.Errorf("directory is not private to the current user: mode=%#o owner=%d", info.Mode().Perm(), stat.Uid)
	}
	return nil
}

// SecureFile 把私钥、credential 或 runtime record 限制为当前 Unix 账号可读写。
// 该函数不创建文件，也不替代调用方的原子发布和 Sync 边界。
func SecureFile(path string) error { return os.Chmod(path, 0o600) }

// SecureFileHandle restricts an already-open file without resolving its path again.
func SecureFileHandle(file *os.File) error {
	if file == nil {
		return errors.New("private file handle is required")
	}
	return file.Chmod(0o600)
}

// IsPrivateFile 验证文件由当前 Unix 账号拥有且 group/other 没有权限。
// 元数据缺失或平台 owner 信息不可用时返回 false，调用方必须 fail closed。
func IsPrivateFile(_ string, info os.FileInfo) bool {
	if info == nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Geteuid()
}

// ValidatePrivateFileHandle checks the metadata of the already-open file descriptor.
func ValidatePrivateFileHandle(file *os.File) error {
	if file == nil {
		return errors.New("private file handle is required")
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !IsPrivateFile(file.Name(), info) {
		return errors.New("file is not private to the current user")
	}
	return nil
}
