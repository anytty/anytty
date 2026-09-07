// Package connecttrace records connection setup stages without recording secrets.
package connecttrace

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

const header = "x-anytty-connect-trace"

type key struct{}

// Attach carries only a validated diagnostic ID, without changing cancellation.
func Attach(ctx context.Context, id string) context.Context {
	if ctx == nil {
		return nil
	}
	parsed, err := uuid.Parse(id)
	if err != nil || parsed == uuid.Nil {
		return ctx
	}
	return context.WithValue(ctx, key{}, parsed.String())
}

type Trace struct {
	mu            sync.Mutex
	id, component string
	started, last time.Time
}

// Start propagates a diagnostic-only ID. It is never authorization evidence.
func Start(ctx context.Context, component string) (context.Context, *Trace) {
	if ctx == nil {
		return nil, nil
	}
	id, _ := ctx.Value(key{}).(string)
	if id == "" {
		md, _ := metadata.FromIncomingContext(ctx)
		if values := md.Get(header); len(values) == 1 {
			if parsed, err := uuid.Parse(values[0]); err == nil && parsed != uuid.Nil {
				id = parsed.String()
			}
		}
	}
	if id == "" {
		id = uuid.NewString()
	}
	ctx = context.WithValue(ctx, key{}, id)
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Set(header, id)
	ctx = metadata.NewOutgoingContext(ctx, md)
	now := time.Now()
	trace := &Trace{id: id, component: component, started: now, last: now}
	trace.Mark("start")
	return ctx, trace
}

func ID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(key{}).(string)
	return id
}

// Mark reports elapsed time since the previous stage, plus total setup time.
func (trace *Trace) Mark(stage string) {
	if trace == nil {
		return
	}
	trace.mu.Lock()
	defer trace.mu.Unlock()
	now := time.Now()
	log.Printf("anytty connect trace_id=%s component=%s stage=%s stage_ms=%d total_ms=%d", trace.id, trace.component, stage, now.Sub(trace.last).Milliseconds(), now.Sub(trace.started).Milliseconds())
	trace.last = now
}

func (trace *Trace) End(err error) {
	if trace == nil {
		return
	}
	stage := "complete"
	if err != nil {
		stage = fmt.Sprintf("failed error_type=%T", err)
	}
	trace.Mark(stage)
}
