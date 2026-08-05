//go:build unix

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"
)

func startDaemonHeapProfiler(ctx context.Context, logger *slog.Logger) func(string) {
	dir := strings.TrimSpace(os.Getenv(daemonHeapProfileDirEnv))
	memstatsDir := strings.TrimSpace(os.Getenv(daemonMemstatsDirEnv))
	if dir == "" {
		if memstatsDir == "" {
			return func(string) {}
		}
		return startDaemonMemstatsSampler(ctx, logger, memstatsDir)
	}
	writeMemstats := startDaemonMemstatsSampler(ctx, logger, memstatsDir)
	var mu sync.Mutex
	write := func(reason string) {
		mu.Lock()
		defer mu.Unlock()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			logger.Warn("core-v2 daemon heap profile directory unavailable", "dir", dir, "error", err)
			return
		}
		runtime.GC()
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		path := filepath.Join(dir, fmt.Sprintf("core-%s-heap%d-%d.pprof", daemonHeapProfileReason(reason), mem.HeapAlloc, time.Now().UnixNano()))
		file, err := os.Create(path)
		if err != nil {
			logger.Warn("core-v2 daemon heap profile create failed", "path", path, "error", err)
			return
		}
		if err := pprof.WriteHeapProfile(file); err != nil {
			_ = file.Close()
			logger.Warn("core-v2 daemon heap profile write failed", "path", path, "error", err)
			return
		}
		if err := file.Close(); err != nil {
			logger.Warn("core-v2 daemon heap profile close failed", "path", path, "error", err)
			return
		}
		logger.Info("core-v2 daemon heap profile written", "path", path, "heap_alloc", mem.HeapAlloc, "reason", reason)
		writeMemstats(reason)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)
	go func() {
		defer signal.Stop(signals)
		for {
			select {
			case <-ctx.Done():
				return
			case <-signals:
				// 中文说明：RSS smoke 用 SIGUSR1 在关键采样点抓 daemon heap，
				// 默认未设置 profile dir 时完全不启用这条诊断链路。
				write("usr1")
			}
		}
	}()
	return write
}

func startDaemonMemstatsSampler(ctx context.Context, logger *slog.Logger, dir string) func(string) {
	if dir == "" {
		return func(string) {}
	}
	var mu sync.Mutex
	write := func(reason string) {
		mu.Lock()
		defer mu.Unlock()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			logger.Warn("core-v2 daemon memstats directory unavailable", "dir", dir, "error", err)
			return
		}
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		path := filepath.Join(dir, "memstats.tsv")
		newFile := false
		if _, err := os.Stat(path); os.IsNotExist(err) {
			newFile = true
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			logger.Warn("core-v2 daemon memstats open failed", "path", path, "error", err)
			return
		}
		defer file.Close()
		if newFile {
			fmt.Fprintln(file, "unix_ns\tprocess\tstage\treason\theap_alloc\theap_sys\theap_idle\theap_inuse\theap_released\theap_objects\tstack_sys\tmspan_sys\tmcache_sys\tbuck_hash_sys\tgc_sys\tother_sys\tsys\tnext_gc\tnum_gc")
		}
		stage := daemonHeapProfileReason(os.Getenv(daemonMemstatsStageEnv))
		if stage == "sample" {
			stage = daemonHeapProfileReason(readDaemonMemstatsStageFile())
		}
		if stage == "sample" {
			stage = daemonHeapProfileReason(reason)
		}
		fmt.Fprintf(file, "%d\tdaemon\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
			time.Now().UnixNano(),
			stage,
			daemonHeapProfileReason(reason),
			mem.HeapAlloc,
			mem.HeapSys,
			mem.HeapIdle,
			mem.HeapInuse,
			mem.HeapReleased,
			mem.HeapObjects,
			mem.StackSys,
			mem.MSpanSys,
			mem.MCacheSys,
			mem.BuckHashSys,
			mem.GCSys,
			mem.OtherSys,
			mem.Sys,
			mem.NextGC,
			mem.NumGC,
		)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR2)
	go func() {
		defer signal.Stop(signals)
		for {
			select {
			case <-ctx.Done():
				return
			case <-signals:
				// 中文说明：SIGUSR2 只写 runtime memstats，用于解释 RSS 与 Go heap/released 的差距。
				write("usr2")
			}
		}
	}()
	return write
}
