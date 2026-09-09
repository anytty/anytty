package client

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/shared/connecttrace"
	"github.com/anytty/anytty/shared/netpath"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// TransportPool caches only authenticated TLS transports, never session grants
// or challenge proofs. Streams hold independent leases on a shared connection.
type TransportPool struct {
	mu          sync.Mutex
	entries     map[string]*pooledTransport
	closed      bool
	idleTimeout time.Duration
}

type pooledTransport struct {
	traceID    string
	connection *grpc.ClientConn
	refs       int
	timer      *time.Timer
	idleEpoch  uint64
}

func NewTransportPool() *TransportPool {
	return &TransportPool{entries: make(map[string]*pooledTransport), idleTimeout: 30 * time.Second}
}

func (client *Client) acquireTransport(ctx context.Context, scope, address, serverName string, trust []byte, pinned bool) (*grpc.ClientConn, func(), error) {
	create := func() (*grpc.ClientConn, error) {
		if pinned {
			return client.dialPinned(address, serverName, trust, connecttrace.ID(ctx))
		}
		return client.dial(address, serverName, trust, connecttrace.ID(ctx))
	}
	if client.transports == nil {
		conn, err := create()
		if err != nil {
			return nil, nil, err
		}
		return conn, func() { _ = conn.Close() }, nil
	}
	scope = client.bootID + "/" + client.config.ControllerAddress + "/" + scope
	return client.transports.acquire(ctx, transportKey(scope, strings.TrimSpace(address), strings.TrimSpace(serverName), trust), create)
}

func transportKey(scope, address, serverName string, trust []byte) string {
	paths := netpath.Paths()
	topology := make([]string, 0, len(paths))
	for _, path := range paths {
		topology = append(topology, path.Key())
	}
	sort.Strings(topology)
	return fmt.Sprintf("%s\x00%s\x00%s\x00%x\x00%s", scope, address, serverName, sha256.Sum256(trust), strings.Join(topology, ";"))
}

func (pool *TransportPool) acquire(ctx context.Context, key string, create func() (*grpc.ClientConn, error)) (*grpc.ClientConn, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	pool.mu.Lock()
	if pool.closed {
		pool.mu.Unlock()
		return nil, nil, fmt.Errorf("Cloud transport pool is closed")
	}
	entry := pool.entries[key]
	hit := entry != nil
	if entry != nil && entry.connection.GetState() == connectivity.Shutdown {
		delete(pool.entries, key)
		entry = nil
		hit = false
	}
	if entry == nil {
		// grpc.NewClient is lazy and performs no network I/O under this lock.
		conn, err := create()
		if err != nil {
			pool.mu.Unlock()
			return nil, nil, err
		}
		entry = &pooledTransport{connection: conn, traceID: connecttrace.ID(ctx)}
		if len(pool.entries) < 16 {
			pool.entries[key] = entry
		}
	}
	if entry.timer != nil {
		entry.timer.Stop()
		entry.timer = nil
	}
	entry.refs++
	entry.idleEpoch++
	pool.mu.Unlock()
	log.Printf("anytty cloud transport trace_id=%s transport_trace_id=%s cache_hit=%t state=%s", connecttrace.ID(ctx), entry.traceID, hit, entry.connection.GetState())
	var once sync.Once
	release := func() {
		once.Do(func() {
			pool.mu.Lock()
			defer pool.mu.Unlock()
			entry.refs--
			if entry.refs != 0 {
				return
			}
			if pool.closed || pool.entries[key] != entry {
				_ = entry.connection.Close()
				return
			}
			epoch := entry.idleEpoch
			entry.timer = time.AfterFunc(pool.idleTimeout, func() { pool.expire(key, entry, epoch) })
		})
	}
	return entry.connection, release, nil
}

func (pool *TransportPool) expire(key string, entry *pooledTransport, epoch uint64) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if entry.refs == 0 && entry.idleEpoch == epoch && pool.entries[key] == entry {
		delete(pool.entries, key)
		_ = entry.connection.Close()
	}
}

// Close belongs to the engine/profile owner, never to an individual stream.
func (pool *TransportPool) Close() error {
	if pool == nil {
		return nil
	}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.closed = true
	for key, entry := range pool.entries {
		if entry.timer != nil {
			entry.timer.Stop()
		}
		_ = entry.connection.Close()
		delete(pool.entries, key)
	}
	return nil
}
