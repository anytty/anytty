package conformance

import (
	"fmt"
	"io"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// RefProgram is the reference conformance program every SDK must be able to
// express (README.zh-CN.md). It drives the typed handlers of an SDK and
// renders one line per observed event so the runner can check decoding,
// request/response correlation and epoch handling without inspecting the
// candidate's internals.
type RefProgram struct {
	client *sdk.Client
	lines  []string
	pane   bool
}

// NewRefFactory builds the in-process factory for the official Go SDK.
func NewRefFactory() InProcessFactory {
	return InProcessFactory{Run: func(in io.Reader, out io.Writer) error {
		return (&RefProgram{}).Run(in, out)
	}}
}

// Run starts the program on one connection and blocks until EOF.
func (p *RefProgram) Run(in io.Reader, out io.Writer) error {
	p.client = sdk.New(in, out, sdk.Handlers{
		Hello:        p.onHello,
		Key:          p.onKey,
		Paste:        p.onPaste,
		Mouse:        p.onMouse,
		Wheel:        p.onWheel,
		Resize:       p.onResize,
		Sources:      p.onSources,
		Notice:       p.onNotice,
		Component:    p.onComponent,
		ViewRejected: p.onViewRejected,
		Response:     p.onResponse,
	})
	return p.client.Loop()
}

func (p *RefProgram) onHello(hello *pb.Hello) {
	p.pane = false
	p.lines = []string{fmt.Sprintf("hello view=%s epoch=%d cols=%d rows=%d schema=%d",
		hello.GetViewId(), hello.GetEpoch(), hello.GetCols(), hello.GetRows(), hello.GetSchema())}
	if len(hello.GetComponents()) > 0 {
		p.lines = append(p.lines, "components="+strings.Join(hello.GetComponents(), ","))
	}
	if len(hello.GetMethods()) > 0 {
		p.lines = append(p.lines, "methods="+strings.Join(hello.GetMethods(), ","))
	}
	p.emitRead()
	p.commit()
}

func (p *RefProgram) onKey(event *pb.KeyEvent) {
	if event.GetKey() == "ctrl-p" {
		p.pane = true
	}
	p.lines = append(p.lines, fmt.Sprintf("key key=%s char=%s pane=%d", event.GetKey(), event.GetChar(), boolInt(p.pane)))
	if event.GetKey() == "ctrl-r" {
		p.emitRead()
	}
	p.commit()
}

func (p *RefProgram) onPaste(event *pb.PasteEvent) {
	p.lines = append(p.lines, fmt.Sprintf("paste id=%s text=%s", event.GetId(), event.GetText()))
	p.commit()
}

func (p *RefProgram) onMouse(event *pb.MouseEvent) {
	p.lines = append(p.lines, fmt.Sprintf("mouse action=%s button=%s x=%d y=%d node=%s",
		event.GetAction(), event.GetButton(), event.GetX(), event.GetY(), event.GetNode()))
	p.commit()
}

func (p *RefProgram) onWheel(event *pb.WheelEvent) {
	p.lines = append(p.lines, fmt.Sprintf("wheel delta=%d x=%d y=%d node=%s",
		event.GetDelta(), event.GetX(), event.GetY(), event.GetNode()))
	p.commit()
}

func (p *RefProgram) onResize(cols, rows int) {
	p.lines = append(p.lines, fmt.Sprintf("resize cols=%d rows=%d", cols, rows))
	p.commit()
}

func (p *RefProgram) onSources(items []*pb.Source) {
	p.lines = append(p.lines, fmt.Sprintf("sources count=%d", len(items)))
	for _, item := range items {
		p.lines = append(p.lines, fmt.Sprintf("source id=%s kind=%s endpoint=%s terminal=%s exited=%d",
			item.GetId(), item.GetKind(), item.GetEndpoint(), item.GetTerminalId(), boolInt(item.GetExited())))
	}
	for _, item := range items {
		if item.GetKind() == "terminal" && item.GetTerminalId() != "" {
			id := item.GetTerminalId()
			p.emit("terminal.attach", &pb.MethodParams{
				Endpoint: item.GetEndpoint(),
				Id:       id,
				Fit:      protoBool(true),
			}, func(resp *pb.Response) { p.callbackLine(resp) })
			break
		}
	}
	p.commit()
}

func (p *RefProgram) onNotice(level, message string) {
	p.lines = append(p.lines, fmt.Sprintf("notice level=%s message=%s", level, message))
	p.commit()
}

func (p *RefProgram) onComponent(event *pb.ComponentEvent) {
	p.lines = append(p.lines, fmt.Sprintf("component source=%s name=%s value=%s",
		event.GetSource(), event.GetName(), event.GetValue()))
	p.commit()
}

func (p *RefProgram) onViewRejected(epoch, rev uint64, reason string) {
	p.lines = append(p.lines, fmt.Sprintf("view-rejected epoch=%d rev=%d reason=%s", epoch, rev, reason))
	p.commit()
}

func (p *RefProgram) onResponse(response *pb.Response) {
	p.lines = append(p.lines, fmt.Sprintf("resp id=%d epoch=%d ok=%d err=%s",
		response.GetRequestId(), response.GetEpoch(), boolInt(response.GetOk()), response.GetError()))
	p.commit()
}

func (p *RefProgram) emitRead() {
	p.emit("clipboard.read", nil, func(resp *pb.Response) { p.callbackLine(resp) })
}

func (p *RefProgram) emit(method string, params *pb.MethodParams, callback func(*pb.Response)) {
	id, err := p.client.Emit(method, params, callback)
	if err != nil {
		p.lines = append(p.lines, "emit-error "+err.Error())
		return
	}
	p.lines = append(p.lines, fmt.Sprintf("emit %s id=%d", method, id))
}

func (p *RefProgram) callbackLine(resp *pb.Response) {
	data := resp.GetData()
	p.lines = append(p.lines, fmt.Sprintf("cb id=%d ok=%d text=%s endpoint=%s data_id=%s rows=%d err=%s",
		resp.GetRequestId(), boolInt(resp.GetOk()), data.GetText(), data.GetEndpoint(),
		data.GetId(), len(data.GetRows()), resp.GetError()))
}

func (p *RefProgram) commit() {
	root := sdk.Text(strings.Join(p.lines, "\n")).Build()
	if err := p.client.Commit(root, sdk.Keys{Claim: []string{"ctrl-p", "?"}}); err != nil {
		p.lines = append(p.lines, "commit-error "+err.Error())
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func protoBool(value bool) *bool { return &value }
