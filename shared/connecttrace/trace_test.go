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
