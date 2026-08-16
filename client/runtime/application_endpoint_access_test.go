package runtime

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/remoteauthpb"
)

func TestEndpointAndAccessContractsDoNotExposeConnectionOrCredentialOwners(t *testing.T) {
	for _, value := range []any{&apipb.EndpointProbeResult{}, &apipb.ClientAccessIdentityResult{}, &remoteauthpb.ClientAccessRecord{}, &apipb.ClientAccessListResult{}} {
		typeOf := reflect.TypeOf(value).Elem()
		for index := 0; index < typeOf.NumField(); index++ {
			name := strings.ToLower(typeOf.Field(index).Name)
			for _, forbidden := range []string{"transport", "protocol", "capabilitygrant", "privatekey", "credential"} {
				if strings.Contains(name, forbidden) {
					t.Fatalf("%s exposes forbidden field %s", typeOf.Name(), typeOf.Field(index).Name)
				}
			}
		}
	}
}

type contractEndpointApplication struct{}

func (contractEndpointApplication) ProbeEndpoint(context.Context, *apipb.EndpointProbeRequest) (*apipb.EndpointProbeResult, error) {
	return &apipb.EndpointProbeResult{}, nil
}

type contractAccessApplication struct{}

func (contractAccessApplication) AccessIdentity(context.Context, *apipb.ClientAccessIdentityCommand) (*apipb.ClientAccessIdentityResult, error) {
	return &apipb.ClientAccessIdentityResult{}, nil
}

func (contractAccessApplication) ListAccess(context.Context, *apipb.ClientAccessListCommand) (*apipb.ClientAccessListResult, error) {
	return &apipb.ClientAccessListResult{}, nil
}

func (contractAccessApplication) RevokeAccess(context.Context, *apipb.ClientAccessRevokeCommand) (*apipb.ClientAccessRevokeResult, error) {
	return &apipb.ClientAccessRevokeResult{}, nil
}

var _ EndpointApplication = contractEndpointApplication{}
var _ AccessApplication = contractAccessApplication{}
