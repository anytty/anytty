// Package runtime implements the TUI v2 host runtime core defined by
// tui2/docs/PROTOCOL.zh-CN.md: the frame session (epoch/view_id, VIEW and
// RESULT handling, backpressure), the §6.5 input routing table, the full
// sources snapshot and the §9.1 z-order composition.
//
// This package deliberately excludes PTY and terminal components: method
// execution goes through a Handler, and the default StubHandler fakes
// terminal.create ids, scroll/history rows and clipboard text. Everything
// that touches a real terminal lives outside this core.
//
// Layout is solved with tui2/kernel; the runtime only owns protocol state,
// epoch invalidation, routing and composition.
package runtime
