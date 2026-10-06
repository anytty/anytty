package endpoint

import (
	"context"
	"testing"

	"github.com/anytty/anytty/proto/access/apipb"
)

// TestManagerExecuteForwardsOpaqueCommand 固定 access.call 的连接层语义：
// 命令原样走到 endpoint 的 ready 连接并返回原始 ResultEnvelope；家族策略与
// 破坏性确认由 host gate 负责，manager 只做路由与错误可读化。
func TestManagerExecuteForwardsOpaqueCommand(t *testing.T) {
	d := newFakeDaemon(t)
	m := testManager(t)
	registerDaemon(t, m, d, "execdev")

	result, err := m.Execute(context.Background(), "execdev", &apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}},
	})
	if err != nil {
		t.Fatalf("execute forwarded command: %v", err)
	}
	if result.GetTerminalList() == nil {
		t.Fatalf("forwarded result = %#v", result)
	}
}

func TestManagerExecuteRejectsUnknownEndpoint(t *testing.T) {
	m := testManager(t)
	if _, err := m.Execute(context.Background(), "missing", &apipb.CommandEnvelope{}); err == nil {
		t.Fatal("unknown endpoint must fail")
	}
}
