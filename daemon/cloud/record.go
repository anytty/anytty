// Package daemon 实现 daemon 进程内唯一 AnyTTY Cloud owner。
package daemon

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/protobuf/proto"
)

const recordVersion = 3

// EnrollmentRecord 是 daemon 唯一允许持久化的 Cloud 状态。
// 它不包含 Controller runtime locator、Presence、session 或任何私钥。
type EnrollmentRecord struct {
	Version       int       `json:"version"`
	DaemonID      string    `json:"daemon_id"`
	AccountID     string    `json:"account_id"`
	DisplayName   string    `json:"display_name"`
	DaemonBinding []byte    `json:"daemon_binding"`
	EdgeLocator   []byte    `json:"edge_locator"`
	EnrolledAt    time.Time `json:"enrolled_at"`
	DaemonCount   uint32    `json:"-"`
	DaemonLimit   uint32    `json:"-"`
}

// Validate 校验 v3 record，并拒绝旧字段、损坏 protobuf 和 binding/locator 身份错配。
func (record EnrollmentRecord) Validate() error {
	if record.Version != recordVersion || strings.TrimSpace(record.DaemonID) == "" || strings.TrimSpace(record.AccountID) == "" ||
		strings.TrimSpace(record.DisplayName) == "" || strings.TrimSpace(record.DisplayName) != record.DisplayName || len(record.DisplayName) > 128 ||
		len(record.DaemonBinding) == 0 || len(record.EdgeLocator) == 0 || record.EnrolledAt.IsZero() {
		return errors.New("Cloud enrollment record is incomplete or unsupported")
	}
	binding := &cloudv1.SignedEnvelope{}
	claims := &cloudv1.DaemonBindingClaims{}
	locator := &cloudv1.EdgeLocator{}
	locatorDigest := sha256.Sum256(record.EdgeLocator)
	if proto.Unmarshal(record.DaemonBinding, binding) != nil || proto.Unmarshal(binding.GetPayload(), claims) != nil || proto.Unmarshal(record.EdgeLocator, locator) != nil ||
		claims.GetDaemonId() != record.DaemonID || claims.GetAccountId() != record.AccountID || claims.GetEdgeId() != locator.GetEdgeId() ||
		!bytes.Equal(claims.GetEdgeLocatorSha256(), locatorDigest[:]) ||
		strings.TrimSpace(locator.GetPublicEndpoint()) == "" || strings.TrimSpace(locator.GetServerName()) == "" || len(locator.GetCaCertificatePem()) == 0 {
		return errors.New("Cloud enrollment record binding or Edge locator is invalid")
	}
	return nil
}

// LoadRecord 严格加载 owner-only JSON；文件不存在表示 daemon 尚未启用 Cloud。
func LoadRecord(path string) (EnrollmentRecord, error) {
	payload, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return EnrollmentRecord{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var record EnrollmentRecord
	if err := decoder.Decode(&record); err != nil {
		return EnrollmentRecord{}, fmt.Errorf("decode Cloud enrollment record: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err == nil {
		return EnrollmentRecord{}, errors.New("Cloud enrollment record has trailing data")
	}
	if err := record.Validate(); err != nil {
		return EnrollmentRecord{}, err
	}
	return record, nil
}

// SaveRecord 原子写入 0600 enrollment record，不触及 DeviceIdentity 私钥文件。
func SaveRecord(path string, record EnrollmentRecord) error {
	record.Version = recordVersion
	if err := record.Validate(); err != nil {
		return err
	}
	directory := filepath.Dir(filepath.Clean(path))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
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

// DeleteRecord removes only the Cloud enrollment credential. Device identity and local access data remain intact.
func DeleteRecord(path string) error {
	err := os.Remove(filepath.Clean(path))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
