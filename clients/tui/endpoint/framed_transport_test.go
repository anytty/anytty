package endpoint

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/anytty/anytty/proto/access/wire"
	"github.com/klauspost/compress/zstd"
)

// The tcp transport mirrors shared/transport/unix framing byte for byte:
// packet kind + big-endian uint32 length over a zstd stream, with logical
// frames split at the 64KiB packet limit. Keeping the exact framing is what
// lets a tcp endpoint terminate at a daemon transport through an ssh -L
// TCP->socket forward or a byte-transparent bridge.
const (
	packetFrame byte = iota
	packetFragmentStart
	packetFragmentContinue
	packetFragmentEnd
)

const (
	maxPacketPayloadSize = 64 << 10
	maxLogicalFrameSize  = wire.MaxEncodedFrameSize
	encoderWindowSize    = 128 << 10
	decoderMaxWindow     = 256 << 10
)

// framedTransport is one bidirectional daemon transport over an arbitrary
// net.Conn (tcp in production, bridge/fake peers in tests). Send is
// serialized: the manager already serializes control and input writes, and
// this lock keeps the packet stream well formed even if a caller does not.
type framedTransport struct {
	conn net.Conn

	sendMu sync.Mutex
	zstdW  *zstd.Encoder
	zstdR  *zstd.Decoder

	closeOnce sync.Once
	closeErr  error
}

// dialTCPTransport opens the framed daemon transport over tcp to address.
func dialTCPTransport(ctx context.Context, address string) (Transport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return newFramedTransport(conn)
}

// newFramedTransport wraps an already connected stream. The zstd parameters
// match shared/transport/unix so both ends can decode each other.
func newFramedTransport(conn net.Conn) (*framedTransport, error) {
	if conn == nil {
		return nil, io.EOF
	}
	writer, err := zstd.NewWriter(
		conn,
		zstd.WithEncoderConcurrency(1),
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithWindowSize(encoderWindowSize),
		zstd.WithLowerEncoderMem(true),
	)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	reader, err := zstd.NewReader(
		conn,
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderLowmem(true),
		zstd.WithDecoderMaxWindow(decoderMaxWindow),
	)
	if err != nil {
		_ = writer.Close()
		_ = conn.Close()
		return nil, err
	}
	return &framedTransport{conn: conn, zstdW: writer, zstdR: reader}, nil
}

// Send writes one complete protocol frame, fragmenting above the packet cap.
func (t *framedTransport) Send(frame []byte) error {
	if len(frame) > maxLogicalFrameSize {
		return wire.ErrFrameTooLarge
	}
	if t == nil || t.zstdW == nil {
		return io.EOF
	}
	t.sendMu.Lock()
	defer t.sendMu.Unlock()
	if err := t.writeLogicalFrame(frame); err != nil {
		return err
	}
	return t.zstdW.Flush()
}

// Recv reads one complete protocol frame, reassembling fragments.
func (t *framedTransport) Recv() ([]byte, error) {
	if t == nil || t.zstdR == nil {
		return nil, io.EOF
	}
	for {
		kind, payload, err := t.readPacket()
		if err != nil {
			return nil, err
		}
		switch kind {
		case packetFrame:
			return payload, nil
		case packetFragmentStart:
			buf := append([]byte(nil), payload...)
			for {
				nextKind, nextPayload, err := t.readPacket()
				if err != nil {
					return nil, err
				}
				switch nextKind {
				case packetFragmentContinue:
					if len(nextPayload) > maxLogicalFrameSize-len(buf) {
						return nil, wire.ErrFrameTooLarge
					}
					buf = append(buf, nextPayload...)
				case packetFragmentEnd:
					if len(nextPayload) > maxLogicalFrameSize-len(buf) {
						return nil, wire.ErrFrameTooLarge
					}
					return append(buf, nextPayload...), nil
				default:
					return nil, fmt.Errorf("endpoint/tcp: unexpected packet kind %d during fragmented frame", nextKind)
				}
			}
		default:
			return nil, fmt.Errorf("endpoint/tcp: unexpected packet kind %d", kind)
		}
	}
}

// Close releases the connection and the send-side zstd stream. It is
// idempotent. The decoder is intentionally not closed: closing it while Recv
// is blocked would race with the reader, and closing the connection already
// unblocks Recv with an error (the same contract as shared/transport/unix).
func (t *framedTransport) Close() error {
	if t == nil {
		return nil
	}
	t.closeOnce.Do(func() {
		t.closeErr = t.conn.Close()
		t.sendMu.Lock()
		if t.zstdW != nil {
			_ = t.zstdW.Close()
		}
		t.sendMu.Unlock()
	})
	return t.closeErr
}

func (t *framedTransport) writeLogicalFrame(frame []byte) error {
	if len(frame) <= maxPacketPayloadSize {
		return t.writePacket(packetFrame, frame)
	}
	offset := 0
	for offset < len(frame) {
		end := offset + maxPacketPayloadSize
		if end > len(frame) {
			end = len(frame)
		}
		kind := packetFragmentContinue
		switch {
		case offset == 0:
			kind = packetFragmentStart
		case end == len(frame):
			kind = packetFragmentEnd
		}
		if err := t.writePacket(kind, frame[offset:end]); err != nil {
			return err
		}
		offset = end
	}
	return nil
}

func (t *framedTransport) writePacket(kind byte, payload []byte) error {
	if len(payload) > maxPacketPayloadSize {
		return fmt.Errorf("endpoint/tcp: packet too large: %d > %d", len(payload), maxPacketPayloadSize)
	}
	var header [5]byte
	header[0] = kind
	binary.BigEndian.PutUint32(header[1:], uint32(len(payload)))
	if _, err := t.zstdW.Write(header[:]); err != nil {
		return err
	}
	_, err := t.zstdW.Write(payload)
	return err
}

func (t *framedTransport) readPacket() (byte, []byte, error) {
	var header [5]byte
	if _, err := io.ReadFull(t.zstdR, header[:]); err != nil {
		return 0, nil, err
	}
	kind := header[0]
	if !validPacketKind(kind) {
		return 0, nil, fmt.Errorf("endpoint/tcp: unexpected packet kind %d", kind)
	}
	n := binary.BigEndian.Uint32(header[1:])
	if n > maxPacketPayloadSize {
		return 0, nil, fmt.Errorf("endpoint/tcp: packet too large: %d > %d", n, maxPacketPayloadSize)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(t.zstdR, buf); err != nil {
		return 0, nil, err
	}
	return kind, buf, nil
}

func validPacketKind(kind byte) bool {
	return kind == packetFrame ||
		kind == packetFragmentStart ||
		kind == packetFragmentContinue ||
		kind == packetFragmentEnd
}

var _ Transport = (*framedTransport)(nil)
