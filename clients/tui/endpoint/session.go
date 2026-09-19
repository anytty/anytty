package endpoint

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
)

// sessionConn is one live daemon connection. The production implementation is
// sharedConn (client/runtime + client/adapter/protocol, see shared_session.go);
// tests inject a raw-wire fake. Manager and RemotePTY only depend on this
// interface, so the connection implementation stays swappable and tui2 owns no
// second dialer (tui2/docs/CLIENT_SHARING.zh-CN.md §3).
type sessionConn interface {
	list(ctx context.Context) ([]*apipb.TerminalInfo, error)
	defaults(ctx context.Context) (*apipb.TerminalDefaults, error)
	create(ctx context.Context, spec *apipb.TerminalCreateSpec) (*apipb.TerminalInfo, error)
	kill(ctx context.Context, id string) error
	remove(ctx context.Context, id string) error
	restart(ctx context.Context, id string) error
	liveScreen(ctx context.Context, id string, observed uint64) (*apipb.NativeScreenResult, error)
	openAttachment(ctx context.Context, id, surface, view string, cols, rows int) (*attachment, error)
	startStream(ctx context.Context, att *attachment, timeout time.Duration) error
	input(ctx context.Context, att *attachment, data []byte) error
	resize(ctx context.Context, att *attachment, cols, rows int, takeOwnership bool, expectedEpoch uint64) (*apipb.TerminalResizeResult, error)
	detach(ctx context.Context, att *attachment) error
	close() error
	Done() <-chan struct{}
	Err() error
}

// SessionDialer opens one daemon session connection. The production value is
// dialSharedSession; tests inject a raw-wire fake.
type SessionDialer func(ctx context.Context, cfg Config) (sessionConn, error)

// ErrEndpointClosed means the daemon connection is closed.
var (
	ErrEndpointClosed = errors.New("endpoint: connection closed")
	ErrStreamSyncLost = errors.New("endpoint: attachment stream lost sync")
)

// APIError is a typed application error from the daemon (ApiError code plus
// the daemon message). It is the readable error surface of the manager.
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	if e == nil {
		return "endpoint request failed"
	}
	return fmt.Sprintf("terminal pool error %d: %s", e.Code, e.Message)
}

// apiErrorFromProto converts a ResultEnvelope error into an APIError.
func apiErrorFromProto(apiError *apipb.ApiError) error {
	if apiError == nil {
		return nil
	}
	message := strings.TrimSpace(apiError.GetMessage())
	if codeName := apiError.GetCode().String(); message == "" {
		message = codeName
	} else if codeName != "" {
		message = codeName + ": " + message
	}
	return &APIError{Code: int(apiError.GetCode()), Message: message}
}

// isNotFound recognizes the daemon NOT_FOUND error across the raw test client
// and the shared client/runtime error projection.
func isNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == int(apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND)
	}
	var runtimeErr *clientruntime.Error
	if errors.As(err, &runtimeErr) {
		return runtimeErr.Code == clientruntime.ErrorNotFound
	}
	return false
}

// streamFrame is one attachment-channel frame.
type streamFrame struct {
	typ     uint8
	payload []byte
}

// streamQueue is the per-attachment frame queue. A consumer that falls this
// far behind is marked sync-lost instead of blocking the connection.
const streamQueue = 256

// stream is one attachment channel demultiplexer with a small front buffer so
// a peeked frame can be handled before newer frames.
type stream struct {
	mu     sync.Mutex
	frames chan streamFrame
	front  []streamFrame
	closed bool
	err    error
}

func newStream() *stream {
	return &stream{frames: make(chan streamFrame, streamQueue)}
}

func (s *stream) push(typ uint8, payload []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	select {
	case s.frames <- streamFrame{typ: typ, payload: payload}:
	default:
		s.closeLocked(ErrStreamSyncLost)
	}
}

// pushFront re-queues a peeked frame ahead of every queued frame.
func (s *stream) pushFront(frame streamFrame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.front = append(s.front, frame)
}

func (s *stream) close(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeLocked(err)
}

func (s *stream) closeLocked(err error) {
	if s.closed {
		return
	}
	s.closed = true
	s.err = err
	close(s.frames)
}

// recv returns the next frame; a closed channel reports the stream error or
// io.EOF for a clean daemon close.
func (s *stream) recv(ctx context.Context) (streamFrame, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	if len(s.front) > 0 {
		frame := s.front[0]
		s.front = s.front[1:]
		s.mu.Unlock()
		return frame, nil
	}
	s.mu.Unlock()
	select {
	case <-ctx.Done():
		return streamFrame{}, ctx.Err()
	case frame, ok := <-s.frames:
		if ok {
			return frame, nil
		}
		s.mu.Lock()
		err := s.err
		s.mu.Unlock()
		if err == nil {
			err = io.EOF
		}
		return streamFrame{}, err
	}
}

// attachment is one live terminal attachment stream. client is the session
// that owns it; resource/stream are meaningful for both implementations.
type attachment struct {
	client   sessionConn
	terminal string
	resource *apipb.ResourceHandle
	channel  uint16
	stream   *stream
	surface  string
	view     string
	mode     apipb.AttachmentMode
	policy   apipb.ResizePolicy
	size     *apipb.TerminalSize
	epoch    uint64
	// stop releases the shared resource stream pump (nil for the raw client).
	stop func()
}

func decodeClosed(payload []byte) int {
	code, err := wire.DecodeClosedPayload(payload)
	if err != nil {
		return -1
	}
	return code
}
