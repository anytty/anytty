// Package runtime 是跨平台客户端连接运行时的领域 owner。
// route race、ReadyPeerSession、generation fence 和 command/event 行为按 CONN003
// 后续切片实现，不得从已删除的 TUI 或 CLI owner 复制旧逻辑。
package runtime
