//go:build windows

package main

import (
	"os"
	"time"

	"github.com/charmbracelet/x/term"
)

// watchResize follows the TTY size on Windows by polling, because Windows has
// no SIGWINCH. The returned function stops the watcher.
func watchResize(host *Host, input *os.File) func() {
	done := make(chan struct{})
	go func() {
		lastCols, lastRows, _ := term.GetSize(input.Fd())
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				cols, rows, err := term.GetSize(input.Fd())
				if err != nil || cols <= 0 || rows <= 0 {
					continue
				}
				if cols != lastCols || rows != lastRows {
					lastCols, lastRows = cols, rows
					host.Resize(cols, rows)
				}
			}
		}
	}()
	return func() { close(done) }
}
