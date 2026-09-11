package port

import (
	"context"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/state"
)

// PluginDelivery contains only daemon-authenticated ingress. Local SDK bridges
// must never synthesize a successful ingress without the daemon round trip.
type PluginDelivery struct {
	EndpointID state.EndpointID
	PluginID   string
	Message    *apipb.PluginMessage
	Err        error
}
type PluginService interface {
	Watch(context.Context) (<-chan PluginDelivery, error)
	Send(context.Context, state.EndpointID, *apipb.PluginMessage) error
	ResolveDaemon(string) (state.EndpointID, bool)
}
