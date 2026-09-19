package server

import (
	"testing"

	"github.com/anytty/anytty/proto/access/apipb"
)

func TestFamilyOfCommandClassifiesRoutingTable(t *testing.T) {
	cases := []struct {
		name    string
		command *apipb.CommandEnvelope
		want    CommandFamily
	}{
		{"terminal create", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalCreate{}}, FamilyTerminal},
		{"terminal attach", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalAttach{}}, FamilyTerminal},
		{"terminal input", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalInput{}}, FamilyTerminal},
		{"release resource", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ReleaseResource{}}, FamilyTerminal},
		{"file list", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_FileList{}}, FamilyFile},
		{"file upload open", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_FileUploadOpen{}}, FamilyFile},
		{"browser proxy", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_BrowserProxyOpen{}}, FamilyProxy},
		{"storage put", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_StoragePut{}}, FamilyStorage},
		{"store list", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_StorageList{}}, FamilyStorage},
		{"client access list", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ClientAccessList{}}, FamilyAuth},
		{"cloud status", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_RemoteCloudStatus{}}, FamilyAuth},
		{"empty", &apipb.CommandEnvelope{}, FamilyUnknown},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := familyOfCommand(testCase.command); got != testCase.want {
				t.Fatalf("familyOfCommand = %s, want %s", got, testCase.want)
			}
		})
	}
}

func TestUnsupportedCommandProducesInvalidRequest(t *testing.T) {
	result := providerErrorResult(&apipb.CommandEnvelope{}, errUnsupportedCommand)
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST {
		t.Fatalf("unsupported command code = %s, want invalid_request", result.GetError().GetCode())
	}
}
