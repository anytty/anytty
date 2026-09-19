// Package binding 实现 JNI、WASM 及未来 Swift/Desktop wrapper 共用的稳定跨语言调用核心。
// 对外 payload 只允许 generated protobuf bytes 与 uint64 opaque handle；本包不暴露 Go pointer、channel、interface 或 core domain struct。
package binding
