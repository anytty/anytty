//go:build windows

package files

import (
	"fmt"
	"strings"
	"syscall"
)

var getLogicalDrives = syscall.NewLazyDLL("kernel32.dll").NewProc("GetLogicalDrives")

// fileSystemRootList 让 Windows 客户端在 "/" 上列出逻辑盘符。
func fileSystemRootList(request ListRequest) (ListResult, bool, error) {
	if strings.TrimSpace(request.Path) != "/" {
		return ListResult{}, false, nil
	}
	if request.Cursor != "" {
		return ListResult{}, true, fmt.Errorf("invalid file list cursor")
	}
	drives, _, callErr := getLogicalDrives.Call()
	if drives == 0 {
		return ListResult{}, true, fmt.Errorf("list Windows drives: %w", callErr)
	}
	result := ListResult{Path: "/"}
	for index := uint32(0); index < 26; index++ {
		if uint32(drives)&(1<<index) == 0 {
			continue
		}
		name := string(rune('A'+index)) + ":"
		result.Entries = append(result.Entries, Entry{Path: name + "/", Name: name, Type: "dir"})
	}
	return result, true, nil
}
