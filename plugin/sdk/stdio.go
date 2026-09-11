package sdk

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

const MaxFrameBytes = 4 << 20

// ReadFrame and WriteFrame share one bounded unsigned-varint Protobuf framing
// contract across language SDKs. stdout belongs to this channel, never logs.
func ReadFrame(reader *bufio.Reader, message proto.Message) error {
	size, err := binary.ReadUvarint(reader)
	if err != nil {
		return err
	}
	if size == 0 || size > MaxFrameBytes {
		return fmt.Errorf("invalid plugin frame size %d", size)
	}
	data := make([]byte, int(size))
	if _, err = io.ReadFull(reader, data); err != nil {
		return err
	}
	return proto.Unmarshal(data, message)
}
func WriteFrame(writer io.Writer, message proto.Message) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	if len(data) == 0 || len(data) > MaxFrameBytes {
		return errors.New("plugin frame exceeds size limit")
	}
	var prefix [10]byte
	n := binary.PutUvarint(prefix[:], uint64(len(data)))
	frame := append(prefix[:n:n], data...)
	for len(frame) > 0 {
		n, err := writer.Write(frame)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		frame = frame[n:]
	}
	return nil
}

type stdioResponse struct {
	value *apipb.PluginResult
	err   error
}
type stdioPending struct {
	response chan stdioResponse
	endpoint string
}
type stdioWrite struct {
	frame *apipb.PluginBridgeFrame
	ctx   context.Context
	bytes int
}
type stdioTransport struct {
	reader      *bufio.Reader
	input       io.Reader
	writer      io.Writer
	mu          sync.Mutex
	pending     map[string]stdioPending
	next        atomic.Uint64
	closed      bool
	failure     error
	done        chan struct{}
	writes      chan stdioWrite
	cancels     chan *apipb.PluginBridgeFrame
	queuedBytes int
}

const maxStdioPending = 128
const maxStdioOutputBytes = 8 << 20

func NewStdioClient(input io.Reader, output io.Writer) *Client {
	t := &stdioTransport{reader: bufio.NewReader(input), input: input, writer: output, pending: make(map[string]stdioPending), done: make(chan struct{}), writes: make(chan stdioWrite, maxStdioPending), cancels: make(chan *apipb.PluginBridgeFrame, maxStdioPending)}
	client := NewClient(func(ctx context.Context, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
		return t.execute(ctx, "", command)
	})
	client.endpointFactory = func(endpoint string) Executor {
		return func(ctx context.Context, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
			return t.execute(ctx, endpoint, command)
		}
	}
	client.closeTransport = func() error { t.fail(io.ErrClosedPipe); return nil }
	go t.read()
	go t.write()
	return client
}
func (t *stdioTransport) execute(ctx context.Context, endpoint string, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}
	if deadline.After(time.Now().Add(2 * time.Minute)) {
		deadline = time.Now().Add(2 * time.Minute)
	}
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	id := strconv.FormatUint(t.next.Add(1), 10)
	frame := &apipb.PluginBridgeFrame{RequestId: id, ViaEndpointId: endpoint, DeadlineUnixMillis: deadline.UnixMilli(), Payload: &apipb.PluginBridgeFrame_Command{Command: command}}
	size := proto.Size(frame) + 10
	if proto.Size(frame) > MaxFrameBytes {
		return nil, errors.New("plugin frame exceeds size limit")
	}
	response := make(chan stdioResponse, 1)
	t.mu.Lock()
	if t.closed {
		err := t.failure
		t.mu.Unlock()
		return nil, err
	}
	if len(t.pending) >= maxStdioPending || t.queuedBytes+size > maxStdioOutputBytes {
		t.mu.Unlock()
		return nil, &Error{Code: "RESOURCE_EXHAUSTED", Message: "plugin bridge request buffer is full"}
	}
	t.pending[id] = stdioPending{response: response, endpoint: endpoint}
	t.queuedBytes += size
	t.mu.Unlock()
	defer func() { t.mu.Lock(); delete(t.pending, id); t.mu.Unlock() }()
	select {
	case t.writes <- stdioWrite{frame: frame, ctx: ctx, bytes: size}:
	case <-t.done:
		t.mu.Lock()
		t.queuedBytes -= size
		t.mu.Unlock()
		return nil, io.ErrClosedPipe
	case <-ctx.Done():
		t.mu.Lock()
		t.queuedBytes -= size
		t.mu.Unlock()
		return nil, ctx.Err()
	}
	select {
	case <-ctx.Done():
		// Control frames use a separate priority queue. If the original write has not
		// begun the writer skips it; otherwise cancellation follows the intact frame.
		cancelFrame := &apipb.PluginBridgeFrame{RequestId: id, ViaEndpointId: endpoint, Payload: &apipb.PluginBridgeFrame_CancelRequestId{CancelRequestId: id}}
		select {
		case t.cancels <- cancelFrame:
		case <-t.done:
		default:
			t.fail(errors.New("plugin cancellation queue exhausted"))
		}
		return nil, ctx.Err()
	case result := <-response:
		return result.value, result.err
	}
}
func (t *stdioTransport) write() {
	for {
		var item stdioWrite
		select {
		case <-t.done:
			return
		case control := <-t.cancels:
			item.frame = control
		default:
			select {
			case <-t.done:
				return
			case control := <-t.cancels:
				item.frame = control
			case item = <-t.writes:
			}
		}
		var err error
		if item.ctx == nil || item.ctx.Err() == nil {
			err = WriteFrame(t.writer, item.frame)
		}
		if item.bytes != 0 {
			t.mu.Lock()
			t.queuedBytes -= item.bytes
			t.mu.Unlock()
		}
		if err != nil {
			t.fail(err)
			return
		}
	}
}
func (t *stdioTransport) read() {
	for {
		frame := &apipb.PluginBridgeFrame{}
		if err := ReadFrame(t.reader, frame); err != nil {
			t.fail(err)
			return
		}
		if frame.GetRequestId() == "" || len(frame.GetRequestId()) > 160 || frame.GetResult() == nil {
			t.fail(errors.New("invalid plugin bridge response"))
			return
		}
		t.mu.Lock()
		pending, ok := t.pending[frame.RequestId]
		t.mu.Unlock()
		if ok {
			if frame.GetViaEndpointId() != "" && frame.GetViaEndpointId() != pending.endpoint {
				t.fail(errors.New("plugin bridge response endpoint mismatch"))
				return
			}
			select {
			case pending.response <- stdioResponse{value: frame.GetResult()}:
			default:
			}
		}
	}
}
func (t *stdioTransport) fail(err error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.failure = err
	close(t.done)
	for _, pending := range t.pending {
		select {
		case pending.response <- stdioResponse{err: err}:
		default:
		}
	}
	t.mu.Unlock()
	// Closing owned pipes also releases a blocked reader or writer. Plain Reader /
	// Writer embeddings must provide their own cancellation-aware I/O contract.
	if closer, ok := t.input.(io.Closer); ok {
		_ = closer.Close()
	}
	if closer, ok := t.writer.(io.Closer); ok {
		_ = closer.Close()
	}
}

// ServeStdio transports commands to the daemon executor, never dispatching UI.
func ServeStdio(ctx context.Context, input io.Reader, output io.Writer, execute Executor) error {
	return ServeStdioRouted(ctx, input, output, func(ctx context.Context, _ string, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
		return execute(ctx, command)
	})
}

type RoutedExecutor func(context.Context, string, *apipb.PluginCommand) (*apipb.PluginResult, error)
type bridgeActive struct {
	cancel   context.CancelFunc
	endpoint string
}

// ServeStdioRouted continuously reads control frames even with 32 busy workers.
// It owns closable input/output for its lifetime and cancels calls on EOF/exit.
func ServeStdioRouted(ctx context.Context, input io.Reader, output io.Writer, execute RoutedExecutor) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var mu sync.Mutex
	active := map[string]bridgeActive{}
	var workers sync.WaitGroup
	replies := make(chan *apipb.PluginBridgeFrame, 64)
	var outputBytes int
	var outputMu sync.Mutex
	failure := make(chan error, 1)
	fail := func(err error) {
		select {
		case failure <- err:
		default:
		}
		cancel()
	}
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-runCtx.Done():
			if closer, ok := input.(io.Closer); ok {
				_ = closer.Close()
			}
			if closer, ok := output.(io.Closer); ok {
				_ = closer.Close()
			}
		case <-stopped:
		}
	}()
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for {
			select {
			case <-runCtx.Done():
				return
			case reply := <-replies:
				err := WriteFrame(output, reply)
				outputMu.Lock()
				outputBytes -= proto.Size(reply) + 10
				outputMu.Unlock()
				if err != nil {
					fail(err)
					return
				}
			}
		}
	}()
	defer func() { cancel(); workers.Wait(); <-writerDone }()
	respond := func(request *apipb.PluginBridgeFrame, result *apipb.PluginResult) {
		reply := &apipb.PluginBridgeFrame{RequestId: request.RequestId, ViaEndpointId: request.ViaEndpointId, Payload: &apipb.PluginBridgeFrame_Result{Result: result}}
		size := proto.Size(reply) + 10
		outputMu.Lock()
		if proto.Size(reply) > MaxFrameBytes || outputBytes+size > maxStdioOutputBytes {
			outputMu.Unlock()
			fail(errors.New("plugin bridge response byte budget exhausted"))
			return
		}
		outputBytes += size
		outputMu.Unlock()
		select {
		case replies <- reply:
		case <-runCtx.Done():
		default:
			fail(errors.New("plugin bridge response queue exhausted"))
		}
	}
	reject := func(request *apipb.PluginBridgeFrame, code, message string) {
		respond(request, &apipb.PluginResult{Result: &apipb.PluginResult_Error{Error: &apipb.PluginError{Code: code, Message: message}}})
	}
	reader := bufio.NewReader(input)
	for {
		request := &apipb.PluginBridgeFrame{}
		if err := ReadFrame(reader, request); err != nil {
			select {
			case failureErr := <-failure:
				return failureErr
			default:
			}
			if errors.Is(err, io.EOF) {
				return nil
			}
			if runCtx.Err() != nil {
				return runCtx.Err()
			}
			return err
		}
		if request.GetRequestId() == "" || len(request.GetRequestId()) > 160 || len(request.GetViaEndpointId()) > 160 {
			return errors.New("invalid plugin bridge request identity")
		}
		if id := request.GetCancelRequestId(); id != "" {
			if id != request.RequestId {
				return errors.New("cancellation identity mismatch")
			}
			mu.Lock()
			call, exists := active[id]
			mu.Unlock()
			if exists {
				if call.endpoint != request.ViaEndpointId {
					return errors.New("cancellation endpoint mismatch")
				}
				call.cancel()
			}
			continue
		}
		if request.GetCommand() == nil {
			return errors.New("invalid plugin bridge command")
		}
		now := time.Now()
		deadline := time.UnixMilli(request.GetDeadlineUnixMillis())
		if request.GetDeadlineUnixMillis() == 0 {
			deadline = now.Add(30 * time.Second)
		}
		if !deadline.After(now) {
			reject(request, "DEADLINE_EXCEEDED", "bridge deadline expired")
			continue
		}
		if deadline.After(now.Add(2 * time.Minute)) {
			reject(request, "INVALID_REQUEST", "bridge deadline exceeds two minutes")
			continue
		}
		mu.Lock()
		if _, exists := active[request.RequestId]; exists {
			mu.Unlock()
			return errors.New("duplicate active plugin bridge request ID")
		}
		if len(active) >= 32 {
			mu.Unlock()
			reject(request, "RESOURCE_EXHAUSTED", "plugin bridge workers are busy")
			continue
		}
		requestCtx, requestCancel := context.WithDeadline(runCtx, deadline)
		active[request.RequestId] = bridgeActive{cancel: requestCancel, endpoint: request.ViaEndpointId}
		mu.Unlock()
		workers.Add(1)
		go func() {
			defer workers.Done()
			defer requestCancel()
			var result *apipb.PluginResult
			err := requestCtx.Err()
			if err == nil {
				result, err = execute(requestCtx, request.ViaEndpointId, request.GetCommand())
			}
			if err != nil {
				code := "TRANSPORT_ERROR"
				if errors.Is(err, context.Canceled) {
					code = "CANCELLED"
				}
				if errors.Is(err, context.DeadlineExceeded) {
					code = "DEADLINE_EXCEEDED"
				}
				result = &apipb.PluginResult{Result: &apipb.PluginResult_Error{Error: &apipb.PluginError{Code: code, Message: err.Error()}}}
			}
			if result == nil {
				result = &apipb.PluginResult{Result: &apipb.PluginResult_Error{Error: &apipb.PluginError{Code: "INTERNAL", Message: "plugin executor returned no result"}}}
			}
			mu.Lock()
			delete(active, request.RequestId)
			mu.Unlock()
			respond(request, result)
		}()
	}
}
