//go:build !unix

package main

import (
	"context"
	"log/slog"
)

func startDaemonHeapProfiler(_ context.Context, _ *slog.Logger) func(string) {
	return func(string) {}
}
