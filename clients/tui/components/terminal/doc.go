// Package terminal implements the builtin terminal component of the TUI v2
// host (PROTOCOL §5 "terminal"): it owns the visual chrome of one terminal
// source, a snapshot of its screen cells, scrollback window state and the
// scroll/scrollEnd/copy capabilities.
//
// The component never talks to a PTY. Its authoritative lifecycle state
// arrives through sources events (attached/exited) and it reaches history
// and clipboard through injected ports (ARCHITECTURE §2.6); the runtime is
// the only code that owns a real PTY. Render returns styled lines relative
// to the component origin, so the runtime decides placement and z-order.
package terminal
