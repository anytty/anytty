//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/x/term"
)

// watchResize follows the TTY size on Unix via SIGWINCH.
// The returned function stops the watcher.
func watchResize(host *Host, input *os.File) func() {
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-winch:
				if cols, rows, err := term.GetSize(input.Fd()); err == nil && cols > 0 && rows > 0 {
					host.Resize(cols, rows)
				}
			}
		}
	}()
	return func() {
		signal.Stop(winch)
		close(done)
	}
}
