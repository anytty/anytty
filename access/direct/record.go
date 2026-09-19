package direct

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const listenerRecordVersion = 1

// ListenerRecord 是 access owner 发布的 Direct listener 运行记录。
// daemon 与 CLI 只读它来生成 pairing 默认地址，不把 daemon 变成网络 owner。
type ListenerRecord struct {
	Version   int       `json:"version"`
	Listen    string    `json:"listen,omitempty"`
	Signaling string    `json:"signaling"`
	ICETCP    string    `json:"ice_tcp"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RecordPath 返回给定 daemon socket 对应的 Direct listener 记录路径。
func RecordPath(socketPath string) string {
	return strings.TrimSpace(socketPath) + ".direct"
}

// WriteListenerRecord 原子写入 0600 JSON 记录。
func WriteListenerRecord(path string, record ListenerRecord) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return errors.New("direct listener record path is required")
	}
	record.Version = listenerRecordVersion
	record.Listen = strings.TrimSpace(record.Listen)
	record.Signaling = strings.TrimSpace(record.Signaling)
	record.ICETCP = strings.TrimSpace(record.ICETCP)
	if record.Signaling == "" || record.ICETCP == "" || record.UpdatedAt.IsZero() {
		return errors.New("direct listener record is incomplete")
	}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(directory, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// ReadListenerRecord 严格读取 Direct listener 记录；缺失或损坏都返回错误。
func ReadListenerRecord(path string) (ListenerRecord, error) {
	payload, err := os.ReadFile(filepath.Clean(strings.TrimSpace(path)))
	if err != nil {
		return ListenerRecord{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var record ListenerRecord
	if err := decoder.Decode(&record); err != nil {
		return ListenerRecord{}, fmt.Errorf("decode direct listener record: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err == nil {
		return ListenerRecord{}, errors.New("direct listener record has trailing data")
	}
	if record.Version != listenerRecordVersion || record.Signaling == "" || record.ICETCP == "" || record.UpdatedAt.IsZero() {
		return ListenerRecord{}, errors.New("direct listener record metadata is invalid")
	}
	return record, nil
}

// RemoveListenerRecord 删除记录；不存在视为成功。
func RemoveListenerRecord(path string) error {
	err := os.Remove(filepath.Clean(strings.TrimSpace(path)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
