// Package ssh 实现 Go SSH direct-tcpip + WebRTC ICE-TCP route connector。
// 本包只拥有单次 attempt 的 SSH handshake、loopback forward 和 cleanup，不拥有 Endpoint selection、session generation 或 Proto API。
package ssh
