package connecttrace

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func TestTracePropagatesWithoutMutatingCallerMetadata(t *testing.T) {
	parent := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("other", "retained"))
	ctx, _ := Start(parent, "client")
	md, _ := metadata.FromOutgoingContext(ctx)
	server, _ := Start(metadata.NewIncomingContext(context.Background(), md), "server")
	if ID(server) != ID(ctx) {
		t.Fatal("trace did not cross RPC metadata")
	}
	original, _ := metadata.FromOutgoingContext(parent)
	if len(original.Get(header)) != 0 || md.Get("other")[0] != "retained" {
		t.Fatal("metadata ownership violated")
	}
}

func TestTraceRejectsUntrustedInvalidIDs(t *testing.T) {
	ctx, _ := Start(metadata.NewIncomingContext(context.Background(), metadata.Pairs(header, "invalid\nsecret")), "server")
	if _, err := uuid.Parse(ID(ctx)); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentComponentsHaveDistinctSpansWithinOneTrace(t *testing.T) {
	ctx, parent := Start(context.Background(), "cloud_route")
	_, first := Start(ctx, "edge_session")
	_, second := Start(ctx, "edge_session")
	if parent.id != first.id || first.id != second.id {
		t.Fatal("attempts lost shared trace identity")
	}
	seen := map[string]bool{}
	for _, trace := range []*Trace{parent, first, second} {
		if id, err := uuid.Parse(trace.spanID); err != nil || id == uuid.Nil || seen[trace.spanID] {
			t.Fatal("attempt span is invalid or reused")
		}
		seen[trace.spanID] = true
	}
}
