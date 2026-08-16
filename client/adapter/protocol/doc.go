// Package protocol 把 ready transport 映射到公开 anytty protocol client 边界。
// 本包为单次 attempt 执行 proof、authorization 与 Hello，并且只返回 ready result；
// 它不选择 route，也不发布 UI 状态。
package protocol
