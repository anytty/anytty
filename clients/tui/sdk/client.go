package sdk

import (
	"io"

	"github.com/anytty/anytty/clients/tui/sdk/core"
)

// Client is one layout-program connection (sdk/core).
type Client = core.Client

// Handlers receives the host events of one connection (sdk/core).
type Handlers = core.Handlers

// Keys is the routing declaration carried by a view (sdk/core).
type Keys = core.Keys

// Errors returned by the client (sdk/core).
var (
	ErrNoHello = core.ErrNoHello
	ErrNilRoot = core.ErrNilRoot
)

// New returns a Client reading host frames from r and writing program frames
// to w.
func New(r io.Reader, w io.Writer, handlers Handlers) *Client {
	return core.New(r, w, handlers)
}
