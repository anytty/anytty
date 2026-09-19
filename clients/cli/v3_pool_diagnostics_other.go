//go:build !unix

package cli

import (
	"context"
	"log/slog"
)

func startPoolHeapProfiler(_ context.Context, _ *slog.Logger) func(string) {
	return func(string) {}
}
