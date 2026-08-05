package webrtc

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLoggerFactoryRoutesPionErrorsToOwningLogger(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	pionLogger := NewLoggerFactory(logger).NewLogger("turnc")

	pionLogger.Info("connection info")
	pionLogger.Warn("connection warning")
	pionLogger.Errorf("Fail to refresh permissions: %s", "transaction failed")

	got := output.String()
	if strings.Contains(got, "connection info") || strings.Contains(got, "connection warning") {
		t.Fatalf("Pion bridge enabled logs below the production error boundary: %s", got)
	}
	for _, expected := range []string{
		`level=ERROR`,
		`msg="Fail to refresh permissions: transaction failed"`,
		`component=pion`,
		`scope=turnc`,
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("Pion error log missing %q: %s", expected, got)
		}
	}
}

func TestLoggerFactoryIsSilentWithoutOwningLogger(t *testing.T) {
	logger := NewLoggerFactory(nil).NewLogger("turnc")
	logger.Trace("trace")
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")
}
