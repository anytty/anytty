//go:build unix && anytty_memory_bench

package main

import (
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

// The memory benchmark copies this file into the baseline source tree so both
// binaries expose the same explicit post-setup GC fence. Production builds do
// not include this build-tagged hook.
func init() {
	gcSignals := make(chan os.Signal, 1)
	signal.Notify(gcSignals, syscall.SIGUSR1)
	go func() {
		for range gcSignals {
			debug.FreeOSMemory()
		}
	}()
}
