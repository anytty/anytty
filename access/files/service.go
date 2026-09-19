package files

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const listLimitMax = 500
const previewMaxBytes = 4 << 20
const previewMaxSourceBytes = 64 << 20

// List 返回分页目录窗口。目录本身允许是 root 内符号链接（解析后仍在 root 内）。
func (service *Service) List(request ListRequest) (ListResult, error) {
	if result, handled, err := fileSystemRootList(request); handled {
		return result, err
	}
	path, err := service.resolver.Resolve(request.Path)
	if err != nil {
		return ListResult{}, err
	}
	directory, err := os.Stat(path)
	if err != nil {
		return ListResult{}, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return ListResult{}, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	offset, err := decodeFileCursor(request.Cursor, directory.ModTime().UnixNano())
	if err != nil {
		return ListResult{}, err
	}
	if offset > len(entries) {
		return ListResult{}, fmt.Errorf("invalid file list cursor")
	}
	limit := request.Limit
	if limit <= 0 || limit > listLimitMax {
		limit = listLimitMax
	}
	end := min(offset+limit, len(entries))
	result := ListResult{Path: path, Entries: make([]Entry, 0, end-offset)}
	for _, item := range entries[offset:end] {
		entry, entryErr := fileEntry(filepath.Join(path, item.Name()))
		if entryErr != nil {
			return ListResult{}, entryErr
		}
		result.Entries = append(result.Entries, entry)
	}
	if end < len(entries) {
		cursor := strconv.FormatInt(directory.ModTime().UnixNano(), 10) + ":" + strconv.Itoa(end)
		result.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(cursor))
	}
	return result, nil
}

// Stat 返回目录项 metadata；最后一个组件是符号链接时保留 symlink 投影。
func (service *Service) Stat(request PathRequest) (Entry, error) {
	path, err := service.resolver.ResolveParent(request.Path)
	if err != nil {
		return Entry{}, err
	}
	return fileEntry(path)
}

// Preview 有界读取普通文件前缀；内容路径按完整符号链接解析，禁止逃逸。
func (service *Service) Preview(request PreviewRequest) (PreviewResult, error) {
	displayPath, err := service.resolver.ResolveParent(request.Path)
	if err != nil {
		return PreviewResult{}, err
	}
	resolvedPath, err := service.resolver.Resolve(request.Path)
	if err != nil {
		return PreviewResult{}, err
	}
	entry, err := fileEntry(displayPath)
	if err != nil {
		return PreviewResult{}, err
	}
	if entry.Type != "file" {
		return PreviewResult{}, fmt.Errorf("preview requires a regular file")
	}
	limit := request.MaxBytes
	if limit <= 0 || limit > previewMaxBytes {
		limit = previewMaxBytes
	}
	file, err := os.Open(resolvedPath)
	if err != nil {
		return PreviewResult{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return PreviewResult{}, err
	}
	if !info.Mode().IsRegular() {
		return PreviewResult{}, fmt.Errorf("preview requires a regular file")
	}
	if info.Size() < 0 || info.Size() > previewMaxSourceBytes {
		return PreviewResult{}, fmt.Errorf("preview source exceeds %d bytes", previewMaxSourceBytes)
	}
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return PreviewResult{}, err
	}
	truncated := int64(len(content)) > limit
	if truncated {
		content = content[:limit]
	}
	mimeType := previewMIMEType(displayPath, content)
	if !supportedPreviewMIMEType(mimeType) {
		return PreviewResult{}, fmt.Errorf("preview type %q is not supported", mimeType)
	}
	if strings.HasPrefix(mimeType, "image/") && truncated {
		return PreviewResult{}, fmt.Errorf("image preview exceeds %d bytes", limit)
	}
	digest := sha256.Sum256(content)
	return PreviewResult{Entry: entry, MIMEType: mimeType, Content: content, Truncated: truncated, SHA256: digest[:]}, nil
}

// Mkdir 创建一个目录项；最后一个组件不被跟随。
func (service *Service) Mkdir(request PathRequest) OperationResult {
	path, err := service.resolver.ResolveParent(request.Path)
	if err == nil {
		if request.Recursive {
			err = os.MkdirAll(path, 0o755)
		} else {
			err = os.Mkdir(path, 0o755)
		}
	}
	return fileOperation(path, path, err)
}

// Rename 原子重命名一个目录项。
func (service *Service) Rename(request RenameRequest) OperationResult {
	source, err := service.resolver.ResolveParent(request.Path)
	if err != nil {
		return fileOperation(request.Path, request.NewPath, err)
	}
	target, err := service.resolver.ResolveParent(request.NewPath)
	if err != nil {
		return fileOperation(source, request.NewPath, err)
	}
	if !request.Overwrite {
		if _, statErr := os.Lstat(target); statErr == nil {
			return fileOperation(source, target, os.ErrExist)
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return fileOperation(source, target, statErr)
		}
	}
	err = os.Rename(source, target)
	return fileOperation(source, target, err)
}

// Delete 删除一个目录项；递归删除不会跟随符号链接。
func (service *Service) Delete(request PathRequest) OperationResult {
	path, err := service.resolver.ResolveParent(request.Path)
	if err == nil {
		if request.Recursive {
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}
	}
	return fileOperation(path, path, err)
}

// Copy 批量复制到目标目录。
func (service *Service) Copy(request CopyMoveRequest) BatchResult {
	return service.copyMove(request, false)
}

// Move 批量移动到目标目录。
func (service *Service) Move(request CopyMoveRequest) BatchResult {
	return service.copyMove(request, true)
}

func (service *Service) copyMove(request CopyMoveRequest, move bool) BatchResult {
	result := BatchResult{Results: make([]OperationResult, 0, len(request.Paths))}
	targetDir, targetErr := service.resolver.Resolve(request.TargetDir)
	for _, raw := range request.Paths {
		source, err := service.resolver.ResolveParent(raw)
		target := filepath.Join(targetDir, filepath.Base(source))
		if targetErr != nil {
			err = targetErr
		}
		if err == nil && !request.Overwrite {
			if _, statErr := os.Lstat(target); statErr == nil {
				err = os.ErrExist
			} else if !errors.Is(statErr, os.ErrNotExist) {
				err = statErr
			}
		}
		if err == nil && request.Overwrite {
			if _, statErr := os.Lstat(target); statErr == nil {
				err = os.RemoveAll(target)
			} else if !errors.Is(statErr, os.ErrNotExist) {
				err = statErr
			}
		}
		if err == nil {
			if move {
				err = os.Rename(source, target)
			} else {
				err = copyFileTree(source, target)
			}
		}
		result.Results = append(result.Results, fileOperation(source, target, err))
	}
	return result
}

func decodeFileCursor(cursor string, directoryVersion int64) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid cursor")
	}
	version, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || version != directoryVersion {
		return 0, fmt.Errorf("stale file list cursor")
	}
	return strconv.Atoi(parts[1])
}

func fileEntry(path string) (Entry, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Entry{}, err
	}
	metadata := info
	typeName := "other"
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		typeName = "symlink"
	case info.IsDir():
		typeName = "dir"
	case info.Mode().IsRegular():
		typeName = "file"
	}
	linkTarget := ""
	if typeName == "symlink" {
		linkTarget, err = os.Readlink(path)
		if err != nil {
			return Entry{}, err
		}
		if target, statErr := os.Stat(path); statErr == nil {
			metadata = target
			switch {
			case target.IsDir():
				typeName = "dir"
			case target.Mode().IsRegular():
				typeName = "file"
			}
		}
	}
	return Entry{Path: path, Name: info.Name(), Type: typeName, Size: metadata.Size(), Mode: uint32(metadata.Mode()), ModifiedAt: metadata.ModTime().UTC(), LinkTarget: linkTarget}, nil
}

func fileOperation(path, target string, err error) OperationResult {
	result := OperationResult{Path: path, TargetPath: target, Success: err == nil}
	if err != nil {
		result.ErrorCode = fileErrorCode(err)
		result.ErrorMessage = err.Error()
	}
	return result
}

func fileErrorCode(err error) string {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "not_found"
	case errors.Is(err, os.ErrExist):
		return "already_exists"
	case errors.Is(err, os.ErrPermission):
		return "permission_denied"
	default:
		return "internal"
	}
}

func copyFileTree(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, readErr := os.Readlink(source)
		if readErr != nil {
			return readErr
		}
		return os.Symlink(link, target)
	}
	if info.IsDir() {
		if err := os.Mkdir(target, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyFileTree(filepath.Join(source, entry.Name()), filepath.Join(target, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported file type")
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func previewMIMEType(path string, content []byte) string {
	detected := normalizedMIMEType(http.DetectContentType(content))
	extension := normalizedMIMEType(mime.TypeByExtension(strings.ToLower(filepath.Ext(path))))
	if detected == "text/plain" && supportedPreviewMIMEType(extension) && !strings.HasPrefix(extension, "image/") {
		return extension
	}
	return detected
}

func normalizedMIMEType(value string) string {
	if separator := strings.IndexByte(value, ';'); separator >= 0 {
		value = value[:separator]
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func supportedPreviewMIMEType(value string) bool {
	if strings.HasPrefix(value, "text/") {
		return true
	}
	switch value {
	case "application/json", "application/xml", "application/yaml", "application/x-yaml",
		"image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp":
		return true
	default:
		return false
	}
}
