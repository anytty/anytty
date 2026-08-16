//go:build unix

package app

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func startRuntimeHeapSignalProfiler(runtime *AppRuntime, logger *slog.Logger) func() {
	if runtime == nil || runtime.diagnostics == nil || runtime.diagnostics.heapProfileDir == "" && runtime.diagnostics.memstatsDir == "" {
		return nil
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1, syscall.SIGUSR2)
	done := make(chan struct{})
	go func() {
		defer signal.Stop(signals)
		for {
			select {
			case <-done:
				return
			case sig := <-signals:
				// 中文说明：真实 RSS harness 用 SIGUSR1 在 live/copy 关键点抓 TUI heap；
				// 只有显式设置 profile dir 时启用，不改变普通运行路径。
				switch sig {
				case syscall.SIGUSR1:
					runtime.RequestHeapProfile("usr1")
				case syscall.SIGUSR2:
					runtime.RequestMemstats("usr2")
				}
			}
		}
	}()
	return func() {
		close(done)
		if logger != nil {
			logger.Debug("tui-v3 heap signal profiler stopped")
		}
	}
}
