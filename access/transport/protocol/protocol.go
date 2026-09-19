// Package protocol defines the stable Cloud wire contract shared by clients,
// daemons, and the proprietary AnyTTY Cloud services.
package protocol

import "time"

const (
	AgentGatewayVersion  uint32 = 4
	ClientGatewayVersion uint32 = 3
	// RelayUsageReportInterval and RelayConcurrencyStaleAfter are one shared
	// liveness contract for Edge snapshots and Controller expiry.
	RelayUsageReportInterval   = time.Minute
	RelayConcurrencyStaleAfter = 3 * RelayUsageReportInterval

	DaemonBlockedCode = "DAEMON_BLOCKED"
	DaemonDeletedCode = "DAEMON_DELETED"
)
