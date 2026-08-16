package remoteauth

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anytty/anytty/shared/filelock"
	"github.com/anytty/anytty/shared/filepublish"
	"github.com/anytty/anytty/shared/securefs"
)

const (
	identityPrivateKeyFile = "remote_device_identity.json"
	identityLockFile       = "remote_device_identity.lock"
)

type privateFileWriteError struct {
	err       error
	published bool
}

func (failure *privateFileWriteError) Error() string { return failure.err.Error() }

func (failure *privateFileWriteError) Unwrap() error { return failure.err }

func privateFileWritePublished(err error) bool {
	var failure *privateFileWriteError
	return errors.As(err, &failure) && failure.published
}

// Identity 是 remote daemon 的长期 Ed25519 设备身份。
// DeviceID 只用于 Hub 发现，Fingerprint 才是客户端配置中的安全身份，PrivateKey 只能留在签发 daemon 本地。
type Identity struct {
	DeviceID    string
	Fingerprint string
	PublicKey   ed25519.PublicKey
	PrivateKey  ed25519.PrivateKey
}

// Validate 校验 daemon-local DeviceIdentity 的字段、key pair 与 fingerprint 是否一致。
// 该校验只在公开进程内运行；失败表示本地身份存储或装配已损坏，调用方必须停止 presence、grant 签发或 DataChannel 授权，不能让 Companion 代签或重建身份。
func (identity Identity) Validate() error {
	if strings.TrimSpace(identity.DeviceID) == "" || len(identity.PublicKey) != ed25519.PublicKeySize || len(identity.PrivateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("incomplete DeviceIdentity")
	}
	if Fingerprint(identity.PublicKey) != identity.Fingerprint || !identity.PublicKey.Equal(identity.PrivateKey.Public()) {
		return fmt.Errorf("DeviceIdentity key and fingerprint mismatch")
	}
	return nil
}

type storedIdentity struct {
	Version    int       `json:"version"`
	DeviceID   string    `json:"device_id"`
	PrivateKey string    `json:"private_key"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewIdentity 从 daemon 已持有的 Ed25519 私钥构造 DeviceIdentity。
// privateKey 必须来自 daemon-local 安全存储；该函数复制 key material，调用方不得把返回值交给 Companion、Hub 或日志。
func NewIdentity(deviceID string, privateKey ed25519.PrivateKey) (Identity, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return Identity{}, fmt.Errorf("remote identity requires device_id")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return Identity{}, fmt.Errorf("remote identity requires ed25519 private key")
	}
	return identityFromPrivateKey(deviceID, privateKey), nil
}

// LoadOrCreateIdentity 加载或创建 daemon-local 设备身份。
// 已有 identity 的 DeviceID 不允许被调用方静默替换；私钥文件固定为 0600，避免展示名或 Hub 路由变化升级成信任迁移。
func LoadOrCreateIdentity(dir string, deviceID string) (Identity, error) {
	dir = strings.TrimSpace(dir)
	deviceID = strings.TrimSpace(deviceID)
	if dir == "" {
		return Identity{}, fmt.Errorf("remote identity requires storage directory")
	}
	if deviceID == "" {
		return Identity{}, fmt.Errorf("remote identity requires device_id")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Identity{}, fmt.Errorf("create remote identity directory: %w", err)
	}
	if err := securefs.SecureDirectory(dir); err != nil {
		return Identity{}, fmt.Errorf("secure remote identity directory: %w", err)
	}
	lock, err := filelock.Acquire(filepath.Join(dir, identityLockFile), false)
	if err != nil {
		return Identity{}, fmt.Errorf("lock remote identity: %w", err)
	}
	defer lock.Close()
	return loadOrCreateIdentityLocked(dir, deviceID)
}

func loadOrCreateIdentityLocked(dir string, deviceID string) (Identity, error) {
	path := filepath.Join(dir, identityPrivateKeyFile)
	data, err := os.ReadFile(path)
	if err == nil {
		stored, err := decodeStoredIdentity(data)
		if err != nil {
			return Identity{}, fmt.Errorf("decode remote identity: %w", err)
		}
		if strings.TrimSpace(stored.DeviceID) != deviceID {
			return Identity{}, fmt.Errorf("remote identity device_id mismatch: stored %q requested %q", stored.DeviceID, deviceID)
		}
		privateKey, err := base64.RawURLEncoding.DecodeString(stored.PrivateKey)
		if err != nil || len(privateKey) != ed25519.PrivateKeySize {
			return Identity{}, fmt.Errorf("remote identity has invalid private key")
		}
		if err := securefs.SecureFile(path); err != nil {
			return Identity{}, fmt.Errorf("secure remote identity permissions: %w", err)
		}
		return identityFromPrivateKey(deviceID, ed25519.PrivateKey(privateKey)), nil
	}
	if !os.IsNotExist(err) {
		return Identity{}, fmt.Errorf("read remote identity: %w", err)
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Identity{}, fmt.Errorf("generate remote identity: %w", err)
	}
	stored := storedIdentity{Version: 1, DeviceID: deviceID, PrivateKey: base64.RawURLEncoding.EncodeToString(privateKey), CreatedAt: time.Now().UTC()}
	payload, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return Identity{}, fmt.Errorf("encode remote identity: %w", err)
	}
	if err := writePrivateFile(path, append(payload, '\n')); err != nil {
		return Identity{}, err
	}
	return identityFromPrivateKey(deviceID, privateKey), nil
}

// LoadOrCreateLocalIdentity 加载已有 daemon DeviceIdentity，首次调用时生成随机稳定 DeviceID 与 Ed25519 key。
// hostname、endpoint label 和 Hub lookup 都不能替代该 daemon-local identity；私钥仍只写入 0600 identity store。
func LoadOrCreateLocalIdentity(dir string) (Identity, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return Identity{}, fmt.Errorf("remote identity requires storage directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Identity{}, fmt.Errorf("create remote identity directory: %w", err)
	}
	if err := securefs.SecureDirectory(dir); err != nil {
		return Identity{}, fmt.Errorf("secure remote identity directory: %w", err)
	}
	lock, err := filelock.Acquire(filepath.Join(dir, identityLockFile), false)
	if err != nil {
		return Identity{}, fmt.Errorf("lock remote identity: %w", err)
	}
	defer lock.Close()
	path := filepath.Join(dir, identityPrivateKeyFile)
	if payload, err := os.ReadFile(path); err == nil {
		stored, decodeErr := decodeStoredIdentity(payload)
		if decodeErr != nil {
			return Identity{}, fmt.Errorf("decode remote identity: %w", decodeErr)
		}
		return loadOrCreateIdentityLocked(dir, stored.DeviceID)
	} else if !os.IsNotExist(err) {
		return Identity{}, fmt.Errorf("read remote identity: %w", err)
	}
	randomID := make([]byte, 16)
	if _, err := rand.Read(randomID); err != nil {
		return Identity{}, fmt.Errorf("generate remote device id: %w", err)
	}
	return loadOrCreateIdentityLocked(dir, "device-"+base64.RawURLEncoding.EncodeToString(randomID))
}

func decodeStoredIdentity(payload []byte) (storedIdentity, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var stored storedIdentity
	if err := decoder.Decode(&stored); err != nil {
		return storedIdentity{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return storedIdentity{}, fmt.Errorf("trailing data")
	}
	if stored.Version != 1 || strings.TrimSpace(stored.DeviceID) == "" || strings.TrimSpace(stored.PrivateKey) == "" || stored.CreatedAt.IsZero() {
		return storedIdentity{}, fmt.Errorf("identity store is incomplete or unsupported")
	}
	stored.DeviceID = strings.TrimSpace(stored.DeviceID)
	stored.PrivateKey = strings.TrimSpace(stored.PrivateKey)
	stored.CreatedAt = stored.CreatedAt.UTC()
	return stored, nil
}

func identityFromPrivateKey(deviceID string, privateKey ed25519.PrivateKey) Identity {
	publicKey := append(ed25519.PublicKey(nil), privateKey.Public().(ed25519.PublicKey)...)
	return Identity{
		DeviceID:    deviceID,
		Fingerprint: Fingerprint(publicKey),
		PublicKey:   publicKey,
		PrivateKey:  append(ed25519.PrivateKey(nil), privateKey...),
	}
}

func writePrivateFile(path string, payload []byte) error {
	return writePrivateFileWithPostPublish(path, payload, finishPrivateFilePublish)
}

func writePrivateFileWithPostPublish(path string, payload []byte, postPublish func(string) error) error {
	temp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create remote identity temp file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := securefs.SecureFile(tempPath); err != nil {
		_ = temp.Close()
		return fmt.Errorf("secure remote identity temp file: %w", err)
	}
	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write remote identity: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync remote identity: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close remote identity: %w", err)
	}
	if err := filepublish.Rename(tempPath, path); err != nil {
		return fmt.Errorf("install remote identity: %w", err)
	}
	if err := postPublish(path); err != nil {
		return &privateFileWriteError{err: err, published: true}
	}
	return nil
}

func finishPrivateFilePublish(path string) error {
	if err := securefs.SecureFile(path); err != nil {
		return fmt.Errorf("secure remote identity: %w", err)
	}
	if err := filepublish.SyncDirectory(filepath.Dir(path)); err != nil {
		return fmt.Errorf("sync remote identity directory: %w", err)
	}
	return nil
}
