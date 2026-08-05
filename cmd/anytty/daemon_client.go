package main

import (
	"context"
	"fmt"
	"time"
)

func waitForSocket(path string, timeout time.Duration, try func() error) error {
	return waitForSocketContext(context.Background(), path, timeout, func(context.Context) error { return try() })
}

func waitForSocketContext(ctx context.Context, path string, timeout time.Duration, try func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		if err := waitCtx.Err(); err != nil {
			if parentErr := ctx.Err(); parentErr != nil {
				return parentErr
			}
			return fmt.Errorf("timed out waiting for daemon at %s", path)
		}
		if err := try(waitCtx); err == nil {
			return nil
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-waitCtx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
			continue
		}
	}
}
