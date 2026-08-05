// Package port 定义 TUI application-facing interface 与不可变 request/result/event DTO。
// 实现位于 adapter；所有结果必须通过 app message path 回投，不能直接修改 reducer-owned state。
package port
