// Package app is a lightweight Elm-style layer over the layout-program SDK.
// A Model owns the state, Update handles one Msg, View returns the full box
// tree, and Run coalesces a burst of host events into exactly one Commit per
// batch:
//
//	type model struct{ n int }
//
//	func (m *model) Init() app.Cmd            { return app.SetKeys(sdk.Keys{All: true}) }
//	func (m *model) Update(msg app.Msg) app.Cmd { m.n++; return app.None }
//	func (m *model) View() *pb.Box             { return sdk.Text(fmt.Sprint(m.n)).Build() }
//
//	func main() {
//		client := sdk.New(os.Stdin, os.Stdout, sdk.Handlers{})
//		if err := app.Run(client, &model{}); err != nil && !errors.Is(err, app.ErrQuit) {
//			log.Fatal(err)
//		}
//	}
//
// The package builds against the standard library, the SDK and the wire
// protocol only (clients/tui/sdk/deps_test.go).
package app

import (
	"errors"
	"time"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Msg is one event delivered to Update. The loop wraps every host event in
// the concrete types below, so a model can type-switch on them.
type Msg any

// Cmd is one deferred side effect. Run executes the Cmd returned by Init and
// Update on a fresh goroutine and feeds the returned Msg back into the
// event loop; a nil Cmd (None) does nothing.
type Cmd func() Msg

// Model is the application state machine.
//
// Init runs once per HELLO epoch: on the first HELLO and again after every
// host restart (a new epoch), together with the optional Resetter hook.
// Update handles messages in arrival order; View returns the box tree that
// Run commits at the end of a batch.
//
// Update must only mutate model state and return a Cmd. Run owns Commit and
// calls View exactly once per batch, so a model must never call
// client.Commit directly (that would double-write the protocol and skip the
// loop's coalescing).
type Model interface {
	Init() Cmd
	Update(Msg) Cmd
	View() *pb.Box
}

// Cleaner is an optional Model extension: when Dirty reports false after a
// batch, Run suppresses that batch's commit. A new HELLO epoch always
// commits, so a restarted host still receives a view.
type Cleaner interface {
	Dirty() bool
}

// Resetter is an optional Model extension: on every new HELLO epoch Run
// calls Reset(epoch) before Init, so the model can drop connection-scoped
// state (focus, in-flight request bookkeeping). Messages and commands from
// the previous epoch are discarded by the loop.
type Resetter interface {
	Reset(epoch uint64)
}

// HelloMsg is delivered to Update for every HELLO, including restarts.
type HelloMsg struct{ Hello *pb.Hello }

// SourcesMsg carries a sources snapshot.
type SourcesMsg struct{ Items []*pb.Source }

// KeyMsg carries one key event that no keymap binding consumed. Models that
// configure a Keymap only see unbound keys this way.
type KeyMsg struct{ Key *pb.KeyEvent }

// PasteMsg carries one paste chunk.
type PasteMsg struct{ Paste *pb.PasteEvent }

// MouseMsg carries one mouse action.
type MouseMsg struct{ Mouse *pb.MouseEvent }

// WheelMsg carries one wheel delta.
type WheelMsg struct{ Wheel *pb.WheelEvent }

// ResizeMsg carries a viewport resize, already converted to int.
type ResizeMsg struct{ Cols, Rows int }

// NoticeMsg carries a host notice.
type NoticeMsg struct{ Level, Message string }

// ComponentMsg carries one component event. When the focus stack's current
// entry implements ComponentHandler the event is routed there first.
type ComponentMsg struct{ Component *pb.ComponentEvent }

// ViewRejectedMsg reports that the host rejected a committed view.
type ViewRejectedMsg struct {
	Epoch  uint64
	Rev    uint64
	Reason string
}

// ResponseMsg carries every host RESPONSE. Responses matched by an Emit Cmd
// are also decoded and returned by that command.
type ResponseMsg struct{ Response *pb.Response }

// StreamMsg carries one STREAM frame of a stream this program opened.
type StreamMsg struct{ Frame *pb.StreamFrame }

// TickMsg is delivered by the Tick Cmd after its program-side delay. No host
// timer is involved.
type TickMsg struct{ At time.Time }

// ErrorMsg reports that a Cmd could not complete (Emit/SendStream transport
// error). Protocol-level failures stop Run with that error instead.
type ErrorMsg struct{ Err error }

// ErrQuit is returned by Program.Run after a Quit Cmd.
var ErrQuit = errors.New("tui2/sdk/app: quit")

// Program binds a Model to a Client with optional routing configuration.
type Program struct {
	Client *sdk.Client
	Model  Model

	// Keymap maps key events to messages before Update sees them. Nil sends
	// every key event to Update as a KeyMsg.
	Keymap *Keymap

	// Focus routes component events to the focused entry. Nil means an empty
	// stack, so every component event goes to Update as a ComponentMsg.
	Focus *Focus

	// Keys is the initial VIEW routing declaration. Nil means
	// sdk.Keys{All: true}, so the program receives every key event by
	// default; the SetKeys Cmd changes it at runtime.
	Keys *sdk.Keys

	// ForceFullView disables the automatic VIEW_DELTA path so every batch
	// commits a full VIEW snapshot. It exists for debugging (and for hosts
	// whose delta application is suspect); the default is false, so a host
	// that advertises features["view_delta"] gets deltas.
	ForceFullView bool

	// Memo, when non-nil, scopes Model.View to one memo frame per committed
	// batch: BeginFrame before View and EndFrame after the commit. A model
	// that builds its subtrees through the same Memo keeps their *pb.Box
	// pointers stable across frames, so the core diff short-circuits on
	// pointer identity and costs O(changed) instead of walking the whole
	// tree. Nil disables memoization (zero behavior change).
	Memo *Memo
}

// Run is the convenience entry point: m runs against client with no keymap,
// an empty focus stack and Keys{All: true}. It blocks until Quit or until
// the host closes the connection. Quit makes Run return ErrQuit; a clean
// transport end returns nil.
func Run(client *sdk.Client, m Model) error {
	return (&Program{Client: client, Model: m}).Run()
}

// Run drives the program: it installs the event handlers, starts the read
// loop, and returns when the program quits or the transport ends. Run does
// not close the transport, so a blocked read loop goroutine may outlive it.
//
// On a new HELLO epoch (host restart) the loop resets the focus stack,
// restores Program.Keys, calls the optional Resetter, re-runs Init and drops
// every command result started in the old epoch, so the model restarts from
// a connection-clean state.
func (p *Program) Run() error {
	if p == nil || p.Client == nil || p.Model == nil {
		return errors.New("tui2/sdk/app: nil client or model")
	}
	eng := newEngine(p.Client, p.Model, p.Keymap, p.Focus, p.initialKeys(), p.ForceFullView, p.Memo)
	p.Client.SetHandlers(sdk.Handlers{
		Hello:     func(h *pb.Hello) { eng.Post(HelloMsg{Hello: h}) },
		Sources:   func(items []*pb.Source) { eng.Post(SourcesMsg{Items: items}) },
		Key:       func(ev *pb.KeyEvent) { eng.Post(KeyMsg{Key: ev}) },
		Paste:     func(ev *pb.PasteEvent) { eng.Post(PasteMsg{Paste: ev}) },
		Mouse:     func(ev *pb.MouseEvent) { eng.Post(MouseMsg{Mouse: ev}) },
		Wheel:     func(ev *pb.WheelEvent) { eng.Post(WheelMsg{Wheel: ev}) },
		Resize:    func(cols, rows int) { eng.Post(ResizeMsg{Cols: cols, Rows: rows}) },
		Notice:    func(level, message string) { eng.Post(NoticeMsg{Level: level, Message: message}) },
		Component: func(ev *pb.ComponentEvent) { eng.Post(ComponentMsg{Component: ev}) },
		ViewRejected: func(epoch, rev uint64, reason string) {
			eng.Post(ViewRejectedMsg{Epoch: epoch, Rev: rev, Reason: reason})
		},
		Response: func(resp *pb.Response) { eng.Post(ResponseMsg{Response: resp}) },
		Stream:   func(frame *pb.StreamFrame) { eng.Post(StreamMsg{Frame: frame}) },
	})
	go func() { eng.Post(loopDoneMsg{err: p.Client.Loop()}) }()
	return eng.run()
}

// initialKeys resolves the Program.Keys default.
func (p *Program) initialKeys() sdk.Keys {
	if p.Keys == nil {
		return sdk.Keys{All: true}
	}
	return sdk.Keys{Claim: append([]string(nil), p.Keys.Claim...), All: p.Keys.All}
}
