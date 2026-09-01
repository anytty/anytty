// Package protocol defines the stable Cloud wire contract shared by clients,
// daemons, and the proprietary AnyTTY Cloud services.
package protocol

const (
	AgentGatewayVersion  uint32 = 4
	ClientGatewayVersion uint32 = 3

	DaemonBlockedCode = "DAEMON_BLOCKED"
	DaemonDeletedCode = "DAEMON_DELETED"
)
