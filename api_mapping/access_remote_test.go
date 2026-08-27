package apimapping

import (
	"testing"
	"time"

	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/remoteauthpb"
)

func TestValidateClientAccessIdentityRequiresFreshChallenge(t *testing.T) {
	command := &apipb.CommandEnvelope{
		Context: terminalRequestContext("identity-proof"),
		Command: &apipb.CommandEnvelope_ClientAccessIdentity{ClientAccessIdentity: &apipb.ClientAccessIdentityCommand{Challenge: make([]byte, deviceIdentityChallengeBytes)}},
	}
	if err := ValidateAccessRemoteCommand(command); err != nil {
		t.Fatal(err)
	}
	command.GetClientAccessIdentity().Challenge = nil
	if err := ValidateAccessRemoteCommand(command); err == nil {
		t.Fatal("identity command without a fresh challenge must fail")
	}
}

func TestClientAccessOwnerLabelRoundTripsThroughAPIMapping(t *testing.T) {
	request := ClientAccessTicketRequestFromProto(&apipb.ClientAccessTicketCreateCommand{Request: &remoteauthpb.ClientAccessTicketCreateRequest{
		Label: "Build host", AccessLabel: "App Store review phone", Scope: &remoteauthpb.ClientAccessScope{AllowDaemon: true}, TicketTtlSeconds: 60,
	}})
	if request.AccessLabel != "App Store review phone" {
		t.Fatalf("ticket access label = %q", request.AccessLabel)
	}
	record := ClientAccessRecordToProto(corev2.ClientAccessRecord{
		GrantID: "grant-1", AccessLabel: request.AccessLabel, ClientLabel: "anytty-ios", IssuedAt: time.Unix(1, 0).UTC(),
	})
	if record.GetAccessLabel() != "App Store review phone" || record.GetClientLabel() != "anytty-ios" {
		t.Fatalf("record labels = access %q client %q", record.GetAccessLabel(), record.GetClientLabel())
	}
}

func TestValidateClientAccessTicketAllowsDaemonDefaultLabel(t *testing.T) {
	command := &apipb.CommandEnvelope{
		Context: terminalRequestContext("default-label"),
		Command: &apipb.CommandEnvelope_ClientAccessTicketCreate{ClientAccessTicketCreate: &apipb.ClientAccessTicketCreateCommand{Request: &remoteauthpb.ClientAccessTicketCreateRequest{
			Scope: &remoteauthpb.ClientAccessScope{AllowDaemon: true}, TicketTtlSeconds: 60,
		}}},
	}
	if err := ValidateAccessRemoteCommand(command); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteLocalPasswordStateRoundTripsThroughAPIMapping(t *testing.T) {
	password := []byte("correct horse battery staple")
	request := RemoteLocalEnableRequestFromProto(&apipb.RemoteLocalEnableCommand{
		LocalWebAddress:  "127.0.0.1:0",
		LocalWebPassword: password,
	})
	if string(request.LocalWebPassword) != string(password) {
		t.Fatalf("local Web password = %q", request.LocalWebPassword)
	}
	request.LocalWebPassword[0] = 'X'
	if password[0] == 'X' {
		t.Fatal("API mapping retained the protobuf password buffer")
	}

	status := RemoteLocalStatusToProto(corev2.RemoteLocalStatus{Enabled: true, PasswordProtected: true})
	if !status.GetPasswordProtected() {
		t.Fatal("password-protected status was not projected")
	}
}
