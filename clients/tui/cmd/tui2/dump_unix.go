//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

// installGoroutineDump writes every goroutine stack to the host log on SIGUSR1.
//
// A frozen TUI (screen and input both dead) is almost always a lock or pipe
// deadlock. SIGQUIT would print the stacks to the terminal and kill the UI;
// SIGUSR1 keeps the process alive and puts the dump where the log already is,
// so the frozen session can still be inspected.
func installGoroutineDump(path string) {
	if path == "" {
		return
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	go func() {
		for range ch {
			file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
			if err != nil {
				continue
			}
			fmt.Fprintf(file, "\n=== tui2 goroutine dump %s ===\n", time.Now().Format(time.RFC3339Nano))
			_ = pprof.Lookup("goroutine").WriteTo(file, 1)
			_ = file.Close()
		}
	}()
}
