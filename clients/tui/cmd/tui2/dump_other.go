//go:build windows

package main

// installGoroutineDump is a no-op where SIGUSR1 does not exist.
func installGoroutineDump(string) {}
