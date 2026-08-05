//go:build !windows

package terminalhost

import (
	"os"
	"os/signal"
	"syscall"
)

func defaultResizeSignalFactory(uintptr) (<-chan os.Signal, func()) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGWINCH)
	return signals, func() {
		signal.Stop(signals)
	}
}
