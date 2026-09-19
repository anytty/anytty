// Package local 实现 local Unix route attempt adapter。
// adapter 只能为已规划 attempt 创建 transport；route selection、session cache
// 和 winner 状态始终归 client/runtime。
package local
