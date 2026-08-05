package main

import (
	"context"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
)

// createCLIProtoTerminal 让 CLI 集成测试通过正式 Proto application API 建立测试终端。
// 测试不再依赖已删除的 protocol terminal DTO 或旧 method codec。
func createCLIProtoTerminal(ctx context.Context, client *protocol.Client, spec *apipb.TerminalCreateSpec) (*apipb.TerminalCreateResult, error) {
	owned, err := wrapCLIProtocolClientForTestContext(ctx, client)
	if err != nil {
		return nil, err
	}
	application := owned.ApplicationSession
	return application.TerminalCreate(ctx, &apipb.TerminalCreateCommand{Terminal: spec})
}
