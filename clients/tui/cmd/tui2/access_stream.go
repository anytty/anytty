package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"

	gproto "google.golang.org/protobuf/proto"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// accessReleaseTimeout bounds the best-effort ReleaseResource call made when a
// stream or subscription closes; it runs detached so a restart never waits on it.
const accessReleaseTimeout = 2 * time.Second

// accessStreamQueueBytes bounds the program -> access queue per stream. A
// program that floods a stalled stream gets an error frame and the stream is
// cancelled instead of blocking the session loop or growing without limit.
const accessStreamQueueBytes = 4 << 20

// accessStreamBridge relays one access ResourceStream as STREAM frames. The
// host stays transparent: payloads and their access wire type bytes pass
// through unchanged, so file transfers and event subscriptions keep their
// semantics without the host interpreting them.
type accessStreamBridge struct {
	host     *Host
	id       uint64
	endpoint string
	resource *apipb.ResourceHandle
	stream   clientruntime.ResourceStream

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once

	mu     sync.Mutex
	queue  []queuedAccessFrame
	queued int
	notify chan struct{}
	closed bool
}

type queuedAccessFrame struct {
	typ     uint8
	payload []byte
}

// openAccessStream implements access.stream.open: it decodes the serialized
// ResourceHandle, opens the access stream and starts both pumps.
func (g *gateHandler) openAccessStream(req runtime.Request) (runtime.Outcome, bool) {
	resource := &apipb.ResourceHandle{}
	if err := gproto.Unmarshal(req.Params.GetAccessResource(), resource); err != nil {
		return runtime.Outcome{Error: fmt.Sprintf("access.stream.open: decode resource: %v", err)}, false
	}
	streamID := req.Params.GetStreamId()
	endpointName := req.Params.GetEndpoint()
	session := g.host.currentSession()
	if session == nil {
		return runtime.Outcome{Error: "access.stream.open: no active session"}, false
	}
	if g.host.endpoints == nil {
		return runtime.Outcome{Error: "access.stream.open: endpoint manager unavailable"}, false
	}
	if !g.host.reserveAccessStream(streamID) {
		return runtime.Outcome{Error: fmt.Sprintf("access.stream.open: stream %d is already open", streamID)}, false
	}
	// The endpoint open is I/O and runs on a background goroutine: the session
	// loop must stay free to paint frames while a slow endpoint dials.
	go func() {
		ctx, cancel := g.host.accessCallContext()
		stream, err := g.host.endpoints.OpenStream(ctx, endpointName, resource)
		cancel()
		if err != nil {
			g.host.releaseAccessReservation(streamID)
			g.completeAsync(session, req.RequestID, runtime.Outcome{Error: "access.stream.open: " + err.Error()})
			return
		}
		bridgeCtx, bridgeCancel := context.WithCancel(context.Background())
		bridge := &accessStreamBridge{
			host: g.host, id: streamID, endpoint: endpointName,
			resource: resource, stream: stream, ctx: bridgeCtx, cancel: bridgeCancel,
			done: make(chan struct{}), notify: make(chan struct{}, 1),
		}
		if !g.host.installAccessStream(session, streamID, bridge) {
			// The session restarted (or the id was reclaimed) while the open was
			// in flight: drop the stream instead of leaking it.
			bridgeCancel()
			_ = stream.Close()
			g.host.releaseAccessResource(endpointName, resource)
			g.completeAsync(session, req.RequestID, runtime.Outcome{Error: "access.stream.open: session reset"})
			return
		}
		go bridge.pumpToProgram()
		go bridge.pumpToResource()
		g.completeAsync(session, req.RequestID, runtime.Outcome{OK: true})
	}()
	return runtime.Outcome{}, true
}

// reserveAccessStream claims a stream id for one in-flight open/subscribe so a
// second request cannot race the same id while the first is still dialing.
func (h *Host) reserveAccessStream(id uint64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.accessPending[id] || h.accessStreams[id] != nil || h.accessEvents[id] != nil {
		return false
	}
	if h.accessPending == nil {
		h.accessPending = map[uint64]bool{}
	}
	h.accessPending[id] = true
	return true
}

// releaseAccessReservation drops a claim after a failed open/subscribe.
func (h *Host) releaseAccessReservation(id uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.accessPending, id)
}

// installAccessStream publishes a fully opened stream bridge and clears its
// reservation. It reports false when the reservation is gone (session reset),
// in which case the caller must release the stream it just opened.
func (h *Host) installAccessStream(session *runtime.Session, id uint64, bridge *accessStreamBridge) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	// The session may have been replaced (program restart) while the open was
	// dialing: the reservation is gone and the stale stream must be dropped.
	if h.session != session || !h.accessPending[id] {
		return false
	}
	delete(h.accessPending, id)
	if h.accessStreams == nil {
		h.accessStreams = map[uint64]*accessStreamBridge{}
	}
	h.accessStreams[id] = bridge
	return true
}

// handleAccessStreamFrame routes one program -> host STREAM frame. Frames for
// unknown (already closed) streams are dropped; the session stays alive.
func (h *Host) handleAccessStreamFrame(frame *pb.StreamFrame) error {
	if frame == nil {
		return nil
	}
	h.mu.Lock()
	bridge := h.accessStreams[frame.GetStreamId()]
	eventBridge := h.accessEvents[frame.GetStreamId()]
	h.mu.Unlock()
	switch frame.GetKind() {
	case "close", "cancel":
		if bridge != nil {
			bridge.close()
		}
		if eventBridge != nil {
			eventBridge.close()
		}
		return nil
	case "error":
		if bridge != nil {
			bridge.fail(frame.GetError())
		}
		return nil
	case "data":
		if bridge != nil {
			bridge.enqueue(uint8(frame.GetWireType()), frame.GetPayload())
		}
		return nil
	default:
		return nil
	}
}

// enqueue hands one program -> access frame to the writer pump. Overflow
// cancels the stream with an error frame instead of growing unbounded.
func (b *accessStreamBridge) enqueue(typ uint8, payload []byte) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	if b.queued+len(payload) > accessStreamQueueBytes {
		b.mu.Unlock()
		b.fail(fmt.Sprintf("stream %d exceeded the %d byte queue", b.id, accessStreamQueueBytes))
		return
	}
	b.queue = append(b.queue, queuedAccessFrame{typ: typ, payload: append([]byte(nil), payload...)})
	b.queued += len(payload)
	b.mu.Unlock()
	select {
	case b.notify <- struct{}{}:
	default:
	}
}

func (b *accessStreamBridge) next() (queuedAccessFrame, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.queue) == 0 {
		return queuedAccessFrame{}, false
	}
	frame := b.queue[0]
	b.queue = b.queue[1:]
	b.queued -= len(frame.payload)
	return frame, true
}

// pumpToProgram forwards access -> program frames, normalizing stream
// termination into close/error frames.
func (b *accessStreamBridge) pumpToProgram() {
	for {
		typ, payload, err := b.stream.Receive(b.ctx)
		if err != nil {
			if b.ctx.Err() == nil {
				_ = b.host.sendStreamFrame(&pb.StreamFrame{StreamId: b.id, Kind: "error", Error: err.Error()})
			}
			b.close()
			return
		}
		switch typ {
		case wire.TypeClosed:
			_ = b.host.sendStreamFrame(&pb.StreamFrame{StreamId: b.id, Kind: "close"})
			b.close()
			return
		case wire.TypeSyncLost:
			_ = b.host.sendStreamFrame(&pb.StreamFrame{StreamId: b.id, Kind: "error", Error: "stream sync lost"})
			b.close()
			return
		case wire.TypeError:
			_ = b.host.sendStreamFrame(&pb.StreamFrame{StreamId: b.id, Kind: "error", Error: string(payload)})
			b.close()
			return
		default:
			if err := b.host.sendStreamFrame(&pb.StreamFrame{
				StreamId: b.id, Kind: "data", WireType: uint32(typ), Payload: payload,
			}); err != nil {
				b.close()
				return
			}
		}
	}
}

// pumpToResource forwards queued program frames to the access stream. Errors
// surface to the program as an error frame before the stream closes.
func (b *accessStreamBridge) pumpToResource() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case <-b.notify:
		}
		for {
			frame, ok := b.next()
			if !ok {
				break
			}
			if err := b.stream.Send(b.ctx, frame.typ, frame.payload); err != nil {
				if b.ctx.Err() == nil {
					_ = b.host.sendStreamFrame(&pb.StreamFrame{StreamId: b.id, Kind: "error", Error: err.Error()})
				}
				b.close()
				return
			}
		}
	}
}

// fail reports one error frame and closes the bridge.
func (b *accessStreamBridge) fail(message string) {
	if message == "" {
		message = "stream failed"
	}
	_ = b.host.sendStreamFrame(&pb.StreamFrame{StreamId: b.id, Kind: "error", Error: message})
	b.close()
}

// close releases the bridge exactly once: cancel pumps, close the resource
// stream and release the access resource token.
func (b *accessStreamBridge) close() {
	b.once.Do(func() {
		b.mu.Lock()
		b.closed = true
		b.mu.Unlock()
		b.host.mu.Lock()
		if b.host.accessStreams[b.id] == b {
			delete(b.host.accessStreams, b.id)
		}
		b.host.mu.Unlock()
		b.cancel()
		_ = b.stream.Close()
		close(b.done)
		if b.resource != nil {
			command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ReleaseResource{
				ReleaseResource: &apipb.ReleaseResourceCommand{Resource: b.resource},
			}}
			// Never release on the caller's goroutine: close() runs on the frame
			// loop during a program restart and an offline endpoint would freeze
			// it for the whole dial+call budget.
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), accessReleaseTimeout)
				defer cancel()
				_, _ = b.host.endpoints.Execute(ctx, b.endpoint, command)
			}()
		}
	})
}

// closeAllAccessStreams cancels every open bridge (program restart / host
// shutdown) so access resources are released instead of leaking.
func (h *Host) closeAllAccessStreams() {
	h.mu.Lock()
	bridges := make([]*accessStreamBridge, 0, len(h.accessStreams))
	for _, bridge := range h.accessStreams {
		bridges = append(bridges, bridge)
	}
	events := make([]*accessEventBridge, 0, len(h.accessEvents))
	for _, bridge := range h.accessEvents {
		events = append(events, bridge)
	}
	// A background open/subscribe that is still dialing sees its reservation
	// gone and releases the stream it opened instead of installing it.
	h.accessPending = nil
	h.mu.Unlock()
	for _, bridge := range bridges {
		bridge.close()
	}
	for _, bridge := range events {
		bridge.close()
	}
}

// sendStreamFrame writes one host -> program STREAM frame.
func (h *Host) sendStreamFrame(frame *pb.StreamFrame) error {
	session := h.currentSession()
	if session == nil {
		return errors.New("access stream: no program session")
	}
	return session.SendStream(frame)
}

// accessEventBridge relays one access event subscription as STREAM data
// frames. The connection event channel is fanned out to every subscriber, so
// the bridge filters by the subscription handle carried in each envelope.
type accessEventBridge struct {
	host     *Host
	id       uint64
	endpoint string
	handle   *apipb.ResourceHandle
	events   <-chan *apipb.EventEnvelope

	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
}

// openAccessSubscription implements access.stream.subscribe: it executes the
// program's EventSubscribe command, binds the returned subscription handle to
// the stream id and forwards matching EventEnvelopes as STREAM data frames
// (wire_type = TypeEvent).
func (g *gateHandler) openAccessSubscription(req runtime.Request) (runtime.Outcome, bool) {
	command := &apipb.CommandEnvelope{}
	if err := gproto.Unmarshal(req.Params.GetAccessCommand(), command); err != nil {
		return runtime.Outcome{Error: fmt.Sprintf("access.stream.subscribe: decode command: %v", err)}, false
	}
	endpointName := req.Params.GetEndpoint()
	streamID := req.Params.GetStreamId()
	g.host.mu.Lock()
	_, busyOpen := g.host.accessStreams[streamID]
	_, busyEvents := g.host.accessEvents[streamID]
	g.host.mu.Unlock()
	if busyOpen || busyEvents {
		return runtime.Outcome{Error: fmt.Sprintf("access.stream.subscribe: stream %d is already open", streamID)}, false
	}

	session := g.host.currentSession()
	if session == nil {
		return runtime.Outcome{Error: "access.stream.subscribe: no active session"}, false
	}
	if g.host.endpoints == nil {
		return runtime.Outcome{Error: "access.stream.subscribe: endpoint manager unavailable"}, false
	}
	if !g.host.reserveAccessStream(streamID) {
		return runtime.Outcome{Error: fmt.Sprintf("access.stream.subscribe: stream %d is already open", streamID)}, false
	}
	// Subscribe is I/O too (execute + event channel); keep it off the session
	// loop so a slow endpoint cannot stall rendering.
	go func() {
		ctx, cancel := g.host.accessCallContext()
		result, err := g.host.endpoints.Execute(ctx, endpointName, command)
		cancel()
		if err != nil {
			g.host.releaseAccessReservation(streamID)
			g.completeAsync(session, req.RequestID, runtime.Outcome{Error: "access.stream.subscribe: " + err.Error()})
			return
		}
		handle := result.GetEventSubscription().GetSubscription()
		if handle == nil || len(handle.GetOpaqueToken()) == 0 {
			g.host.releaseAccessReservation(streamID)
			g.completeAsync(session, req.RequestID, runtime.Outcome{Error: "access.stream.subscribe: command did not return an event subscription"})
			return
		}
		// The returned channel is closed when its context is cancelled, so it
		// must live with the bridge, not with the setup call.
		bridgeCtx, bridgeCancel := context.WithCancel(context.Background())
		events, err := g.host.endpoints.SubscribeEvents(bridgeCtx, endpointName)
		if err != nil {
			bridgeCancel()
			g.host.releaseAccessResource(endpointName, handle)
			g.host.releaseAccessReservation(streamID)
			g.completeAsync(session, req.RequestID, runtime.Outcome{Error: "access.stream.subscribe: " + err.Error()})
			return
		}
		bridge := &accessEventBridge{
			host: g.host, id: streamID, endpoint: endpointName, handle: handle,
			events: events, ctx: bridgeCtx, cancel: bridgeCancel,
		}
		if !g.host.installAccessEvent(session, streamID, bridge) {
			bridgeCancel()
			g.host.releaseAccessResource(endpointName, handle)
			g.completeAsync(session, req.RequestID, runtime.Outcome{Error: "access.stream.subscribe: session reset"})
			return
		}
		go bridge.pump()
		g.completeAsync(session, req.RequestID, runtime.Outcome{OK: true})
	}()
	return runtime.Outcome{}, true
}

// installAccessEvent is installAccessStream for event subscriptions.
func (h *Host) installAccessEvent(session *runtime.Session, id uint64, bridge *accessEventBridge) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.session != session || !h.accessPending[id] {
		return false
	}
	delete(h.accessPending, id)
	if h.accessEvents == nil {
		h.accessEvents = map[uint64]*accessEventBridge{}
	}
	h.accessEvents[id] = bridge
	return true
}

func (b *accessEventBridge) pump() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case envelope, ok := <-b.events:
			if !ok {
				if b.ctx.Err() == nil {
					_ = b.host.sendStreamFrame(&pb.StreamFrame{StreamId: b.id, Kind: "error", Error: "event stream closed"})
				}
				b.close()
				return
			}
			if !sameSubscription(envelope.GetSubscription(), b.handle) {
				continue
			}
			payload, err := gproto.Marshal(envelope)
			if err != nil {
				continue
			}
			if err := b.host.sendStreamFrame(&pb.StreamFrame{
				StreamId: b.id, Kind: "data", WireType: uint32(wire.TypeEvent), Payload: payload,
			}); err != nil {
				b.close()
				return
			}
		}
	}
}

func (b *accessEventBridge) close() {
	b.once.Do(func() {
		b.host.mu.Lock()
		if b.host.accessEvents[b.id] == b {
			delete(b.host.accessEvents, b.id)
		}
		b.host.mu.Unlock()
		b.cancel()
		b.host.releaseAccessResource(b.endpoint, b.handle)
	})
}

// sameSubscription compares only the opaque token: the envelope mirrors the
// handle's session/generation metadata, but the token is the subscription id.
func sameSubscription(a, b *apipb.ResourceHandle) bool {
	if a == nil || b == nil {
		return false
	}
	return bytes.Equal(a.GetOpaqueToken(), b.GetOpaqueToken())
}

// releaseAccessResource releases one access resource token (best effort).
func (h *Host) releaseAccessResource(endpointName string, handle *apipb.ResourceHandle) {
	if handle == nil || h.endpoints == nil {
		return
	}
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ReleaseResource{
		ReleaseResource: &apipb.ReleaseResourceCommand{Resource: handle},
	}}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), accessReleaseTimeout)
		defer cancel()
		_, _ = h.endpoints.Execute(ctx, endpointName, command)
	}()
}
