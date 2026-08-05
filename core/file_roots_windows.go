//go:build windows

package core

import (
	"fmt"
	"strings"
	"syscall"
)

var getLogicalDrives = syscall.NewLazyDLL("kernel32.dll").NewProc("GetLogicalDrives")

func fileSystemRootList(params FileListRequest) (FileListResult, bool, error) {
	if strings.TrimSpace(params.Path) != "/" {
		return FileListResult{}, false, nil
	}
	if params.Cursor != "" {
		return FileListResult{}, true, fmt.Errorf("invalid file list cursor")
	}
	drives, _, callErr := getLogicalDrives.Call()
	if drives == 0 {
		return FileListResult{}, true, fmt.Errorf("list Windows drives: %w", callErr)
	}
	result := FileListResult{Path: "/"}
	for index := uint32(0); index < 26; index++ {
		if uint32(drives)&(1<<index) == 0 {
			continue
		}
		name := string(rune('A'+index)) + ":"
		result.Entries = append(result.Entries, FileEntry{
			Path: name + "/",
			Name: name,
			Type: "dir",
		})
	}
	return result, true, nil
}
