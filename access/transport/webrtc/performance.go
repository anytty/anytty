package webrtc

import (
	"context"
	"log/slog"
	"time"

	pion "github.com/pion/webrtc/v4"
)

// Enabled only with the process performance recorder. Log counters/windows, not
// SDP, credentials or application payloads. Session cancellation joins lifetime.
func tracePeerPerformance(ctx context.Context, peer *pion.PeerConnection, logger *slog.Logger, sessionID string) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if peer.ConnectionState() == pion.PeerConnectionStateClosed {
				return
			}
			logPeerPerformance(logger, sessionID, peer.GetStats())
		}
	}
}

func logPeerPerformance(logger *slog.Logger, sessionID string, report pion.StatsReport) {
	for _, raw := range report {
		if stats, ok := raw.(pion.SCTPTransportStats); ok {
			logger.Info("AnyTTY WebRTC performance", "session_id", sessionID,
				"sctp_sent_bytes", stats.BytesSent, "sctp_received_bytes", stats.BytesReceived,
				"sctp_rtt_ms", stats.SmoothedRoundTripTime*1000,
				"sctp_cwnd_bytes", stats.CongestionWindow, "sctp_rwnd_bytes", stats.ReceiverWindow,
				"sctp_mtu_bytes", stats.MTU)
		}
	}
}
