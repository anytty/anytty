//go:build !unix

package app

import "log/slog"

func startRuntimeHeapSignalProfiler(_ *AppRuntime, _ *slog.Logger) func() {
	return nil
}
