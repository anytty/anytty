package remoteauth

import (
	"testing"
	"time"
)

func TestClockSkewToleranceIsTenMinutes(t *testing.T) {
	if ClockSkewTolerance != 10*time.Minute {
		t.Fatalf("ClockSkewTolerance = %s, want 10m", ClockSkewTolerance)
	}
}
