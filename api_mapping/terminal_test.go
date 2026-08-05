package apimapping

import (
	"math"
	"reflect"
	"testing"
	"time"

	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

func TestTerminalRecordFromProtoCopiesCreateSpecIntoCoreDomain(t *testing.T) {
	spec := &apipb.TerminalCreateSpec{TerminalId: "term-1", Name: "demo", Command: []string{"sh", "-l"}, Tags: map[string]string{"role": "dev"}, Size: &apipb.TerminalSize{Cols: 100, Rows: 30}, Cwd: "/srv/app", Env: []string{"A=1"}, ScrollbackRows: 2000, ScrollbackMaxBytes: 4096, ScrollbackMaxAgeSeconds: 120}
	record, err := TerminalRecordFromProto(spec)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != "term-1" || record.Size != (corev2.Size{Cols: 100, Rows: 30}) || record.Options.Dir != "/srv/app" || record.Options.ScrollbackMaxAge != 2*time.Minute {
		t.Fatalf("record=%#v", record)
	}
	spec.Command[0] = "mutated"
	spec.Tags["role"] = "mutated"
	if record.Command[0] != "sh" || record.Tags["role"] != "dev" {
		t.Fatalf("record aliases proto input: %#v", record)
	}
}

func TestTerminalInfoToProtoAddsEndpointWithoutMutatingCore(t *testing.T) {
	exitCode := 23
	now := time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC)
	info := corev2.TerminalInfo{ID: "term-1", Name: "demo", Command: []string{"sh"}, Tags: map[string]string{"role": "dev"}, Size: corev2.Size{Cols: 80, Rows: 24}, State: corev2.TerminalStateExited, CreatedAt: now.Add(-time.Minute), ExitedAt: now, ExitCode: &exitCode}
	projection, err := TerminalInfoToProto("studio", info, 2)
	if err != nil {
		t.Fatal(err)
	}
	if projection.GetRef().GetEndpointId() != "studio" || projection.GetState() != apipb.TerminalState_TERMINAL_STATE_EXITED || projection.GetExitCode() != 23 || projection.GetAttachmentCount() != 2 {
		t.Fatalf("projection=%#v", projection)
	}
	projection.Command[0] = "mutated"
	projection.Tags["role"] = "mutated"
	if !reflect.DeepEqual(info.Command, []string{"sh"}) || info.Tags["role"] != "dev" {
		t.Fatalf("projection aliases core domain: %#v", info)
	}
}

func TestTerminalInfoToProtoRejectsUnknownCoreState(t *testing.T) {
	_, err := TerminalInfoToProto("studio", corev2.TerminalInfo{ID: "term-1", State: corev2.TerminalState("future")}, 0)
	if err == nil {
		t.Fatal("unknown core terminal state must fail")
	}
}

func TestValidateTerminalInputRejectsMismatchedOperationSession(t *testing.T) {
	contextMessage := terminalRequestContext("request-1")
	command := &apipb.CommandEnvelope{Context: contextMessage, Command: &apipb.CommandEnvelope_TerminalInput{TerminalInput: &apipb.TerminalInputCommand{
		Attachment: &apipb.ResourceHandle{OpaqueToken: []byte("attachment-1"), Kind: apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT, Session: contextMessage.GetSession(), Generation: 1},
		Operation:  &apipb.OperationStamp{Session: &apipb.EndpointSessionStamp{EndpointId: "studio", RouteId: "ssh", Generation: 6}, OperationId: "input-1"},
		Data:       []byte("x"),
	}}}
	if err := ValidateTerminalCommand(command); err == nil {
		t.Fatal("mismatched operation session must fail")
	}
}

func TestValidateTerminalCreateRejectsNegativeAndOverflowingLimits(t *testing.T) {
	base := &apipb.TerminalCreateSpec{TerminalId: "term-1", Command: []string{"sh"}, Size: &apipb.TerminalSize{Cols: 80, Rows: 24}}
	negative := proto.Clone(base).(*apipb.TerminalCreateSpec)
	negative.ScrollbackRows = -1
	if err := ValidateTerminalCreateSpec(negative); err == nil {
		t.Fatal("negative scrollback rows must fail")
	}
	overflow := proto.Clone(base).(*apipb.TerminalCreateSpec)
	overflow.ScrollbackMaxAgeSeconds = math.MaxInt64/int64(time.Second) + 1
	if err := ValidateTerminalCreateSpec(overflow); err == nil {
		t.Fatal("overflowing scrollback duration must fail")
	}
}

func TestValidateTerminalAttachRejectsUnknownEnums(t *testing.T) {
	contextMessage := terminalRequestContext("request-enum")
	command := &apipb.CommandEnvelope{Context: contextMessage, Command: &apipb.CommandEnvelope_TerminalAttach{TerminalAttach: &apipb.TerminalAttachCommand{
		Terminal: &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"}, Mode: apipb.AttachmentMode(99),
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER, SurfaceId: "surface", ViewId: "view",
		Operation: &apipb.OperationStamp{Session: contextMessage.GetSession(), OperationId: "attach-1"},
	}}}
	if err := ValidateTerminalCommand(command); err == nil {
		t.Fatal("unknown attachment mode must fail")
	}
}

func TestValidateTerminalCommandRejectsNilTypedPayload(t *testing.T) {
	command := &apipb.CommandEnvelope{
		Context: terminalRequestContext("request-nil"),
		Command: &apipb.CommandEnvelope_TerminalDefaults{},
	}
	if err := ValidateTerminalCommand(command); err == nil {
		t.Fatal("nil typed payload must fail before API Layer cloning")
	}
}

func TestValidateTerminalCommandRejectsOversizedInputAndPathLimit(t *testing.T) {
	contextMessage := terminalRequestContext("request-limits")
	input := &apipb.CommandEnvelope{Context: contextMessage, Command: &apipb.CommandEnvelope_TerminalInput{TerminalInput: &apipb.TerminalInputCommand{
		Attachment: &apipb.ResourceHandle{OpaqueToken: []byte("attachment"), Kind: apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT, Session: contextMessage.GetSession(), Generation: 1},
		Operation:  &apipb.OperationStamp{Session: contextMessage.GetSession(), OperationId: "input"},
		Data:       make([]byte, maxTerminalInputBytes+1),
	}}}
	if err := ValidateTerminalCommand(input); err == nil {
		t.Fatal("oversized input must fail")
	}
	path := &apipb.CommandEnvelope{Context: contextMessage, Command: &apipb.CommandEnvelope_PathListDirectories{PathListDirectories: &apipb.PathListDirectoriesCommand{Limit: maxPathEntries + 1}}}
	if err := ValidateTerminalCommand(path); err == nil {
		t.Fatal("oversized path limit must fail")
	}
}

func TestValidateTerminalAttachResultRejectsCrossSessionResource(t *testing.T) {
	session := terminalRequestContext("request-result").GetSession()
	other := &apipb.EndpointSessionStamp{EndpointId: "studio", RouteId: "ssh", Generation: 6}
	command := &apipb.TerminalAttachCommand{
		Terminal: &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"},
		Mode:     apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId: "surface", ViewId: "view", Operation: &apipb.OperationStamp{Session: session, OperationId: "attach"},
	}
	result := &apipb.TerminalAttachResult{
		Attachment: &apipb.AttachmentHandle{
			Resource:  &apipb.ResourceHandle{OpaqueToken: []byte("attachment"), Kind: apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT, Session: other, Generation: 1},
			Terminal:  &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"},
			Operation: &apipb.OperationStamp{Session: session, OperationId: "attach"},
		},
		Mode: apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		Size: &apipb.TerminalSize{Cols: 80, Rows: 24},
	}
	if err := ValidateTerminalAttachResult(command, result, session); err == nil {
		t.Fatal("cross-session attachment result must fail")
	}
}

func TestHistorySearchAdmissionCarriesTerminalScope(t *testing.T) {
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_HistorySearch{HistorySearch: &apipb.HistorySearchCommand{
		Terminal: &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"},
	}}}
	admission := ApplicationAdmissionFromCommand(command, apipb.ApiCapability_API_CAPABILITY_HISTORY)
	if admission.TerminalID != "term-1" || admission.Capability != corev2.ApplicationCapabilityHistory {
		t.Fatalf("history search admission = %#v", admission)
	}
}

func terminalRequestContext(requestID string) *apipb.RequestContext {
	return &apipb.RequestContext{
		RequestId: requestID, ApiVersion: &apipb.ApiVersion{Major: 1},
		Session: &apipb.EndpointSessionStamp{EndpointId: "studio", RouteId: "ssh", Generation: 7},
	}
}
