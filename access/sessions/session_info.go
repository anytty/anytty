package sessions

import (
	"context"
	"time"
)

// RemoteSessionInfo 是 access owner 完成 remoteauth 后交给 transport server 的最小会话身份。
// 它只用于会话生命周期（撤销/过期踢线），不携带任何能力限制。
type RemoteSessionInfo struct {
	GrantID     string
	PrincipalID string
	ExpiresAt   time.Time
}

type remoteSessionContextKey struct{}

// WithRemoteSessionInfo 把已验证的会话身份附着到 transport server 的 context。
func WithRemoteSessionInfo(ctx context.Context, info RemoteSessionInfo) context.Context {
	return context.WithValue(ctx, remoteSessionContextKey{}, info)
}

// RemoteSessionInfoFromContext 读取已验证的会话身份。
func RemoteSessionInfoFromContext(ctx context.Context) (RemoteSessionInfo, bool) {
	info, ok := ctx.Value(remoteSessionContextKey{}).(RemoteSessionInfo)
	return info, ok
}
