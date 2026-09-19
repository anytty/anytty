//go:build !windows

package files

// fileSystemRootList 在非 Windows 平台上没有虚拟盘符根，直接交给路径解析。
func fileSystemRootList(ListRequest) (ListResult, bool, error) {
	return ListResult{}, false, nil
}
