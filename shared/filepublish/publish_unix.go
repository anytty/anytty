//go:build !windows

// Package filepublish 提供原子文件发布后的平台持久化边界。
package filepublish

import (
	"errors"
	"os"
)

// Rename 原子替换目标文件；Unix 调用方随后必须调用 SyncDirectory 提交目录项。
// 真值来源是同一文件系统上的 rename，跨文件系统或目录项提交失败会直接返回错误。
func Rename(temporaryPath, targetPath string) error {
	return os.Rename(temporaryPath, targetPath)
}

// SyncDirectory 把已经完成的 rename 目录项提交到稳定存储。
// 该函数只负责目录 durability，不替代临时文件在 rename 前的 Sync。
func SyncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
