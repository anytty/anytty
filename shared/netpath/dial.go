package netpath

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/shared/connecttrace"
)

// Bound process-wide work across Controller, Edge and Direct races.
var socketBudget = make(chan struct{}, 12)

type success struct {
	path string
	at   time.Time
}
type Dialer struct {
	Paths  func() []Path
	dial   func(context.Context, Path, string, string) (net.Conn, error)
	mu     sync.Mutex
	recent map[string]success
}

var Default = &Dialer{Paths: Paths}

type Handshake func(context.Context, net.Conn) (net.Conn, error)

func (dialer *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return dialer.Connect(ctx, network, address, nil)
}

// Connect races real, reusable connections, including the supplied handshake.
// Three paths run immediately; failures free slots for remaining interfaces.
func (dialer *Dialer) Connect(ctx context.Context, network, address string, handshake Handshake) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}
	paths := append([]Path(nil), dialer.Paths()...)
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(strings.Split(host, "%")[0])
	if ip != nil && ip.IsLoopback() {
		paths = []Path{{ID: "default", Name: "system default"}}
	}
	filtered := paths[:0]
	for _, path := range paths {
		if path.ID != "default" && ip != nil {
			family := false
			for _, source := range path.Addresses {
				if (source.IP.To4() != nil) == (ip.To4() != nil) {
					family = true
				}
			}
			if !family {
				continue
			}
		}
		filtered = append(filtered, path)
	}
	paths = filtered
	if len(paths) == 0 {
		return nil, fmt.Errorf("no usable network for %s", address)
	}
	var topology []string
	for _, path := range paths {
		topology = append(topology, path.Key())
	}
	key := network + "/" + address + "/" + strings.Join(topology, ";")
	dialer.mu.Lock()
	preferred := dialer.recent[key]
	dialer.mu.Unlock()
	if time.Since(preferred.at) < 30*time.Second {
		sort.SliceStable(paths, func(i, j int) bool { return paths[i].ID == preferred.path && paths[j].ID != preferred.path })
	}
	race, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		conn net.Conn
		err  error
		path Path
	}
	results := make(chan result, 3)
	next, active := 0, 0
	launch := func(path Path) {
		active++
		go func() {
			queuedAt := time.Now()
			attempt, stop := context.WithTimeout(race, 4*time.Second)
			defer stop()
			select {
			case socketBudget <- struct{}{}:
			case <-attempt.Done():
				log.Printf("anytty network attempt trace_id=%s path=%s target=%s stage=budget_wait elapsed_ms=%d error=%v", connecttrace.ID(ctx), path.Name, address, time.Since(queuedAt).Milliseconds(), attempt.Err())
				results <- result{err: attempt.Err(), path: path}
				return
			}
			defer func() { <-socketBudget }()
			started := time.Now()
			queueMS := started.Sub(queuedAt).Milliseconds()
			connect := dialer.dial
			if connect == nil {
				connect = func(ctx context.Context, path Path, network, address string) (net.Conn, error) {
					return path.Dialer().DialContext(ctx, network, address)
				}
			}
			conn, err := connect(attempt, path, network, address)
			dialMS := time.Since(started).Milliseconds()
			handshakeStarted := time.Now()
			if err == nil && handshake != nil {
				raw := conn
				conn, err = handshake(attempt, raw)
				if err != nil {
					_ = raw.Close()
				}
			}
			if err != nil && conn != nil {
				_ = conn.Close()
				conn = nil
			}
			outcome := "success"
			if err != nil {
				outcome = "failed"
			}
			if errors.Is(err, context.Canceled) {
				outcome = "canceled"
			}
			log.Printf("anytty network attempt trace_id=%s path=%s target=%s queue_ms=%d dns_tcp_ms=%d tls_ms=%d elapsed_ms=%d outcome=%s success=%t error=%v", connecttrace.ID(ctx), path.Name, address, queueMS, dialMS, time.Since(handshakeStarted).Milliseconds(), time.Since(started).Milliseconds(), outcome, err == nil, err)
			results <- result{conn: conn, err: err, path: path}
		}()
	}
	drain := func() {
		cancel()
		remaining := active
		go func() {
			for ; remaining > 0; remaining-- {
				r := <-results
				if r.conn != nil {
					_ = r.conn.Close()
				}
			}
		}()
	}
	var failures []error
	for {
		for active < 3 && next < len(paths) {
			launch(paths[next])
			next++
		}
		if active == 0 {
			return nil, errors.Join(failures...)
		}
		select {
		case <-ctx.Done():
			drain()
			return nil, errors.Join(append(failures, ctx.Err())...)
		case r := <-results:
			active--
			if r.err == nil && r.conn != nil {
				if ctx.Err() != nil {
					_ = r.conn.Close()
					drain()
					return nil, ctx.Err()
				}
				dialer.mu.Lock()
				if len(dialer.recent) >= 128 || dialer.recent == nil {
					dialer.recent = make(map[string]success)
				}
				dialer.recent[key] = success{path: r.path.ID, at: time.Now()}
				dialer.mu.Unlock()
				drain()
				return r.conn, nil
			}
			failures = append(failures, fmt.Errorf("network %s: %w", r.path.Name, r.err))
		}
	}
}
