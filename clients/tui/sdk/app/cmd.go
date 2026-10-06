package app

import (
	"time"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// None is the empty command. A nil Cmd is accepted everywhere a Cmd is.
var None Cmd

// Batch runs several commands and feeds every non-nil result back into the
// loop. Sub-commands run concurrently, so Batch starts independent effects
// from one Update.
func Batch(cmds ...Cmd) Cmd {
	alive := make([]Cmd, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd != nil {
			alive = append(alive, cmd)
		}
	}
	if len(alive) == 0 {
		return nil
	}
	return func() Msg { return batchMsg(alive) }
}

// RunCmd executes one command tree outside the event loop and returns every
// message it produced, expanding Batch sub-commands depth-first. Run owns
// command execution in a real program; this helper exists for tests and tools
// that drive a Model directly (a Cmd returned by Update can contain a Batch,
// which the engine would otherwise expand).
func RunCmd(cmd Cmd) []Msg {
	return runCmd(cmd)
}

func runCmd(cmd Cmd) []Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(batchMsg); ok {
		var out []Msg
		for _, sub := range batch {
			out = append(out, runCmd(sub)...)
		}
		return out
	}
	if msg == nil {
		return nil
	}
	return []Msg{msg}
}

// Quit stops the program: Run returns ErrQuit once the current batch is
// done.
func Quit() Cmd { return func() Msg { return quitMsg{} } }

// Tick delivers TickMsg{At: time.Now()} after d. It uses program-side time
// only: the host has no timer, so Tick is a convenience, not a protocol
// event. A non-positive d fires immediately.
func Tick(d time.Duration) Cmd {
	return func() Msg {
		if d > 0 {
			time.Sleep(d)
		}
		return TickMsg{At: time.Now()}
	}
}

// SetKeys changes the VIEW routing declaration used by the following
// commits. Commands run asynchronously, so the change applies from the next
// batch; use Program.Keys for the declaration of the first view.
func SetKeys(keys sdk.Keys) Cmd {
	copied := sdk.Keys{Claim: append([]string(nil), keys.Claim...), All: keys.All}
	return func() Msg { return keysMsg{keys: copied} }
}

// Emit sends one RESULT and decodes its response. decode receives the
// MethodData payload bytes (response.data.access_result, the opaque result a
// method such as access.call returns), or nil when the host sent no data; it
// runs for failure responses too, where the payload is nil. The slice aliases
// the response message, so copy it to keep it. A transport failure is
// delivered as ErrorMsg. A nil decode delivers the full ResponseMsg instead.
//
// If the host reconnects while the call is in flight, Run discards the
// eventual result; the command goroutine ends when the host answers.
func Emit(client *sdk.Client, method string, params *pb.MethodParams, decode func([]byte) Msg) Cmd {
	return func() Msg {
		respCh := make(chan *pb.Response, 1)
		if _, err := client.Emit(method, params, func(resp *pb.Response) { respCh <- resp }); err != nil {
			return ErrorMsg{Err: err}
		}
		resp := <-respCh
		if decode == nil {
			return ResponseMsg{Response: resp}
		}
		return decode(resp.GetData().GetAccessResult())
	}
}

// EmitResponse is Emit without decoding: Update receives the full
// ResponseMsg.
func EmitResponse(client *sdk.Client, method string, params *pb.MethodParams) Cmd {
	return Emit(client, method, params, nil)
}

// SendStream sends one STREAM frame (data, ack, close or cancel) for a
// stream this program opened. A transport failure is delivered as ErrorMsg.
func SendStream(client *sdk.Client, frame *pb.StreamFrame) Cmd {
	return func() Msg {
		if err := client.SendStream(frame); err != nil {
			return ErrorMsg{Err: err}
		}
		return nil
	}
}
