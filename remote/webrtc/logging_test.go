package webrtc

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	pion "github.com/pion/webrtc/v4"
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

func TestEnsureLoggerFactoryDefaultsSilentAndPreservesCustom(t *testing.T) {
	var settings pion.SettingEngine
	EnsureLoggerFactory(&settings, nil)
	if settings.LoggerFactory == nil {
		t.Fatal("zero SettingEngine kept Pion's default stderr logger")
	}

	var output bytes.Buffer
	custom := NewLoggerFactory(slog.New(slog.NewTextHandler(&output, nil)))
	settings = pion.SettingEngine{LoggerFactory: custom}
	EnsureLoggerFactory(&settings, nil)
	settings.LoggerFactory.NewLogger("turnc").Error("custom logger")
	if !strings.Contains(output.String(), "custom logger") {
		t.Fatalf("custom logger factory was not preserved: %s", output.String())
	}
}
