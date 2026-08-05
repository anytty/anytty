package binding

import (
	"context"
	"fmt"
	"sync"
)

const maxEngines = 64

// Registry 是 C/JNI/WASM wrapper 共用的 engine opaque-handle owner。
// 平台不得自行保存 Go pointer 或复制 engine map；所有调用先以 engine handle 进入该 registry，再由 Engine 解析 operation/session handle。
type Registry struct {
	mu         sync.Mutex
	nextHandle uint64
	engines    map[uint64]*Engine
	closed     bool
}

// NewRegistry 创建空的跨语言 engine registry。
// engine handle 从 1 单调递增且不复用，避免平台迟到 callback 命中新建实例。
func NewRegistry() *Registry {
	return &Registry{engines: make(map[uint64]*Engine)}
}

// CreateEngine 创建 Engine 并返回只在当前 Registry 内有效的 opaque handle。
// Host 装配由具体 Android/WASM wrapper 提供；失败时不会发布部分 handle。
func (registry *Registry) CreateEngine(host Host) (uint64, error) {
	if registry == nil {
		return 0, ErrClosed
	}
	engine, err := NewEngine(host)
	if err != nil {
		return 0, err
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.closed {
		_ = engine.Close()
		return 0, ErrClosed
	}
	if len(registry.engines) >= maxEngines || registry.nextHandle == ^uint64(0) {
		_ = engine.Close()
		return 0, fmt.Errorf("binding engine capacity is exhausted")
	}
	registry.nextHandle++
	registry.engines[registry.nextHandle] = engine
	return registry.nextHandle, nil
}

// OpenSession 把 serialized bindingpb.OpenSessionRequest 路由到指定 engine。
func (registry *Registry) OpenSession(engineHandle uint64, payload []byte) (uint64, error) {
	engine, err := registry.engine(engineHandle)
	if err != nil {
		return 0, err
	}
	return engine.OpenSession(payload)
}

// Execute 把 serialized apipb.CommandEnvelope 路由到指定 engine/session。
func (registry *Registry) Execute(engineHandle, sessionHandle uint64, payload []byte) (uint64, error) {
	engine, err := registry.engine(engineHandle)
	if err != nil {
		return 0, err
	}
	return engine.Execute(sessionHandle, payload)
}

// OpenResourceStream 把 serialized bindingpb.OpenResourceStreamRequest 路由到指定 session。
func (registry *Registry) OpenResourceStream(engineHandle, sessionHandle uint64, payload []byte) (uint64, error) {
	engine, err := registry.engine(engineHandle)
	if err != nil {
		return 0, err
	}
	return engine.OpenResourceStream(sessionHandle, payload)
}

// SendResourceStreamFrame 把 serialized bindingpb.ResourceStreamFrame 路由到 opaque stream handle。
func (registry *Registry) SendResourceStreamFrame(engineHandle, streamHandle uint64, payload []byte) error {
	engine, err := registry.engine(engineHandle)
	if err != nil {
		return err
	}
	return engine.SendResourceStreamFrame(streamHandle, payload)
}

// CloseResourceStream 关闭指定 opaque stream handle。
func (registry *Registry) CloseResourceStream(engineHandle, streamHandle uint64) error {
	engine, err := registry.engine(engineHandle)
	if err != nil {
		return err
	}
	return engine.CloseResourceStream(streamHandle)
}

// EngineCommand 把 serialized bindingpb.EngineCommand 路由到指定 engine。
// Registry 不解释具体业务命令，跨语言 ABI 只随该 Proto envelope 演进。
func (registry *Registry) EngineCommand(engineHandle uint64, payload []byte) (uint64, error) {
	engine, err := registry.engine(engineHandle)
	if err != nil {
		return 0, err
	}
	return engine.EngineCommand(payload)
}

// NextEvent 读取指定 engine 的下一条 serialized bindingpb.EventEnvelope。
func (registry *Registry) NextEvent(ctx context.Context, engineHandle uint64) ([]byte, error) {
	engine, err := registry.engine(engineHandle)
	if err != nil {
		return nil, err
	}
	return engine.NextEvent(ctx)
}

// Cancel 取消指定 engine 内的 operation handle。
func (registry *Registry) Cancel(engineHandle, operationHandle uint64) error {
	engine, err := registry.engine(engineHandle)
	if err != nil {
		return err
	}
	return engine.Cancel(operationHandle)
}

// CloseSession 关闭指定 engine 内的 session handle。
func (registry *Registry) CloseSession(engineHandle, sessionHandle uint64) error {
	engine, err := registry.engine(engineHandle)
	if err != nil {
		return err
	}
	return engine.CloseSession(sessionHandle)
}

// Release 释放指定 engine 内已经完成的 operation 或已经关闭的 session handle。
func (registry *Registry) Release(engineHandle, handle uint64) error {
	engine, err := registry.engine(engineHandle)
	if err != nil {
		return err
	}
	return engine.Release(handle)
}

// CloseEngine 从 registry 原子移除 engine handle，再取消其 operation 并关闭 session。
// 移除后任何迟到平台调用都返回 ErrInvalidHandle，不能命中后续新 engine。
func (registry *Registry) CloseEngine(engineHandle uint64) error {
	if registry == nil {
		return ErrClosed
	}
	registry.mu.Lock()
	if registry.closed {
		registry.mu.Unlock()
		return ErrClosed
	}
	engine := registry.engines[engineHandle]
	if engine == nil {
		registry.mu.Unlock()
		return ErrInvalidHandle
	}
	delete(registry.engines, engineHandle)
	registry.mu.Unlock()
	return engine.Close()
}

// Close 关闭 registry 内全部 engine 并禁止创建新 handle。
// 方法幂等，适用于 Android process teardown、WASM instance disposal 和桌面 library unload。
func (registry *Registry) Close() error {
	if registry == nil {
		return nil
	}
	registry.mu.Lock()
	if registry.closed {
		registry.mu.Unlock()
		return nil
	}
	registry.closed = true
	engines := make([]*Engine, 0, len(registry.engines))
	for handle, engine := range registry.engines {
		engines = append(engines, engine)
		delete(registry.engines, handle)
	}
	registry.mu.Unlock()
	var first error
	for _, engine := range engines {
		if err := engine.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (registry *Registry) engine(handle uint64) (*Engine, error) {
	if registry == nil {
		return nil, ErrClosed
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.closed {
		return nil, ErrClosed
	}
	engine := registry.engines[handle]
	if engine == nil {
		return nil, ErrInvalidHandle
	}
	return engine, nil
}
