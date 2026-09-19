package client

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

func testTransport() (*grpc.ClientConn, error) {
	return grpc.NewClient("passthrough:///127.0.0.1:1", grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func TestTransportPoolConcurrentLeasesShareAndReleaseIndependently(t *testing.T) {
	pool := NewTransportPool()
	defer pool.Close()
	pool.idleTimeout = 30 * time.Millisecond
	var creates atomic.Int32
	create := func() (*grpc.ClientConn, error) { creates.Add(1); return testTransport() }
	first, release, err := pool.acquire(context.Background(), "edge", create)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, done, err := pool.acquire(context.Background(), "edge", create)
			if err != nil {
				t.Error(err)
				return
			}
			if conn != first {
				t.Error("parallel caller did not reuse transport")
			}
			done()
			done()
		}()
	}
	wg.Wait()
	if creates.Load() != 1 || first.GetState() == connectivity.Shutdown {
		t.Fatal("sibling release closed the active lease")
	}
	release()
	deadline := time.Now().Add(time.Second)
	for first.GetState() != connectivity.Shutdown && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if first.GetState() != connectivity.Shutdown {
		t.Fatal("idle transport was not closed")
	}
}

func TestTransportPoolReuseCancelsIdleExpiryAndCloseFencesAcquire(t *testing.T) {
	pool := NewTransportPool()
	pool.idleTimeout = 20 * time.Millisecond
	first, release, err := pool.acquire(context.Background(), "edge", testTransport)
	if err != nil {
		t.Fatal(err)
	}
	release()
	second, done, err := pool.acquire(context.Background(), "edge", testTransport)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	time.Sleep(40 * time.Millisecond)
	if first != second || second.GetState() == connectivity.Shutdown {
		t.Fatal("reacquired transport expired")
	}
	_ = pool.Close()
	if second.GetState() != connectivity.Shutdown {
		t.Fatal("pool close left transport open")
	}
	if _, _, err := pool.acquire(context.Background(), "edge", testTransport); err == nil {
		t.Fatal("closed pool accepted a lease")
	}
}

func TestTransportKeySeparatesTrustAndScope(t *testing.T) {
	key := transportKey("profile-a/edge", "edge:443", "edge", []byte("ca-a"))
	for _, other := range []string{
		transportKey("profile-b/edge", "edge:443", "edge", []byte("ca-a")),
		transportKey("profile-a/edge", "edge:443", "other", []byte("ca-a")),
		transportKey("profile-a/edge", "edge:443", "edge", []byte("ca-b")),
	} {
		if key == other {
			t.Fatal("trust or scope change reused a cache key")
		}
	}
}

func TestTransportPoolIgnoresAlreadyFiredOldIdleTimer(t *testing.T) {
	pool := NewTransportPool()
	defer pool.Close()
	conn, release, err := pool.acquire(context.Background(), "edge", testTransport)
	if err != nil {
		t.Fatal(err)
	}
	entry := pool.entries["edge"]
	epoch := entry.idleEpoch
	release()
	_, secondRelease, err := pool.acquire(context.Background(), "edge", testTransport)
	if err != nil {
		t.Fatal(err)
	}
	secondRelease()
	pool.expire("edge", entry, epoch)
	if conn.GetState() == connectivity.Shutdown {
		t.Fatal("old idle timer closed a newly released transport")
	}
}
