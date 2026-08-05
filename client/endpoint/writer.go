package endpoint

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anytty/anytty/shared/filelock"
	"github.com/anytty/anytty/shared/filepublish"
	"github.com/anytty/anytty/shared/securefs"
	"gopkg.in/yaml.v3"
)

type registryWriteError struct {
	err       error
	published bool
}

func (failure *registryWriteError) Error() string { return failure.err.Error() }

func (failure *registryWriteError) Unwrap() error { return failure.err }

// RegistryWritePublished 判断 Save 返回错误时新 registry 是否已经完成原子 rename。
// true 表示调用方不得回滚已经提交的 credential；该错误只表示父目录同步失败、掉电耐久性不确定。
func RegistryWritePublished(err error) bool {
	var failure *registryWriteError
	return errors.As(err, &failure) && failure.published
}

// Encode 把规范化 Endpoint registry 编码为稳定 v2 YAML。
// 输出只包含客户端期望状态与 credential ref；secret、runtime attempt、Path、错误和临时 LAN candidate 永远不落盘。
func Encode(registry Registry) ([]byte, error) {
	normalized, err := registry.Normalize()
	if err != nil {
		return nil, err
	}
	defaultEndpoint := string(normalized.Default)
	document := registryDocument{Version: RegistryVersion, Default: &defaultEndpoint, Endpoints: make(map[string]endpointDocument, len(normalized.Endpoints))}
	for _, endpoint := range normalized.List() {
		enabled := endpoint.Enabled
		value := endpointDocument{
			Label: endpoint.Label, LabelSource: string(endpoint.LabelSource), DeviceID: endpoint.DaemonIdentity.DeviceID,
			DeviceFingerprint: endpoint.DaemonIdentity.DeviceFingerprint, Enabled: &enabled, ConnectMode: string(endpoint.ConnectMode),
			Routes: make(map[string]routeDocument, len(endpoint.Routes)),
		}
		if endpoint.SelectionPolicy.HedgeDelayConfigured {
			value.Selection.HedgeDelay = fmt.Sprintf("%dms", endpoint.SelectionPolicy.HedgeDelay/time.Millisecond)
		}
		value.Selection.RoutePreference = string(endpoint.SelectionPolicy.RoutePreference)
		for _, route := range endpoint.RouteList() {
			routeEnabled := route.Enabled
			var credentialDescriptor *credentialDescriptorDocument
			if route.CredentialDescriptor != nil {
				credentialDescriptor = &credentialDescriptorDocument{
					DescriptorID: route.CredentialDescriptor.DescriptorID,
					Kind:         string(route.CredentialDescriptor.Kind),
					Exportable:   route.CredentialDescriptor.Exportable,
				}
			}
			value.Routes[string(route.ID)] = routeDocument{
				Kind: string(route.Kind), DisplayName: route.DisplayName, Enabled: &routeEnabled, ManualOnly: route.ManualOnly, Priority: clonePriority(route.Priority),
				CredentialRef: route.CredentialRef, Source: string(route.Source), PolicySource: string(route.PolicySource), Socket: route.Socket,
				Host: route.Host, Port: route.Port, User: route.User, ProxyJump: route.ProxyJump,
				HostKeyFingerprints: append([]string(nil), route.HostKeyFingerprints...), CredentialDescriptor: credentialDescriptor,
				RemoteSignalingAddress: route.RemoteSignalingAddress, RemoteICETCPAddress: route.RemoteICETCPAddress,
				SignalingAddresses: append([]string(nil), route.SignalingAddresses...), ICETCPAddresses: append([]string(nil), route.ICETCPAddresses...),
				AdvertisedAddresses: append([]string(nil), route.AdvertisedAddresses...), ServerName: route.ServerName,
				TargetDeviceID: route.TargetDeviceID, AccountProfileRef: route.AccountProfileRef,
				RelayMode: string(route.RelayMode), RelayTransport: string(route.RelayTransport),
			}
		}
		document.Endpoints[string(endpoint.ID)] = value
	}
	payload, err := yaml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode endpoint registry: %w", err)
	}
	if len(payload) > MaxRegistryBytes {
		return nil, connectionError(ErrorSizeLimit, "encoded endpoint registry exceeds %d bytes", MaxRegistryBytes)
	}
	return payload, nil
}

// Save 在跨进程事务锁内原子写入 Endpoint registry；空 path 使用 DefaultPath。
// 文件固定为 0600，rename 前失败保留旧文件；rename 后同步父目录以缩小掉电丢失窗口。
// rename 后的同步错误由 RegistryWritePublished 标记，调用方不能把它误当成未提交并回滚 credential。
// 需要 read-modify-write 的调用方必须使用 Update，不能在独立 Load/Save 之间假设 registry 未被其他进程修改。
func Save(path string, registry Registry) error {
	return SaveContext(context.Background(), path, registry)
}

// SaveContext 等价于 Save，但等待 registry transaction lock 时响应调用方取消。
// CLI 等带 deadline 的入口必须使用本方法或 UpdateContext，避免 owner 进程卡住时突破根命令超时。
func SaveContext(ctx context.Context, path string, registry Registry) error {
	path = resolvedRegistryPath(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create endpoint registry directory: %w", err)
	}
	if err := securefs.SecureDirectory(filepath.Dir(path)); err != nil {
		return fmt.Errorf("secure endpoint registry directory: %w", err)
	}
	lock, err := filelock.AcquireContext(ctx, path+".lock", false)
	if err != nil {
		return fmt.Errorf("lock endpoint registry: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(err, lock.Close())
	}
	writeErr := saveRegistryFile(path, registry)
	closeErr := lock.Close()
	if writeErr == nil && closeErr != nil {
		return &registryWriteError{err: fmt.Errorf("release endpoint registry lock: %w", closeErr), published: true}
	}
	return errors.Join(writeErr, closeErr)
}

// Update 在同一个跨进程锁内执行 Endpoint registry 的 load、领域 mutation 与原子 save。
// createIfMissing 只允许显式创建入口把缺失的显式 path 解释为空 v2 registry；其他错误、identity conflict 和 mutation 失败均不写文件。
// callback 不得泄漏 secret 或发起嵌套 registry Update；它返回的 registry 是唯一待提交客户端配置真值。
func Update(path string, createIfMissing bool, mutate func(Registry) (Registry, error)) (Registry, error) {
	return UpdateContext(context.Background(), path, createIfMissing, mutate)
}

// UpdateContext 等价于 Update，但 registry lock 等待受 ctx 控制。
// callback 仍在同一锁内执行；它发起的 PairingExchange 等阻塞操作也必须复用该 ctx，确保取消后及时释放 transaction。
func UpdateContext(ctx context.Context, path string, createIfMissing bool, mutate func(Registry) (Registry, error)) (Registry, error) {
	if mutate == nil {
		return Registry{}, fmt.Errorf("endpoint registry update requires mutation")
	}
	explicit := strings.TrimSpace(path) != ""
	path = resolvedRegistryPath(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return Registry{}, fmt.Errorf("create endpoint registry directory: %w", err)
	}
	lock, err := filelock.AcquireContext(ctx, path+".lock", false)
	if err != nil {
		return Registry{}, fmt.Errorf("lock endpoint registry: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Registry{}, errors.Join(err, lock.Close())
	}
	registry, loadErr := Load(path)
	if loadErr != nil && !explicit && errors.Is(loadErr, os.ErrNotExist) {
		registry = DefaultRegistry()
		loadErr = nil
	}
	if loadErr != nil && explicit && createIfMissing && errors.Is(loadErr, os.ErrNotExist) {
		registry = Registry{Version: RegistryVersion, Endpoints: map[EndpointID]Endpoint{}}
		loadErr = nil
	}
	if loadErr != nil {
		return Registry{}, errors.Join(loadErr, lock.Close())
	}
	updated, updateErr := mutate(registry)
	if updateErr != nil {
		return Registry{}, errors.Join(updateErr, lock.Close())
	}
	if err := ctx.Err(); err != nil {
		return Registry{}, errors.Join(err, lock.Close())
	}
	writeErr := saveRegistryFile(path, updated)
	closeErr := lock.Close()
	if writeErr == nil && closeErr != nil {
		writeErr = &registryWriteError{err: fmt.Errorf("release endpoint registry lock: %w", closeErr), published: true}
	} else if closeErr != nil {
		writeErr = errors.Join(writeErr, closeErr)
	}
	return updated, writeErr
}

func resolvedRegistryPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return DefaultPath()
	}
	return path
}

func saveRegistryFile(path string, registry Registry) error {
	payload, err := Encode(registry)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create endpoint registry directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".connections-*.yaml")
	if err != nil {
		return fmt.Errorf("create endpoint registry temp file: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := securefs.SecureFile(temporaryPath); err != nil {
		return fmt.Errorf("secure endpoint registry temp file: %w", err)
	}
	if _, err := temporary.Write(payload); err != nil {
		return fmt.Errorf("write endpoint registry: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync endpoint registry: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close endpoint registry: %w", err)
	}
	if err := filepublish.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish endpoint registry: %w", err)
	}
	committed = true
	if err := filepublish.SyncDirectory(filepath.Dir(path)); err != nil {
		return &registryWriteError{err: fmt.Errorf("sync endpoint registry directory: %w", err), published: true}
	}
	return nil
}
