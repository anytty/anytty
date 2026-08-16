package port

import (
	"context"

	"github.com/anytty/anytty/tui/state"
)

// EndpointConnectionService owns TUI reads and atomic writes for shared connection settings.
// 实现必须委托 Go Endpoint registry/planner；不得在 reducer 中复制 priority 校验、credential 或 route 可用性规则。
type EndpointConnectionService interface {
	LoadConnections(context.Context) (state.EndpointStore, error)
	SampleConnection(context.Context, state.EndpointID) (state.EndpointConnectionSnapshot, bool, error)
	SetEndpointEnabled(context.Context, state.EndpointID, bool) (state.EndpointStore, error)
	ApplyConnectionSettings(context.Context, state.EndpointID, state.EndpointConnectionPolicy, map[string]*int) (state.EndpointStore, error)
}

// EndpointConnectionUpdate publishes one externally committed registry snapshot.
// A non-empty Store remains usable when Err reports a lifecycle or planner projection failure.
type EndpointConnectionUpdate struct {
	Store state.EndpointStore
	Err   error
}

// EndpointConnectionWatchService is an optional long-lived extension used by interactive clients.
// Implementations must keep the last valid registry active while malformed external writes are repaired.
type EndpointConnectionWatchService interface {
	WatchConnections(context.Context) (<-chan EndpointConnectionUpdate, error)
}
