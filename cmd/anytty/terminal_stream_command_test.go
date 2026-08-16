package main

import (
	"bytes"
	"context"
	"testing"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/wire"
)

func TestReceiveTerminalPTYStreamWritesExactBytes(t *testing.T) {
	raw := []byte{0x00, 0xff, 'A', 0x1b, '[', 'm', '\r', '\n'}
	stream := &terminalStreamFixture{frames: []protocol.StreamFrame{
		{Type: wire.TypePTYOutput, Payload: raw},
		{Type: wire.TypeClosed, Payload: wire.EncodeClosedPayload(23)},
	}}
	var output bytes.Buffer
	if err := receiveTerminalPTYStream(context.Background(), stream, &output); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), raw) {
		t.Fatalf("raw PTY output=%x want=%x", output.Bytes(), raw)
	}
}

func TestReceiveTerminalPTYStreamReportsSyncLoss(t *testing.T) {
	stream := &terminalStreamFixture{frames: []protocol.StreamFrame{{
		Type: wire.TypeSyncLost, Payload: wire.EncodeSyncLostPayload(4096),
	}}}
	err := receiveTerminalPTYStream(context.Background(), stream, &bytes.Buffer{})
	if cliExitCode(err) != 8 {
		t.Fatalf("sync loss error=%v exit=%d", err, cliExitCode(err))
	}
}

type terminalStreamFixture struct {
	frames []protocol.StreamFrame
}

func (stream *terminalStreamFixture) Receive(context.Context) (uint8, []byte, error) {
	if len(stream.frames) == 0 {
		return 0, nil, context.Canceled
	}
	frame := stream.frames[0]
	stream.frames = stream.frames[1:]
	return frame.Type, append([]byte(nil), frame.Payload...), nil
}

func (*terminalStreamFixture) Send(context.Context, uint8, []byte) error { return nil }
func (*terminalStreamFixture) Close() error                              { return nil }

var _ clientruntime.ResourceStream = (*terminalStreamFixture)(nil)
