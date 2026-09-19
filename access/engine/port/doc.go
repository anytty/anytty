// Package port 定义跨平台客户端运行时消费的宿主能力接口。
// 平台生命周期、credential custody、Cloud access 和 clock 实现只能经这些接口进入；
// 本包不拥有 route 或 session 状态。
package port
