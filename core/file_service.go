package core

import (
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

const fileListLimitMax = 500
const filePreviewMaxBytes = 4 << 20

func fileList(params FileListRequest) (FileListResult, error) {
	if result, handled, err := fileSystemRootList(params); handled {
		return result, err
	}
	path, err := absoluteFilePath(params.Path)
	if err != nil {
		return FileListResult{}, err
	}
	directory, err := os.Stat(path)
	if err != nil {
		return FileListResult{}, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return FileListResult{}, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	offset, err := decodeFileCursor(params.Cursor, directory.ModTime().UnixNano())
	if err != nil {
		return FileListResult{}, err
	}
	if offset > len(entries) {
		return FileListResult{}, fmt.Errorf("invalid file list cursor")
	}
	limit := params.Limit
	if limit <= 0 || limit > fileListLimitMax {
		limit = fileListLimitMax
	}
	end := min(offset+limit, len(entries))
	result := FileListResult{Path: path, Entries: make([]FileEntry, 0, end-offset)}
	for _, item := range entries[offset:end] {
		entry, entryErr := fileEntry(filepath.Join(path, item.Name()))
		if entryErr != nil {
			return FileListResult{}, entryErr
		}
		result.Entries = append(result.Entries, entry)
	}
	if end < len(entries) {
		cursor := strconv.FormatInt(directory.ModTime().UnixNano(), 10) + ":" + strconv.Itoa(end)
		result.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(cursor))
	}
	return result, nil
}

func fileStat(params FilePathRequest) (FileEntry, error) {
	path, err := absoluteFilePath(params.Path)
	if err != nil {
		return FileEntry{}, err
	}
	return fileEntry(path)
}

func filePreview(params FilePreviewRequest) (FilePreviewResult, error) {
	path, err := absoluteFilePath(params.Path)
	if err != nil {
		return FilePreviewResult{}, err
	}
	entry, err := fileEntry(path)
	if err != nil {
		return FilePreviewResult{}, err
	}
	if entry.Type != "file" {
		return FilePreviewResult{}, fmt.Errorf("preview requires a regular file")
	}
	limit := params.MaxBytes
	if limit <= 0 || limit > filePreviewMaxBytes {
		limit = filePreviewMaxBytes
	}
	file, err := os.Open(path)
	if err != nil {
		return FilePreviewResult{}, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return FilePreviewResult{}, err
	}
	truncated := int64(len(content)) > limit
	if truncated {
		content = content[:limit]
	}
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if mimeType == "" {
		mimeType = http.DetectContentType(content)
	}
	return FilePreviewResult{Entry: entry, MIMEType: mimeType, Content: content, Truncated: truncated}, nil
}

func fileMkdir(params FilePathRequest) FileOperationResult {
	path, err := absoluteFilePath(params.Path)
	if err == nil {
		if params.Recursive {
			err = os.MkdirAll(path, 0o755)
		} else {
			err = os.Mkdir(path, 0o755)
		}
	}
	return fileOperation(path, path, err)
}

func fileRename(params FileRenameRequest) FileOperationResult {
	source, err := absoluteFilePath(params.Path)
	if err != nil {
		return fileOperation(params.Path, params.NewPath, err)
	}
	target, err := absoluteFilePath(params.NewPath)
	if err != nil {
		return fileOperation(source, params.NewPath, err)
	}
	if !params.Overwrite {
		if _, statErr := os.Lstat(target); statErr == nil {
			return fileOperation(source, target, os.ErrExist)
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return fileOperation(source, target, statErr)
		}
	}
	err = os.Rename(source, target)
	return fileOperation(source, target, err)
}

func fileDelete(params FilePathRequest) FileOperationResult {
	path, err := absoluteFilePath(params.Path)
	if err == nil {
		if params.Recursive {
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}
	}
	return fileOperation(path, path, err)
}

func fileCopyMove(params FileCopyMoveRequest, move bool) FileBatchResult {
	result := FileBatchResult{Results: make([]FileOperationResult, 0, len(params.Paths))}
	targetDir, targetErr := absoluteFilePath(params.TargetDir)
	for _, raw := range params.Paths {
		source, err := absoluteFilePath(raw)
		target := filepath.Join(targetDir, filepath.Base(source))
		if targetErr != nil {
			err = targetErr
		}
		if err == nil && !params.Overwrite {
			if _, statErr := os.Lstat(target); statErr == nil {
				err = os.ErrExist
			} else if !errors.Is(statErr, os.ErrNotExist) {
				err = statErr
			}
		}
		if err == nil && params.Overwrite {
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

func absoluteFilePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("file path must be absolute")
	}
	return filepath.Clean(path), nil
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

func fileEntry(path string) (FileEntry, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return FileEntry{}, err
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
			return FileEntry{}, err
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
	return FileEntry{Path: path, Name: info.Name(), Type: typeName, Size: metadata.Size(), Mode: uint32(metadata.Mode()), ModifiedAt: metadata.ModTime().UTC(), LinkTarget: linkTarget}, nil
}

func fileOperation(path, target string, err error) FileOperationResult {
	result := FileOperationResult{Path: path, TargetPath: target, Success: err == nil}
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
