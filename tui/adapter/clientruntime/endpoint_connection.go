package clientruntimeadapter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

const defaultEndpointRegistryWatchInterval = 250 * time.Millisecond

// EndpointConnectionControl 把共享 Endpoint registry 与 runtime planner 环境投影为 TUI EndpointStore。
// RegistryPath 为空时使用 Endpoint 默认路径；Runtime 只提供只读 plan snapshot，不把 session owner 下放给 UI。
type EndpointConnectionControl struct {
	RegistryPath    string
	Runtime         EndpointPlanSnapshotSource
	Diagnostics     EndpointConnectionSnapshotSource
	Lifecycle       EndpointEnableLifecycle
	InitialRegistry endpoint.Registry
	watchInterval   time.Duration
}

// EndpointEnableLifecycle 接收已经持久化的 Endpoint 开关结果。
// 实现可以立即关闭空闲 session，或在仍有 attachment 时进入 draining。
type EndpointEnableLifecycle interface {
	SetEndpointEnabled(context.Context, state.EndpointID, bool) error
}

// EndpointConnectionSnapshotSource samples only sessions already owned by the TUI.
type EndpointConnectionSnapshotSource interface {
	ConnectionSnapshot(context.Context, state.EndpointID) (clientruntime.ConnectionSnapshot, bool, error)
}

// EndpointPlanSnapshotSource 是 adapter 读取 Go runtime planner policy 的窄只读能力。
// 它不包含 EnsureSession/Disconnect，因此 Connections 页面不能借此拥有连接 lifecycle。
type EndpointPlanSnapshotSource interface {
	PlanSnapshot(context.Context, endpoint.EndpointID) (clientruntime.EndpointPlanSnapshot, error)
}

// LoadConnections 读取最新 registry，并用同一个 Go planner 环境标记每条 Route 的当前可用性。
func (control EndpointConnectionControl) LoadConnections(ctx context.Context) (state.EndpointStore, error) {
	registry, err := endpoint.Load(control.RegistryPath)
	if err != nil {
		return state.EndpointStore{}, err
	}
	return control.project(ctx, registry)
}

// WatchConnections polls the shared registry as a normalized snapshot instead of watching one inode.
// Endpoint writers publish with atomic rename, so snapshot polling works consistently across platforms
// and never exposes the temporary file as committed state.
func (control EndpointConnectionControl) WatchConnections(ctx context.Context) (<-chan port.EndpointConnectionUpdate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	interval := control.watchInterval
	if interval <= 0 {
		interval = defaultEndpointRegistryWatchInterval
	}
	updates := make(chan port.EndpointConnectionUpdate, 1)
	go control.watchConnections(ctx, interval, updates)
	return updates, nil
}

func (control EndpointConnectionControl) watchConnections(ctx context.Context, interval time.Duration, updates chan<- port.EndpointConnectionUpdate) {
	defer close(updates)
	previous, previousPayload, previousOK := endpointRegistryWatchBaseline(control.InitialRegistry)
	lastLoadError := ""
	poll := func() bool {
		current, err := endpoint.Load(control.RegistryPath)
		if err != nil {
			message := err.Error()
			if message == lastLoadError {
				return true
			}
			lastLoadError = message
			return sendEndpointConnectionUpdate(ctx, updates, port.EndpointConnectionUpdate{Err: err})
		}
		payload, err := endpoint.Encode(current)
		if err != nil {
			message := err.Error()
			if message == lastLoadError {
				return true
			}
			lastLoadError = message
			return sendEndpointConnectionUpdate(ctx, updates, port.EndpointConnectionUpdate{Err: err})
		}
		lastLoadError = ""
		if previousOK && bytes.Equal(payload, previousPayload) {
			return true
		}
		lifecycleErr := control.applyEndpointEnabledChanges(ctx, previous, previousOK, current)
		if isContextError(lifecycleErr) {
			return false
		}
		store, projectErr := control.project(ctx, current)
		if isContextError(projectErr) {
			return false
		}
		previous, previousPayload, previousOK = current, payload, true
		return sendEndpointConnectionUpdate(ctx, updates, port.EndpointConnectionUpdate{
			Store: store,
			Err:   errors.Join(lifecycleErr, projectErr),
		})
	}
	if !poll() {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !poll() {
				return
			}
		}
	}
}

func endpointRegistryWatchBaseline(registry endpoint.Registry) (endpoint.Registry, []byte, bool) {
	if len(registry.Endpoints) == 0 {
		return endpoint.Registry{}, nil, false
	}
	payload, err := endpoint.Encode(registry)
	if err != nil {
		return endpoint.Registry{}, nil, false
	}
	return registry, payload, true
}

func (control EndpointConnectionControl) applyEndpointEnabledChanges(ctx context.Context, previous endpoint.Registry, previousOK bool, current endpoint.Registry) error {
	if control.Lifecycle == nil {
		return nil
	}
	var errs []error
	if previousOK {
		for _, oldEndpoint := range previous.List() {
			if _, stillConfigured := current.Endpoints[oldEndpoint.ID]; stillConfigured {
				continue
			}
			if err := control.Lifecycle.SetEndpointEnabled(ctx, state.EndpointID(oldEndpoint.ID), false); err != nil {
				errs = append(errs, fmt.Errorf("disable removed endpoint %q: %w", oldEndpoint.ID, err))
			}
		}
	}
	for _, currentEndpoint := range current.List() {
		oldEndpoint, existed := previous.Endpoints[currentEndpoint.ID]
		if existed && oldEndpoint.Enabled == currentEndpoint.Enabled {
			continue
		}
		if !existed && currentEndpoint.Enabled {
			continue
		}
		if err := control.Lifecycle.SetEndpointEnabled(ctx, state.EndpointID(currentEndpoint.ID), currentEndpoint.Enabled); err != nil {
			errs = append(errs, fmt.Errorf("apply endpoint %q enabled=%t: %w", currentEndpoint.ID, currentEndpoint.Enabled, err))
		}
	}
	return errors.Join(errs...)
}

func sendEndpointConnectionUpdate(ctx context.Context, updates chan<- port.EndpointConnectionUpdate, update port.EndpointConnectionUpdate) bool {
	select {
	case <-ctx.Done():
		return false
	case updates <- update:
		return true
	}
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// SetEndpointEnabled 原子回写共享 registry，再把已发布的状态同步给当前 TUI session lifecycle。
// lifecycle 或 projection 失败不能回滚已经原子发布的配置，因此仍返回最新可用投影。
func (control EndpointConnectionControl) SetEndpointEnabled(ctx context.Context, endpointID state.EndpointID, enabled bool) (state.EndpointStore, error) {
	registry, err := endpoint.UpdateContext(ctx, control.RegistryPath, false, func(current endpoint.Registry) (endpoint.Registry, error) {
		return endpoint.SetEndpointEnabled(current, endpoint.EndpointID(endpointID), enabled)
	})
	if err != nil && !endpoint.RegistryWritePublished(err) {
		return state.EndpointStore{}, err
	}
	writeErr := err
	var lifecycleErr error
	if control.Lifecycle != nil {
		lifecycleErr = control.Lifecycle.SetEndpointEnabled(ctx, endpointID, enabled)
	}
	store, projectErr := control.project(ctx, registry)
	if lifecycleErr != nil {
		lifecycleErr = fmt.Errorf("endpoint setting was saved but session lifecycle update failed: %w", lifecycleErr)
	}
	return store, errors.Join(writeErr, lifecycleErr, projectErr)
}

// ApplyConnectionSettings atomically commits policy and the complete automatic-route priority set.
func (control EndpointConnectionControl) ApplyConnectionSettings(ctx context.Context, endpointID state.EndpointID, policy state.EndpointConnectionPolicy, priorities map[string]*int) (state.EndpointStore, error) {
	domainPriorities := make(map[endpoint.RouteID]*int, len(priorities))
	for routeID, priority := range priorities {
		domainPriorities[endpoint.RouteID(routeID)] = priority
	}
	registry, err := endpoint.UpdateContext(ctx, control.RegistryPath, false, func(current endpoint.Registry) (endpoint.Registry, error) {
		next, err := endpoint.SetConnectionPolicy(current, endpoint.EndpointID(endpointID), endpoint.ConnectionPolicy{
			RoutePreference: policy.RoutePreference, CloudRelayMode: policy.CloudRelayMode, RelayTransport: policy.RelayTransport,
		})
		if err != nil {
			return endpoint.Registry{}, err
		}
		return endpoint.SetAutomaticRoutePriorities(next, endpoint.EndpointID(endpointID), domainPriorities)
	})
	if err != nil {
		return state.EndpointStore{}, err
	}
	return control.project(ctx, registry)
}

// SampleConnection reads the selected pair from an existing TUI session and never dials an offline endpoint.
func (control EndpointConnectionControl) SampleConnection(ctx context.Context, endpointID state.EndpointID) (state.EndpointConnectionSnapshot, bool, error) {
	if control.Diagnostics == nil {
		return state.EndpointConnectionSnapshot{}, false, nil
	}
	snapshot, valid, err := control.Diagnostics.ConnectionSnapshot(ctx, endpointID)
	if err != nil || !valid {
		return state.EndpointConnectionSnapshot{}, valid, err
	}
	return state.EndpointConnectionSnapshot{
		SampledAt: snapshot.SampledAt, RoundTrip: snapshot.RoundTrip,
		LocalCandidateType: snapshot.LocalCandidateType, RemoteCandidateType: snapshot.RemoteCandidateType,
		LocalAddress: snapshot.LocalAddress, RemoteAddress: snapshot.RemoteAddress, LocalPort: snapshot.LocalPort, RemotePort: snapshot.RemotePort,
		LocalProtocol: snapshot.LocalProtocol, RemoteProtocol: snapshot.RemoteProtocol, RelayTransport: snapshot.RelayTransport,
		NetworkClass: snapshot.NetworkClass, BytesSent: snapshot.BytesSent, BytesReceived: snapshot.BytesReceived,
		PacketsSent: snapshot.PacketsSent, LossEvents: snapshot.LossEvents, Connected: snapshot.Connected,
	}, true, nil
}

func (control EndpointConnectionControl) project(ctx context.Context, registry endpoint.Registry) (state.EndpointStore, error) {
	store := (state.EndpointStore{}).ApplyConnectionRegistry(registry)
	if control.Runtime == nil {
		return store, nil
	}
	for _, target := range registry.List() {
		snapshot, err := control.Runtime.PlanSnapshot(ctx, target.ID)
		if err != nil {
			return store, fmt.Errorf("load endpoint %q planner policy: %w", target.ID, err)
		}
		availability, err := endpoint.EvaluateRouteAvailability(endpoint.RouteAvailabilityRequest{
			Endpoint: target, PlanningEndpoint: snapshot.Endpoint,
			SupportedRouteKinds: snapshot.Environment.SupportedRouteKinds, AvailableCredentialRefs: snapshot.Environment.AvailableCredentialRefs,
		})
		if err != nil {
			return store, fmt.Errorf("evaluate endpoint %q route availability: %w", target.ID, err)
		}
		store = store.ApplyRouteAvailability(state.EndpointID(target.ID), availability)
	}
	return store, nil
}

var _ port.EndpointConnectionWatchService = EndpointConnectionControl{}
