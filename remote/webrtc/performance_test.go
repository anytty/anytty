package webrtc

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	pion "github.com/pion/webrtc/v4"
)

func TestPeerPerformanceRecordsWindowAndRTTUnits(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	logPeerPerformance(logger, "test-session", pion.StatsReport{
		"sctpTransport": pion.SCTPTransportStats{SmoothedRoundTripTime: 0.125, CongestionWindow: 65536, ReceiverWindow: 1048576, MTU: 1228, BytesSent: 123, BytesReceived: 456},
	})
	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]float64{"sctp_rtt_ms": 125, "sctp_cwnd_bytes": 65536, "sctp_rwnd_bytes": 1048576, "sctp_mtu_bytes": 1228, "sctp_sent_bytes": 123, "sctp_received_bytes": 456} {
		if record[key] != want {
			t.Fatalf("%s = %v, want %v", key, record[key], want)
		}
	}
}

func TestPeerPerformanceStopsOnSessionCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		tracePeerPerformance(ctx, nil, nil, "test-session")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("performance sampler survived session cancellation")
	}
}
