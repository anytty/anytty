//go:build !windows

package core

func fileSystemRootList(FileListRequest) (FileListResult, bool, error) {
	return FileListResult{}, false, nil
}
