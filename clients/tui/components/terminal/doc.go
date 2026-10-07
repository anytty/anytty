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
//
// Content framing (PROTOCOL §5): a terminal source has ONE authoritative extent
// (its PTY cols/rows); a program may position that extent inside the component's
// content area without resizing the PTY by declaring content.offset ("x,y", the
// extent origin in cell units; x/y may be negative) and content.size ("cols,rows",
// the extent footprint). Render draws the screen 1:1 at the offset and fills every
// content cell outside the footprint with the chrome.placeholder glyph (default
// muted "·"). Absent props are inert: offset (0,0) and an implicit footprint
// covering the content area, so output stays byte-identical to the legacy render.
package terminal
