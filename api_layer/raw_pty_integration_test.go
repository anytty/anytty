package apilayer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	corev2 "github.com/anytty/anytty/core"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
)

func TestTerminalAttachmentStreamsRawPTYOutputAndAcceptsRawInput(t *testing.T) {
	factory := &rawPTYTestProcessFactory{spawned: make(chan *rawPTYTestProcess, 1)}
	server := corev2.NewServer(
		corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory),
		corev2.WithProcessFactory(factory),
	)
	defer func() { _ = server.Shutdown(context.Background()) }()
	application, client, closeClient := newProtoTransportClient(t, server, nil, 1)
	defer closeClient()

	createProtoTestTerminal(t, application, "term-raw-pty")
	process := <-factory.spawned
	attached, err := application.TerminalAttach(context.Background(), &apipb.TerminalAttachCommand{
		Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-raw-pty"},
		Mode:     apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId: "raw-surface", ViewId: "raw-view",
	})
	if err != nil {
		t.Fatal(err)
	}
	resource := attached.GetAttachment().GetResource()
	channel, ok := client.ApplicationAttachmentChannel(resource)
	if !ok {
		t.Fatal("terminal attachment was not bound to a resource channel")
	}
	frames, stop := client.Stream(channel)
	defer stop()
	if err := client.SendAttachmentReady(channel); err != nil {
		t.Fatal(err)
	}
	ready := waitRawPTYFrame(t, frames)
	if ready.Type != wire.TypeStreamReady {
		t.Fatalf("raw PTY ready frame type=%d want=%d", ready.Type, wire.TypeStreamReady)
	}

	rawOutput := []byte{0x00, 0xff, 'A', 0x1b, '[', '3', '1', 'm', '\r', '\n'}
	process.emit(rawOutput)
	frame := waitRawPTYFrame(t, frames)
	if frame.Type != wire.TypePTYOutput || !bytes.Equal(frame.Payload, rawOutput) {
		t.Fatalf("raw PTY output frame type=%d payload=%x want=%x", frame.Type, frame.Payload, rawOutput)
	}

	rawInput := []byte{0x00, 0xfe, 0x1b, '[', 'A'}
	if err := application.TerminalInput(context.Background(), &apipb.TerminalInputCommand{
		Attachment: resource, Data: rawInput,
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-process.inputs:
		if !bytes.Equal(got, rawInput) {
			t.Fatalf("PTY input=%x want=%x", got, rawInput)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for raw PTY input")
	}

	if err := client.SendAttachmentStreamClose(channel); err != nil {
		t.Fatal(err)
	}
	stop()
	frames, stopReopened := client.Stream(channel)
	defer stopReopened()
	if err := client.SendAttachmentReady(channel); err != nil {
		t.Fatal(err)
	}
	ready = waitRawPTYFrame(t, frames)
	if ready.Type != wire.TypeStreamReady {
		t.Fatalf("reopened raw PTY ready frame type=%d want=%d", ready.Type, wire.TypeStreamReady)
	}
	reopenedOutput := []byte{0xfe, 'r', 'e', 'o', 'p', 'e', 'n'}
	process.emit(reopenedOutput)
	frame = waitRawPTYFrame(t, frames)
	if frame.Type != wire.TypePTYOutput || !bytes.Equal(frame.Payload, reopenedOutput) {
		t.Fatalf("reopened raw PTY frame type=%d payload=%x want=%x", frame.Type, frame.Payload, reopenedOutput)
	}

	process.exit(23)
	closed := waitRawPTYFrame(t, frames)
	if closed.Type != wire.TypeClosed {
		t.Fatalf("terminal exit frame type=%d want=%d", closed.Type, wire.TypeClosed)
	}
	exitCode, err := wire.DecodeClosedPayload(closed.Payload)
	if err != nil || exitCode != 23 {
		t.Fatalf("terminal exit code=%d err=%v", exitCode, err)
	}
}

func waitRawPTYFrame(t *testing.T, frames <-chan internalprotocol.StreamFrame) internalprotocol.StreamFrame {
	t.Helper()
	select {
	case frame, ok := <-frames:
		if !ok {
			t.Fatal("raw PTY stream closed")
		}
		return frame
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for raw PTY stream")
		return internalprotocol.StreamFrame{}
	}
}

type rawPTYTestProcessFactory struct {
	spawned chan *rawPTYTestProcess
}

func (factory *rawPTYTestProcessFactory) Spawn(context.Context, corev2.ProcessSpec) (corev2.TerminalProcess, error) {
	process := &rawPTYTestProcess{
		output: make(chan []byte, 16), wait: make(chan corev2.ProcessExit, 1), inputs: make(chan []byte, 16),
	}
	factory.spawned <- process
	return process, nil
}

type rawPTYTestProcess struct {
	output chan []byte
	wait   chan corev2.ProcessExit
	inputs chan []byte
	once   sync.Once
}

func (process *rawPTYTestProcess) Input(data []byte) error {
	process.inputs <- append([]byte(nil), data...)
	return nil
}

func (process *rawPTYTestProcess) Resize(corev2.Size) error { return nil }
func (process *rawPTYTestProcess) Output() <-chan []byte    { return process.output }
func (process *rawPTYTestProcess) CancelOutput()            {}
func (process *rawPTYTestProcess) Wait() <-chan corev2.ProcessExit {
	return process.wait
}
func (process *rawPTYTestProcess) Kill() error {
	process.finish(corev2.ProcessExit{Code: -1, Err: errors.New("killed")})
	return nil
}
func (process *rawPTYTestProcess) Close() error {
	process.finish(corev2.ProcessExit{Code: -1, Err: io.EOF})
	return nil
}
func (process *rawPTYTestProcess) emit(data []byte) {
	process.output <- append([]byte(nil), data...)
}
func (process *rawPTYTestProcess) exit(code int) {
	process.finish(corev2.ProcessExit{Code: code})
}
func (process *rawPTYTestProcess) finish(exit corev2.ProcessExit) {
	process.once.Do(func() {
		close(process.output)
		process.wait <- exit
		close(process.wait)
	})
}
