// Package keys is the host-side input encoder (PROTOCOL §6.5, §6.6, §6.8).
//
// It is the only place that turns normalized input events into PTY bytes:
// layout programs describe what the user did (key name, modifiers, paste
// text) and the host encodes it exactly once, on the way to the PTY. Events
// never carry bytes in either direction.
//
// The package is deliberately free of protocol and process state: callers
// (tui2/runtime) decide routing and destination, this package only encodes,
// chunks paste text and retains the last key/paste events for
// input.forward.
package keys
