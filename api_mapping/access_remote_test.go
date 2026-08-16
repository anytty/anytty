package apimapping

import (
	"testing"

	"github.com/anytty/anytty/proto/apipb"
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
