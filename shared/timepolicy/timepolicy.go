// Package timepolicy defines clock-tolerance policy shared by signed cross-device protocols.
package timepolicy

import "time"

// ClockSkewTolerance is the maximum accepted wall-clock difference between protocol peers.
const ClockSkewTolerance = 10 * time.Minute
