// Package conformance defines the language-neutral SDK conformance suite:
// fixtures (JSON Lines) that drive any layout program through the TUI v2
// protocol and check the frames it writes back.
//
// The suite is the executable contract every SDK must pass (PROTOCOL §0-§6):
// a fixture is a scripted host session (HELLO/EVENT/RESPONSE in) with the
// expected program output (VIEW/RESULT) asserted semantically, so a
// hand-written codec in any language can self-certify with:
//
//	tui2-sdk-verify --cmd "python3 my_sdk.py"
//
// See README.zh-CN.md for the conformance program behaviour a candidate must
// implement and fixtures.jsonl for the cases.
package conformance

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Fixture is one conformance scenario: a named sequence of host sends and
// expected program output, ended by a process exit check.
type Fixture struct {
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Steps []Step `json:"steps"`
}

// Step is one fixture action. Exactly one field is set.
type Step struct {
	Send   *SendStep   `json:"send,omitempty"`
	Expect *ExpectStep `json:"expect,omitempty"`
	Exit   *int        `json:"exit,omitempty"`
}

// SendStep is one host -> program frame. Hello/Event/Response are symbolic and
// encoded by the runner; Raw is a hex-encoded complete frame (length prefix +
// type byte + payload) sent byte for byte.
type SendStep struct {
	Hello    *HelloSpec    `json:"hello,omitempty"`
	Event    *EventSpec    `json:"event,omitempty"`
	Response *ResponseSpec `json:"response,omitempty"`
	Raw      string        `json:"raw,omitempty"`
}

// HelloSpec is a symbolic HELLO.
type HelloSpec struct {
	Schema     uint32          `json:"schema"`
	ViewID     string          `json:"view_id"`
	Epoch      uint64          `json:"epoch"`
	Cols       uint32          `json:"cols"`
	Rows       uint32          `json:"rows"`
	Components []string        `json:"components,omitempty"`
	Events     []string        `json:"events,omitempty"`
	Methods    []string        `json:"methods,omitempty"`
	Features   map[string]bool `json:"features,omitempty"`
	Limits     *LimitsSpec     `json:"limits,omitempty"`
}

// LimitsSpec mirrors hello.limits.
type LimitsSpec struct {
	MaxNodes           uint32 `json:"max_nodes,omitempty"`
	MaxMessageBytes    uint32 `json:"max_message_bytes,omitempty"`
	MaxPasteBytes      uint32 `json:"max_paste_bytes,omitempty"`
	MaxInflight        uint32 `json:"max_inflight_requests,omitempty"`
	OwnerLeaseTTLmilli uint32 `json:"owner_lease_ttl_ms,omitempty"`
}

// EventSpec is a symbolic EVENT; Kind selects the oneof.
type EventSpec struct {
	Kind    string       `json:"kind"`
	ID      string       `json:"id,omitempty"`
	Key     string       `json:"key,omitempty"`
	Char    string       `json:"char,omitempty"`
	Text    string       `json:"text,omitempty"`
	Action  string       `json:"action,omitempty"`
	Button  string       `json:"button,omitempty"`
	X       int32        `json:"x,omitempty"`
	Y       int32        `json:"y,omitempty"`
	Node    string       `json:"node,omitempty"`
	Delta   int32        `json:"delta,omitempty"`
	Cols    uint32       `json:"cols,omitempty"`
	Rows    uint32       `json:"rows,omitempty"`
	Level   string       `json:"level,omitempty"`
	Message string       `json:"message,omitempty"`
	Source  string       `json:"source,omitempty"`
	Name    string       `json:"name,omitempty"`
	Value   string       `json:"value,omitempty"`
	Epoch   uint64       `json:"epoch,omitempty"`
	Rev     uint64       `json:"rev,omitempty"`
	Reason  string       `json:"reason,omitempty"`
	Items   []SourceSpec `json:"items,omitempty"`
}

// SourceSpec is one source item of a sources event.
type SourceSpec struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Endpoint    string `json:"endpoint"`
	TerminalID  string `json:"terminal_id"`
	Attached    bool   `json:"attached"`
	Exited      bool   `json:"exited"`
	ExitCode    int32  `json:"exit_code"`
	Health      string `json:"health"`
	ResizeOwner string `json:"resize_owner"`
	OwnerEpoch  uint64 `json:"owner_epoch"`
}

// ResponseSpec is a symbolic RESPONSE. RequestID is a JSON number or a
// "$rN" reference to the Nth RESULT the program emitted.
type ResponseSpec struct {
	RequestID any       `json:"request_id"`
	Epoch     uint64    `json:"epoch"`
	OK        bool      `json:"ok"`
	Error     string    `json:"error,omitempty"`
	Data      *DataSpec `json:"data,omitempty"`
}

// DataSpec mirrors MethodData.
type DataSpec struct {
	Rows     []string `json:"rows,omitempty"`
	Text     string   `json:"text,omitempty"`
	Endpoint string   `json:"endpoint,omitempty"`
	ID       string   `json:"id,omitempty"`
}

// ExpectStep lists program frames expected in order. An empty list means the
// step sends input and asserts nothing (the next expect still catches stray
// frames).
type ExpectStep struct {
	Frames []ExpectFrame `json:"frames"`
}

// ExpectFrame matches one VIEW or RESULT frame. Unset fields are not checked.
type ExpectFrame struct {
	Result *ExpectResult `json:"result,omitempty"`
	View   *ExpectView   `json:"view,omitempty"`
}

// ExpectResult matches a RESULT frame.
type ExpectResult struct {
	RequestID string         `json:"request_id,omitempty"`
	Epoch     *uint64        `json:"epoch,omitempty"`
	Method    string         `json:"method"`
	Params    map[string]any `json:"params,omitempty"`
}

// ExpectView matches a VIEW frame; Text is the flattened content lines.
type ExpectView struct {
	Epoch *uint64  `json:"epoch,omitempty"`
	Rev   *uint64  `json:"rev,omitempty"`
	Claim []string `json:"claim,omitempty"`
	All   *bool    `json:"all,omitempty"`
	Text  []string `json:"text,omitempty"`
}

// LoadFixtures reads a JSON Lines fixture file.
func LoadFixtures(path string) ([]Fixture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var out []Fixture
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "//") {
			continue
		}
		var fixture Fixture
		if err := json.Unmarshal([]byte(text), &fixture); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		if fixture.Name == "" {
			return nil, fmt.Errorf("%s:%d: fixture name is empty", path, line)
		}
		out = append(out, fixture)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Validate checks the fixture for structural mistakes so a broken fixture
// fails loudly instead of silently passing.
func (f Fixture) Validate() error {
	if len(f.Steps) == 0 {
		return fmt.Errorf("fixture %q has no steps", f.Name)
	}
	for i, step := range f.Steps {
		set := 0
		if step.Send != nil {
			set++
			if err := step.Send.validate(); err != nil {
				return fmt.Errorf("fixture %q step %d: %w", f.Name, i, err)
			}
		}
		if step.Expect != nil {
			set++
			for _, frame := range step.Expect.Frames {
				if (frame.Result == nil) == (frame.View == nil) {
					return fmt.Errorf("fixture %q step %d: expect frame must set exactly one of result/view", f.Name, i)
				}
			}
		}
		if step.Exit != nil {
			set++
		}
		if set != 1 {
			return fmt.Errorf("fixture %q step %d: exactly one of send/expect/exit must be set", f.Name, i)
		}
	}
	return nil
}

func (s SendStep) validate() error {
	set := 0
	if s.Hello != nil {
		set++
	}
	if s.Event != nil {
		set++
	}
	if s.Response != nil {
		set++
	}
	if s.Raw != "" {
		set++
		if _, err := hex.DecodeString(s.Raw); err != nil {
			return fmt.Errorf("raw frame is not hex: %w", err)
		}
	}
	if set != 1 {
		return fmt.Errorf("send must set exactly one of hello/event/response/raw")
	}
	return nil
}

// Factory builds the candidate endpoint under test.
type Factory interface {
	// Start launches one candidate session; it must be killed if the fixture
	// times out.
	Start() (Endpoint, error)
}

// Endpoint is one running candidate program.
type Endpoint interface {
	// Write sends bytes to the program stdin.
	Write([]byte) error
	// Read reads program stdout bytes.
	Read([]byte) (int, error)
	// CloseInput closes stdin (clean EOF).
	CloseInput() error
	// Wait waits for the process to exit and returns its exit code.
	Wait() (int, error)
	// Kill aborts the process; safe to call after Wait.
	Kill()
}

// InProcessFactory runs a Go reference program on io.Pipes.
type InProcessFactory struct {
	Run func(in io.Reader, out io.Writer) error
}

// Start implements Factory.
func (f InProcessFactory) Start() (Endpoint, error) {
	progIn, hostIn := io.Pipe()
	hostOut, progOut := io.Pipe()
	done := make(chan error, 1)
	go func() { done <- f.Run(progIn, progOut) }()
	return &pipeEndpoint{in: hostIn, out: hostOut, done: done}, nil
}

type pipeEndpoint struct {
	in   *io.PipeWriter
	out  *io.PipeReader
	done chan error
	once bool
}

func (p *pipeEndpoint) Write(data []byte) error { _, err := p.in.Write(data); return err }
func (p *pipeEndpoint) Read(data []byte) (int, error) {
	return p.out.Read(data)
}
func (p *pipeEndpoint) CloseInput() error {
	if p.once {
		return nil
	}
	p.once = true
	return p.in.Close()
}
func (p *pipeEndpoint) Wait() (int, error) {
	if err := p.CloseInput(); err != nil {
		return 0, err
	}
	err := <-p.done
	if err == nil || err == io.EOF {
		return 0, nil
	}
	return 1, err
}
func (p *pipeEndpoint) Kill() {
	_ = p.in.CloseWithError(io.ErrClosedPipe)
	_ = p.out.CloseWithError(io.ErrClosedPipe)
}

// CommandFactory launches an external candidate with exec.Command.
type CommandFactory struct {
	Argv []string
}

// Result is the outcome of one fixture run.
type Result struct {
	Name   string
	Passed bool
	Steps  int
	Err    error
}

// Run executes every fixture against a fresh endpoint and returns the results.
func Run(fixtures []Fixture, factory Factory, timeout time.Duration) []Result {
	results := make([]Result, 0, len(fixtures))
	for _, fixture := range fixtures {
		results = append(results, runOne(fixture, factory, timeout))
	}
	return results
}

func runOne(fixture Fixture, factory Factory, timeout time.Duration) Result {
	result := Result{Name: fixture.Name}
	if err := fixture.Validate(); err != nil {
		result.Err = err
		return result
	}
	endpoint, err := factory.Start()
	if err != nil {
		result.Err = fmt.Errorf("start candidate: %w", err)
		return result
	}
	var timedOut atomic.Bool
	watchdog := time.AfterFunc(timeout, func() {
		timedOut.Store(true)
		endpoint.Kill()
	})
	defer watchdog.Stop()
	defer endpoint.Kill()

	session := &session{endpoint: endpoint, dec: wire.NewDecoder(endpoint, wire.RoleHost, wire.DefaultMaxMessageBytes), binds: map[string]uint64{}}
	expectedExit := 0
	for i, step := range fixture.Steps {
		result.Steps = i + 1
		switch {
		case step.Send != nil:
			result.Err = session.send(step.Send)
		case step.Expect != nil:
			result.Err = session.expect(step.Expect)
		case step.Exit != nil:
			expectedExit = *step.Exit
			result.Err = session.finish(expectedExit)
		}
		if result.Err != nil {
			if timedOut.Load() {
				result.Err = fmt.Errorf("timeout after %s: %v", timeout, result.Err)
			}
			result.Err = fmt.Errorf("step %d: %w", i+1, result.Err)
			return result
		}
	}
	if result.Err == nil {
		if err := session.finish(expectedExit); err != nil {
			if timedOut.Load() {
				err = fmt.Errorf("timeout after %s: %v", timeout, err)
			}
			result.Err = err
			return result
		}
	}
	result.Passed = true
	return result
}

type session struct {
	endpoint Endpoint
	dec      *wire.Decoder
	binds    map[string]uint64
	results  int
	finished bool
}

func (s *session) send(step *SendStep) error {
	frame, err := buildFrame(step, s.binds)
	if err != nil {
		return err
	}
	if err := s.endpoint.Write(frame); err != nil {
		return fmt.Errorf("write host frame: %w", err)
	}
	return nil
}

func (s *session) expect(step *ExpectStep) error {
	for _, want := range step.Frames {
		actual, err := s.readFrame()
		if err != nil {
			return err
		}
		if err := matchFrame(want, actual, s.binds, &s.results); err != nil {
			return err
		}
	}
	return nil
}

func (s *session) readFrame() (*outFrame, error) {
	t, payload, err := s.dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("read program frame: %w", err)
	}
	switch t {
	case wire.TypeView:
		view, err := decodeView(payload)
		if err != nil {
			return nil, err
		}
		return &outFrame{view: view}, nil
	case wire.TypeResult:
		result, err := decodeResult(payload, s.results)
		if err != nil {
			return nil, err
		}
		return &outFrame{result: result}, nil
	default:
		return nil, fmt.Errorf("program sent illegal %s frame", t)
	}
}

func (s *session) finish(expectedExit int) error {
	if s.finished {
		return nil
	}
	s.finished = true
	if err := s.endpoint.CloseInput(); err != nil {
		return err
	}
	code, err := s.endpoint.Wait()
	if err != nil {
		return fmt.Errorf("candidate failed: %w", err)
	}
	if code != expectedExit {
		return fmt.Errorf("candidate exit code %d, want %d", code, expectedExit)
	}
	return nil
}

// outFrame is one decoded program frame.
type outFrame struct {
	view   *outView
	result *outResult
}

type outView struct {
	Epoch uint64
	Rev   uint64
	Claim []string
	All   bool
	Text  string
}

type outResult struct {
	RequestID uint64
	Epoch     uint64
	Method    string
	Params    map[string]any
}

func decodeView(payload []byte) (*outView, error) {
	message, err := wire.UnmarshalPayload(wire.TypeView, payload)
	if err != nil {
		return nil, err
	}
	view := message.(*pb.View)
	out := &outView{
		Epoch: view.GetEpoch(),
		Rev:   view.GetRev(),
		Claim: append([]string(nil), view.GetKeys().GetClaim()...),
		All:   view.GetKeys().GetAll(),
	}
	if root := view.GetRoot(); root != nil {
		var lines []string
		flattenBox(root, &lines)
		out.Text = strings.Join(lines, "\n")
	}
	return out, nil
}

func flattenBox(box *pb.Box, lines *[]string) {
	if content := box.GetContent(); content != nil {
		switch {
		case content.GetSelf() != "":
			*lines = append(*lines, "self:"+content.GetSelf())
		case len(content.GetLines()) > 0:
			*lines = append(*lines, content.GetLines()...)
		case content.GetText() != "":
			*lines = append(*lines, strings.Split(content.GetText(), "\n")...)
		}
	}
	for _, child := range box.GetChildren() {
		flattenBox(child, lines)
	}
}

func decodeResult(payload []byte, index int) (*outResult, error) {
	message, err := wire.UnmarshalPayload(wire.TypeResult, payload)
	if err != nil {
		return nil, err
	}
	result := message.(*pb.Result)
	return &outResult{
		RequestID: result.GetRequestId(),
		Epoch:     result.GetEpoch(),
		Method:    result.GetMethod(),
		Params:    paramsJSON(result.GetParams()),
	}, nil
}

func matchFrame(want ExpectFrame, got *outFrame, binds map[string]uint64, index *int) error {
	if want.Result != nil {
		if got.result == nil {
			return fmt.Errorf("want RESULT %s, got a VIEW frame", want.Result.Method)
		}
		return matchResult(want.Result, got.result, binds, index)
	}
	if got.view == nil {
		return fmt.Errorf("want VIEW, got a RESULT frame")
	}
	return matchView(want.View, got.view)
}

func matchResult(want *ExpectResult, got *outResult, binds map[string]uint64, index *int) error {
	if want.Method != got.Method {
		return fmt.Errorf("RESULT method %q, want %q", got.Method, want.Method)
	}
	if want.Epoch != nil && *want.Epoch != got.Epoch {
		return fmt.Errorf("RESULT epoch %d, want %d", got.Epoch, *want.Epoch)
	}
	if name := fmt.Sprintf("$r%d", *index); true {
		binds[name] = got.RequestID
	}
	*index++
	if want.RequestID != "" {
		bind, ok := binds[want.RequestID]
		if !ok {
			return fmt.Errorf("RESULT request_id reference %q is not bound", want.RequestID)
		}
		if bind != got.RequestID {
			return fmt.Errorf("RESULT request_id %d, want %d (%s)", got.RequestID, bind, want.RequestID)
		}
	}
	if want.Params != nil {
		if err := compareJSON("RESULT params", want.Params, got.Params); err != nil {
			return err
		}
	}
	return nil
}

func matchView(want *ExpectView, got *outView) error {
	if want.Epoch != nil && *want.Epoch != got.Epoch {
		return fmt.Errorf("VIEW epoch %d, want %d", got.Epoch, *want.Epoch)
	}
	if want.Rev != nil && *want.Rev != got.Rev {
		return fmt.Errorf("VIEW rev %d, want %d", got.Rev, *want.Rev)
	}
	if want.Claim != nil && !equalStrings(want.Claim, got.Claim) {
		return fmt.Errorf("VIEW claim %v, want %v", got.Claim, want.Claim)
	}
	if want.All != nil && *want.All != got.All {
		return fmt.Errorf("VIEW all=%v, want %v", got.All, *want.All)
	}
	if want.Text != nil {
		wantText := strings.Join(want.Text, "\n")
		if got.Text != wantText {
			return fmt.Errorf("VIEW text:\n  got  %q\n  want %q", got.Text, wantText)
		}
	}
	return nil
}

func compareJSON(label string, want, got any) error {
	wantJSON, err := json.Marshal(normalize(want))
	if err != nil {
		return err
	}
	gotJSON, err := json.Marshal(normalize(got))
	if err != nil {
		return err
	}
	if string(wantJSON) != string(gotJSON) {
		return fmt.Errorf("%s:\n  got  %s\n  want %s", label, gotJSON, wantJSON)
	}
	return nil
}

// normalize turns numbers decoded into any into a canonical shape so JSON
// round-trips compare stably.
func normalize(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = normalize(item)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = normalize(item)
		}
		return out
	case float64:
		if typed == float64(int64(typed)) {
			return int64(typed)
		}
		return typed
	default:
		return value
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// paramsJSON converts MethodParams into a canonical map holding only the
// fields the program actually set, so fixtures compare meaning, not bytes.
func paramsJSON(params *pb.MethodParams) map[string]any {
	if params == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	setString := func(key, value string) {
		if value != "" {
			out[key] = value
		}
	}
	setString("endpoint", params.GetEndpoint())
	setString("id", params.GetId())
	if params.Fit != nil {
		out["fit"] = params.GetFit()
	}
	if params.ExpectedOwnerEpoch != nil {
		out["expected_owner_epoch"] = params.GetExpectedOwnerEpoch()
	}
	if argv := params.GetArgv(); len(argv) > 0 {
		out["argv"] = argv
	}
	setString("cwd", params.GetCwd())
	if env := params.GetEnv(); len(env) > 0 {
		out["env"] = env
	}
	setString("title", params.GetTitle())
	if params.Ephemeral != nil {
		out["ephemeral"] = params.GetEphemeral()
	}
	if params.GetDelta() != 0 {
		out["delta"] = params.GetDelta()
	}
	if sel := params.GetSel(); sel != nil {
		out["sel"] = map[string]any{"mode": sel.GetMode(), "start": sel.GetStart(), "end": sel.GetEnd()}
	}
	if params.GetOffset() != 0 {
		out["offset"] = params.GetOffset()
	}
	if params.GetRows() != 0 {
		out["rows"] = params.GetRows()
	}
	setString("event_id", params.GetEventId())
	setString("source", params.GetSource())
	if params.CleanupOwned != nil {
		out["cleanup_owned"] = params.GetCleanupOwned()
	}
	setString("kind", params.GetKind())
	setString("socket", params.GetSocket())
	setString("connect_mode", params.GetConnectMode())
	setString("address", params.GetAddress())
	return out
}

// buildFrame encodes one symbolic host step into wire bytes.
func buildFrame(step *SendStep, binds map[string]uint64) ([]byte, error) {
	switch {
	case step.Hello != nil:
		return wire.Marshal(wire.TypeHello, helloMessage(step.Hello), 0)
	case step.Event != nil:
		event, err := eventMessage(step.Event)
		if err != nil {
			return nil, err
		}
		return wire.Marshal(wire.TypeEvent, event, 0)
	case step.Response != nil:
		response, err := responseMessage(step.Response, binds)
		if err != nil {
			return nil, err
		}
		return wire.Marshal(wire.TypeResponse, response, 0)
	default:
		return hex.DecodeString(step.Raw)
	}
}

func helloMessage(spec *HelloSpec) *pb.Hello {
	hello := &pb.Hello{
		Schema:     spec.Schema,
		ViewId:     spec.ViewID,
		Epoch:      spec.Epoch,
		Cols:       spec.Cols,
		Rows:       spec.Rows,
		Components: append([]string(nil), spec.Components...),
		Events:     append([]string(nil), spec.Events...),
		Methods:    append([]string(nil), spec.Methods...),
	}
	if len(spec.Features) > 0 {
		hello.Features = make(map[string]bool, len(spec.Features))
		for key, value := range spec.Features {
			hello.Features[key] = value
		}
	}
	if spec.Limits != nil {
		hello.Limits = &pb.Limits{
			MaxNodes:            spec.Limits.MaxNodes,
			MaxMessageBytes:     spec.Limits.MaxMessageBytes,
			MaxPasteBytes:       spec.Limits.MaxPasteBytes,
			MaxInflightRequests: spec.Limits.MaxInflight,
			OwnerLeaseTtlMs:     spec.Limits.OwnerLeaseTTLmilli,
		}
	}
	return hello
}

func eventMessage(spec *EventSpec) (*pb.Event, error) {
	switch spec.Kind {
	case "key":
		return &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: spec.ID, Key: spec.Key, Char: spec.Char}}}, nil
	case "paste":
		return &pb.Event{Event: &pb.Event_Paste{Paste: &pb.PasteEvent{Id: spec.ID, Text: spec.Text}}}, nil
	case "mouse":
		return &pb.Event{Event: &pb.Event_Mouse{Mouse: &pb.MouseEvent{Action: spec.Action, Button: spec.Button, X: spec.X, Y: spec.Y, Node: spec.Node}}}, nil
	case "wheel":
		return &pb.Event{Event: &pb.Event_Wheel{Wheel: &pb.WheelEvent{Delta: spec.Delta, X: spec.X, Y: spec.Y, Node: spec.Node}}}, nil
	case "resize":
		return &pb.Event{Event: &pb.Event_Resize{Resize: &pb.ResizeEvent{Cols: spec.Cols, Rows: spec.Rows}}}, nil
	case "sources":
		items := make([]*pb.Source, 0, len(spec.Items))
		for _, item := range spec.Items {
			items = append(items, &pb.Source{
				Id: item.ID, Kind: item.Kind, Title: item.Title,
				Endpoint: item.Endpoint, TerminalId: item.TerminalID,
				Attached: item.Attached, Exited: item.Exited, ExitCode: item.ExitCode,
				Health: item.Health, ResizeOwner: item.ResizeOwner, OwnerEpoch: item.OwnerEpoch,
			})
		}
		return &pb.Event{Event: &pb.Event_Sources{Sources: &pb.SourcesEvent{Items: items}}}, nil
	case "notice":
		return &pb.Event{Event: &pb.Event_Notice{Notice: &pb.NoticeEvent{Level: spec.Level, Message: spec.Message}}}, nil
	case "component":
		return &pb.Event{Event: &pb.Event_Component{Component: &pb.ComponentEvent{Source: spec.Source, Name: spec.Name, Value: spec.Value}}}, nil
	case "view_rejected":
		return &pb.Event{Event: &pb.Event_ViewRejected{ViewRejected: &pb.ViewRejectedEvent{Epoch: spec.Epoch, Rev: spec.Rev, Reason: spec.Reason}}}, nil
	default:
		return nil, fmt.Errorf("unknown event kind %q", spec.Kind)
	}
}

func responseMessage(spec *ResponseSpec, binds map[string]uint64) (*pb.Response, error) {
	response := &pb.Response{Epoch: spec.Epoch, Ok: spec.OK, Error: spec.Error}
	switch value := spec.RequestID.(type) {
	case nil:
		response.RequestId = 0
	case float64:
		response.RequestId = uint64(value)
	case string:
		id, ok := binds[value]
		if !ok {
			return nil, fmt.Errorf("response request_id reference %q is not bound", value)
		}
		response.RequestId = id
	default:
		return nil, fmt.Errorf("response request_id must be a number or $rN, got %T", spec.RequestID)
	}
	if spec.Data != nil {
		response.Data = &pb.MethodData{
			Rows: append([]string(nil), spec.Data.Rows...),
			Text: spec.Data.Text, Endpoint: spec.Data.Endpoint, Id: spec.Data.ID,
		}
	}
	return response, nil
}

// Summary renders the final report lines, one per fixture.
func Summary(results []Result) string {
	var builder strings.Builder
	passed := 0
	for _, result := range results {
		if result.Passed {
			passed++
			fmt.Fprintf(&builder, "PASS  %s\n", result.Name)
			continue
		}
		fmt.Fprintf(&builder, "FAIL  %s: %v\n", result.Name, result.Err)
	}
	fmt.Fprintf(&builder, "sdk-verify: %d/%d fixtures passed\n", passed, len(results))
	return builder.String()
}

// SortByName orders fixtures deterministically.
func SortByName(fixtures []Fixture) {
	sort.Slice(fixtures, func(i, j int) bool { return fixtures[i].Name < fixtures[j].Name })
}
