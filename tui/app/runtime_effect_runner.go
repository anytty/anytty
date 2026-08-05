package app

import (
	"context"
	"sync"
)

// SyncEffectRunner 立即执行 effect，适合 deterministic harness。
type SyncEffectRunner struct {
	canceled map[CancelToken]struct{}
}

func NewSyncEffectRunner() *SyncEffectRunner {
	return &SyncEffectRunner{canceled: make(map[CancelToken]struct{})}
}

func (runner *SyncEffectRunner) Run(ctx context.Context, effect Effect, post func(Msg)) {
	switch effect := effect.(type) {
	case FuncEffect:
		if effect.Run == nil {
			return
		}
		if effect.Async && !effect.ForceSyncInTests {
			return
		}
		if _, canceled := runner.canceled[effect.Token]; canceled && effect.Token != "" {
			return
		}
		msg := effect.Run(ctx)
		if msg != nil {
			post(msg)
		}
	case StreamEffect:
		if effect.Run == nil {
			return
		}
		if _, canceled := runner.canceled[effect.Token]; canceled && effect.Token != "" {
			return
		}
		effect.Run(ctx, post)
	default:
		return
	}
}

func (runner *SyncEffectRunner) Cancel(token CancelToken) {
	if token == "" {
		return
	}
	runner.canceled[token] = struct{}{}
}

// AsyncEffectRunner 同步执行普通 effect，异步执行标记为 Async 的 FuncEffect 和 StreamEffect。
type AsyncEffectRunner struct {
	mu      sync.Mutex
	nextID  uint64
	cancels map[CancelToken]asyncEffectHandle
	serials map[string]*asyncSerialQueue
}

type asyncEffectHandle struct {
	ID     uint64
	Cancel context.CancelFunc
}

type asyncSerialQueue struct {
	items   []asyncSerialItem
	running bool
}

type asyncSerialItem struct {
	ctx    context.Context
	effect FuncEffect
	post   func(Msg)
	done   func()
}

func NewAsyncEffectRunner() *AsyncEffectRunner {
	return &AsyncEffectRunner{
		cancels: make(map[CancelToken]asyncEffectHandle),
		serials: make(map[string]*asyncSerialQueue),
	}
}

func (runner *AsyncEffectRunner) Run(ctx context.Context, effect Effect, post func(Msg)) {
	switch effect := effect.(type) {
	case FuncEffect:
		if effect.Run == nil {
			return
		}
		if effect.Async {
			runner.runAsyncFunc(ctx, effect, post)
			return
		}
		msg := effect.Run(ctx)
		if msg != nil {
			post(msg)
		}
	case StreamEffect:
		if effect.Run == nil {
			return
		}
		runner.runStream(ctx, effect, post)
	}
}

func (runner *AsyncEffectRunner) runAsyncFunc(ctx context.Context, effect FuncEffect, post func(Msg)) {
	effectCtx, done := runner.start(effect.Token, ctx)
	if effect.SerialKey != "" {
		runner.enqueueSerialFunc(effectCtx, effect, post, done)
		return
	}
	go func() {
		defer done()
		msg := effect.Run(effectCtx)
		if msg != nil {
			post(msg)
		}
	}()
}

func (runner *AsyncEffectRunner) enqueueSerialFunc(ctx context.Context, effect FuncEffect, post func(Msg), done func()) {
	key := effect.SerialKey
	item := asyncSerialItem{ctx: ctx, effect: effect, post: post, done: done}
	runner.mu.Lock()
	queue := runner.serials[key]
	if queue == nil {
		queue = &asyncSerialQueue{}
		runner.serials[key] = queue
	}
	queue.items = append(queue.items, item)
	shouldStart := !queue.running
	if shouldStart {
		queue.running = true
	}
	runner.mu.Unlock()
	if shouldStart {
		go runner.runSerialFuncQueue(key)
	}
}

func (runner *AsyncEffectRunner) runSerialFuncQueue(key string) {
	for {
		runner.mu.Lock()
		queue := runner.serials[key]
		if queue == nil || len(queue.items) == 0 {
			if queue != nil {
				queue.running = false
				delete(runner.serials, key)
			}
			runner.mu.Unlock()
			return
		}
		item := queue.items[0]
		copy(queue.items, queue.items[1:])
		last := len(queue.items) - 1
		queue.items[last] = asyncSerialItem{}
		queue.items = queue.items[:last]
		runner.mu.Unlock()

		func() {
			defer item.done()
			msg := item.effect.Run(item.ctx)
			if msg != nil {
				item.post(msg)
			}
		}()
	}
}

func (runner *AsyncEffectRunner) runStream(ctx context.Context, effect StreamEffect, post func(Msg)) {
	effectCtx, done := runner.start(effect.Token, ctx)
	go func() {
		defer done()
		effect.Run(effectCtx, post)
	}()
}

func (runner *AsyncEffectRunner) start(token CancelToken, parent context.Context) (context.Context, func()) {
	if token == "" {
		ctx, cancel := context.WithCancel(parent)
		return ctx, cancel
	}
	runner.Cancel(token)
	ctx, cancel := context.WithCancel(parent)
	runner.mu.Lock()
	runner.nextID++
	id := runner.nextID
	runner.cancels[token] = asyncEffectHandle{ID: id, Cancel: cancel}
	runner.mu.Unlock()
	return ctx, func() {
		runner.mu.Lock()
		if current := runner.cancels[token]; current.ID == id {
			delete(runner.cancels, token)
		}
		runner.mu.Unlock()
		cancel()
	}
}

func (runner *AsyncEffectRunner) Cancel(token CancelToken) {
	if token == "" {
		return
	}
	runner.mu.Lock()
	handle := runner.cancels[token]
	if handle.Cancel != nil {
		delete(runner.cancels, token)
	}
	runner.mu.Unlock()
	if handle.Cancel != nil {
		handle.Cancel()
	}
}
