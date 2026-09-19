// Package provider defines the session-provider seam of the access product
// line. A provider turns an endpoint identity into one live byte stream; the
// pool provider (access/provider/pool) dials the local pool socket, and
// future providers (for example tmux) implement the same Dial contract.
//
// The contract deliberately stops at net.Conn: the gateway is byte-transparent
// to the access wire, so providers must not parse frames, negotiate versions
// or carry protocol state. Callers own framing and connection lifecycle of the
// returned stream. List/Close are intentionally absent from this seam because
// listener lifecycle belongs to the gateway and session truth stays with the
// endpoint owner; a tmux provider can grow its own discovery surface later
// without widening SessionProvider.
package provider
