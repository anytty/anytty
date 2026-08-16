package runtime

import (
	"context"

	"github.com/anytty/anytty/proto/apipb"
)

// EndpointApplication 是 CLI/App 使用的 endpoint 诊断 application interface。
// ProbeEndpoint 由共享 runtime 执行完整 identity/authorization/Hello，再立即释放 probe session lease。
type EndpointApplication interface {
	ProbeEndpoint(context.Context, *apipb.EndpointProbeRequest) (*apipb.EndpointProbeResult, error)
}

// AccessApplication 是 CLI/App 管理 daemon-local client access 的 application interface。
// 所有方法都必须通过已认证 endpoint session 到达 owning daemon，不能直接读取客户端 credential store。
type AccessApplication interface {
	AccessIdentity(context.Context, *apipb.ClientAccessIdentityCommand) (*apipb.ClientAccessIdentityResult, error)
	ListAccess(context.Context, *apipb.ClientAccessListCommand) (*apipb.ClientAccessListResult, error)
	RevokeAccess(context.Context, *apipb.ClientAccessRevokeCommand) (*apipb.ClientAccessRevokeResult, error)
}
