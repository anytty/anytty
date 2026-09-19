// Package endpoint connects the tui2 host to an existing anytty daemon and
// projects daemon terminals onto the same runtime.Terminal component pipeline
// as local PTYs (docs/ENDPOINTS.zh-CN.md).
//
// The package is host-side only: it speaks the repository wire protocol
// (proto/wire framing + proto/apipb application commands) over a transport
// connection and never imports the legacy tui/ packages. A layout program
// describes endpoints in its config and passes the connection parameters
// through MethodParams.kind/socket/connect_mode; the host registers them with
// a Manager and attaches terminals through RemotePTY, which satisfies
// tui2/pty.PTY so the runtime cannot tell a daemon terminal from a local one.
package endpoint
