package main

import (
	"os"

	"github.com/anytty/anytty/clients/tui/endpoint"
)

// localAccessEndpointName is the built-in daemon endpoint for the local access
// stack. It stays separate from the `local` host-PTY endpoint on purpose: the
// shell can forward access commands (access.call) to this endpoint without
// changing how `local` terminals are created.
const localAccessEndpointName = "local-access"

// registerLocalAccessEndpoint exposes the running local access stack under a
// distinct daemon endpoint. It is a fallback for registries that define no
// local-unix endpoint at all: normal registries (including socket "auto") are
// projected by the endpoint package and already register `local` as a daemon
// endpoint. It only registers when the socket exists at startup and no other
// endpoint already uses it.
func (h *Host) registerLocalAccessEndpoint() {
	socket := endpoint.DefaultLocalAccessSocket()
	if socket == "" {
		return
	}
	if _, err := os.Stat(socket); err != nil {
		return
	}
	if _, exists := h.endpoints.Config(localAccessEndpointName); exists {
		return
	}
	if h.endpoints.HasSocket(socket) {
		return
	}
	_ = h.endpoints.Register(endpoint.Config{
		Name:        localAccessEndpointName,
		Label:       "Local access",
		Kind:        endpoint.KindDaemon,
		ConnectMode: endpoint.ConnectLocalUnix,
		Socket:      socket,
	})
}
