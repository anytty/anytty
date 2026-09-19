package endpoint

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	protocoladapter "github.com/anytty/anytty/access/engine/adapter/protocol"
	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	cloudclient "github.com/anytty/anytty/access/transport/client"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
)

// sharedClient is the production sessionConn: it wraps one ready
// client/runtime session (client/adapter/protocol ApplicationClient) and
// projects it onto the same operations the tui2 manager and RemotePTY have
// always used. This is the single daemon dial path of the TUI; there is no
// tui2-owned protocol/framing client anymore (CLIENT_SHARING §3, G1).
type sharedClient struct {
	cfg     Config
	app     *protocoladapter.ApplicationClient
	owner   *clientruntime.SessionOwner
	runtime *clientruntime.ClientRuntime
	bridge  *tcpBridge

	ctx    context.Context
	cancel context.CancelFunc

	mu      sync.Mutex
	streams map[*stream]struct{}
	ended   bool
	err     error
	done    chan struct{}
	wg      sync.WaitGroup
}

// dialSharedSession builds the shared endpoint runtime for cfg and acquires
// one ready session against the resolved registry read list
// (TUI2_ENDPOINTS + client/endpoint.DefaultPath).
func dialSharedSession(ctx context.Context, cfg Config) (sessionConn, error) {
	return dialSharedSessionWithRegistry(ctx, cfg, nil)
}

// dialSharedSessionWithRegistry is dialSharedSession with an explicit ordered
// registry path list (empty = TUI2_ENDPOINTS + client/endpoint.DefaultPath).
// Explicit files win by name. tcp keeps its compatibility meaning: relay to a
// local unix socket, then dial it with the shared local-unix route.
func dialSharedSessionWithRegistry(ctx context.Context, cfg Config, registryPaths []string) (sessionConn, error) {
	return dialSharedSessionWithRoutes(ctx, cfg, registryPaths, nil)
}

// dialSharedSessionWithRoutes is dialSharedSessionWithRegistry with the
// effective route policy (nil = DefaultRouteKinds). Only enabled kinds enter
// the planner environment, so an old registry with direct/cloud routes cannot
// trigger a 4s WebRTC dial unless the operator opted in (TUI2_ROUTES/-routes).
func dialSharedSessionWithRoutes(ctx context.Context, cfg Config, registryPaths []string, routes []clientendpoint.RouteKind) (sessionConn, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.KindName() != KindDaemon {
		return nil, fmt.Errorf("endpoint %q is not a daemon endpoint", cfg.Name)
	}
	routes = normalizeRouteKinds(routes)
	var bridge *tcpBridge
	socket := strings.TrimSpace(cfg.Socket)
	if cfg.ConnectModeName() == ConnectDirectTCP {
		// The relay lives as long as the session; the dial context must not
		// tear down an established relay when the dial deadline expires.
		started, err := startTCPBridge(nil, cfg.Address, cfg.Name)
		if err != nil {
			return nil, err
		}
		bridge = started
		socket = started.Socket()
	}
	closeBridge := func() {
		if bridge != nil {
			_ = bridge.Close()
		}
	}
	var cloudProtocol *cloudclient.Client
	if routeKindEnabled(routes, clientendpoint.RouteManagedWebRTC) {
		cloudProtocol, _ = tuiCloudClient()
	}
	snapshot, err := sharedPlanSnapshot(ctx, registryPaths, cfg, socket, cloudProtocol != nil, routes)
	if err != nil {
		closeBridge()
		return nil, err
	}
	owner, runtime, err := newSharedEndpointRuntime(snapshot.Endpoint, snapshot.Environment, cloudProtocol)
	if err != nil {
		closeBridge()
		return nil, err
	}
	ready, err := runtime.AcquireSession(ctx, clientruntime.ConnectRequest{
		EndpointID: snapshot.Endpoint.ID,
		Intent:     clientruntime.ConnectIntentInteractive,
	})
	if err != nil {
		_ = owner.Close()
		closeBridge()
		return nil, err
	}
	app, err := protocoladapter.NewRuntimeApplicationClient(ready, runtime)
	if err != nil {
		_ = owner.Close()
		closeBridge()
		return nil, err
	}
	sessionCtx, cancel := context.WithCancel(context.Background())
	client := &sharedClient{
		cfg:     cfg,
		app:     app,
		owner:   owner,
		runtime: runtime,
		bridge:  bridge,
		ctx:     sessionCtx,
		cancel:  cancel,
		streams: map[*stream]struct{}{},
		done:    make(chan struct{}),
	}
	go func() {
		<-app.Done()
		client.fail(client.appEndError())
	}()
	return client, nil
}

func (c *sharedClient) appEndError() error {
	if err := c.app.Err(); err != nil {
		return fmt.Errorf("endpoint %q: connection lost: %w", c.cfg.Name, err)
	}
	return fmt.Errorf("endpoint %q: %w", c.cfg.Name, ErrEndpointClosed)
}

// Done is closed when the ready session ends.
func (c *sharedClient) Done() <-chan struct{} { return c.done }

// Err reports the session end cause.
func (c *sharedClient) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *sharedClient) fail(err error) {
	c.mu.Lock()
	if c.ended {
		c.mu.Unlock()
		return
	}
	c.ended = true
	c.err = err
	streams := make([]*stream, 0, len(c.streams))
	for s := range c.streams {
		streams = append(streams, s)
	}
	c.streams = map[*stream]struct{}{}
	close(c.done)
	c.mu.Unlock()
	for _, s := range streams {
		s.close(err)
	}
}

func (c *sharedClient) track(s *stream) {
	c.mu.Lock()
	c.streams[s] = struct{}{}
	c.mu.Unlock()
}

func (c *sharedClient) untrack(s *stream) {
	c.mu.Lock()
	delete(c.streams, s)
	c.mu.Unlock()
}

// close releases the session, the owner generation and the tcp relay.
func (c *sharedClient) close() error {
	c.cancel()
	_ = c.app.Close()
	_ = c.owner.Close()
	if c.bridge != nil {
		_ = c.bridge.Close()
	}
	c.fail(fmt.Errorf("endpoint %q: %w", c.cfg.Name, ErrEndpointClosed))
	return nil
}

func (c *sharedClient) terminalRef(id string) *apipb.TerminalRef {
	return &apipb.TerminalRef{EndpointId: c.cfg.Name, TerminalId: id}
}

func (c *sharedClient) list(ctx context.Context) ([]*apipb.TerminalInfo, error) {
	result, err := c.app.TerminalList(ctx, &apipb.TerminalListCommand{})
	if err != nil {
		return nil, err
	}
	return result.GetTerminals(), nil
}

func (c *sharedClient) defaults(ctx context.Context) (*apipb.TerminalDefaults, error) {
	result, err := c.app.TerminalDefaults(ctx, &apipb.TerminalDefaultsCommand{})
	if err != nil {
		return nil, err
	}
	return result.GetDefaults(), nil
}

func (c *sharedClient) create(ctx context.Context, spec *apipb.TerminalCreateSpec) (*apipb.TerminalInfo, error) {
	result, err := c.app.TerminalCreate(ctx, &apipb.TerminalCreateCommand{Terminal: spec})
	if err != nil {
		return nil, err
	}
	return result.GetTerminal(), nil
}

func (c *sharedClient) kill(ctx context.Context, id string) error {
	return c.app.TerminalKill(ctx, &apipb.TerminalKillCommand{Terminal: c.terminalRef(id)})
}

func (c *sharedClient) remove(ctx context.Context, id string) error {
	return c.app.TerminalRemove(ctx, &apipb.TerminalRemoveCommand{Terminal: c.terminalRef(id)})
}

func (c *sharedClient) restart(ctx context.Context, id string) error {
	return c.app.TerminalRestart(ctx, &apipb.TerminalRestartCommand{Terminal: c.terminalRef(id)})
}

func (c *sharedClient) liveScreen(ctx context.Context, id string, observed uint64) (*apipb.NativeScreenResult, error) {
	return c.app.LiveScreenNext(ctx, &apipb.LiveScreenNextCommand{
		Terminal:         c.terminalRef(id),
		ObservedRevision: observed,
	})
}

// openAttachment reserves one collaborator attachment. The raw stream is not
// started yet: the caller fetches the authoritative screen snapshot first and
// then calls startStream (snapshot is truth, live deltas never replay).
func (c *sharedClient) openAttachment(ctx context.Context, id, surface, view string, cols, rows int) (*attachment, error) {
	result, err := c.app.ExecuteTerminal(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalAttach{TerminalAttach: &apipb.TerminalAttachCommand{
		Terminal:     c.terminalRef(id),
		Mode:         apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId:    surface,
		ViewId:       view,
	}}})
	if err != nil {
		return nil, err
	}
	attach := result.GetTerminalAttach()
	handle := attach.GetAttachment()
	if handle == nil || handle.GetResource() == nil {
		return nil, fmt.Errorf("endpoint %q: attach %s returned no attachment resource", c.cfg.Name, id)
	}
	resource := handle.GetResource()
	channel, _ := c.app.ApplicationAttachmentChannel(resource)
	att := &attachment{
		client:   c,
		terminal: id,
		resource: resource,
		channel:  channel,
		surface:  surface,
		view:     view,
		mode:     attach.GetMode(),
		policy:   attach.GetResizePolicy(),
		size:     attach.GetSize(),
	}
	if control := attach.GetResizeControl(); control.GetOwnership() != nil {
		att.epoch = control.GetOwnership().GetEpoch()
	}
	return att, nil
}

// startStream opens the session-bound resource stream and pumps its frames
// into the attachment demultiplexer. OpenResourceStream already performs the
// stream-ready handshake for terminal attachments.
func (c *sharedClient) startStream(ctx context.Context, att *attachment, timeout time.Duration) error {
	resource, err := c.app.OpenResourceStream(att.resource)
	if err != nil {
		return fmt.Errorf("endpoint %q: attach %s stream: %w", c.cfg.Name, att.terminal, err)
	}
	streamCtx, cancel := context.WithCancel(c.ctx)
	att.stop = func() {
		cancel()
		_ = resource.Close()
	}
	att.stream = newStream()
	c.track(att.stream)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			typ, payload, err := resource.Receive(streamCtx)
			if err != nil {
				att.stream.close(c.streamEndError(err))
				return
			}
			att.stream.push(typ, payload)
			if typ == wire.TypeClosed {
				att.stream.close(nil)
				return
			}
		}
	}()
	return nil
}

// streamEndError classifies a resource stream end. A clean daemon terminal
// close always arrives as a TypeClosed frame first (handled by the pump), so
// a bare io.EOF/context cancellation means the connection dropped and must
// stay transient: RemotePTY waits for the supervisor to rebind instead of
// treating it as the terminal exiting.
func (c *sharedClient) streamEndError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
		select {
		case <-c.done:
			return fmt.Errorf("endpoint %q: connection lost", c.cfg.Name)
		default:
		}
		return fmt.Errorf("endpoint %q: attachment stream ended without a close frame: %v", c.cfg.Name, err)
	}
	return fmt.Errorf("endpoint %q: attachment stream lost: %w", c.cfg.Name, err)
}

func (c *sharedClient) input(ctx context.Context, att *attachment, data []byte) error {
	return c.app.TerminalInput(ctx, &apipb.TerminalInputCommand{
		Attachment: att.resource,
		Data:       append([]byte(nil), data...),
	})
}

// resize applies a window size through the shared command path with the
// daemon resize-owner CAS.
func (c *sharedClient) resize(ctx context.Context, att *attachment, cols, rows int, takeOwnership bool, expectedEpoch uint64) (*apipb.TerminalResizeResult, error) {
	result, err := c.app.TerminalResize(ctx, &apipb.TerminalResizeCommand{
		Attachment:         att.resource,
		Size:               &apipb.TerminalSize{Cols: uint32(cols), Rows: uint32(rows)},
		ResizePolicy:       apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		TakeOwnership:      takeOwnership,
		ExpectedOwnerEpoch: expectedEpoch,
	})
	if err != nil {
		return nil, err
	}
	if control := result.GetResizeControl(); control.GetOwnership() != nil {
		if epoch := control.GetOwnership().GetEpoch(); epoch > att.epoch {
			att.epoch = epoch
		}
	}
	return result, nil
}

// detach stops the resource stream and releases the attachment.
func (c *sharedClient) detach(ctx context.Context, att *attachment) error {
	if att.stop != nil {
		att.stop()
		att.stop = nil
	}
	if att.stream != nil {
		att.stream.close(nil)
		c.untrack(att.stream)
	}
	return c.app.TerminalDetach(ctx, &apipb.TerminalDetachCommand{Attachment: att.resource})
}

var _ sessionConn = (*sharedClient)(nil)
