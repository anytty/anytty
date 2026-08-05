package core

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/shared/transport"
)

var ErrGrantTransportInactive = errors.New("grant transport is not active")

type transportCloseReason uint8

const (
	transportCloseNormal transportCloseReason = iota + 1
	transportCloseRevoked
	transportCloseExpired
	transportCloseShutdown
)

type trackedTransport struct {
	transport.Transport
	closeOnce      sync.Once
	finishOnce     sync.Once
	closeErr       error
	reason         transportCloseReason
	grantOperation atomic.Pointer[grantOperation]
}

func (tracked *trackedTransport) Close() error {
	return tracked.closeWithReason(transportCloseNormal)
}

func (tracked *trackedTransport) closeWithReason(reason transportCloseReason) error {
	if tracked == nil {
		return nil
	}
	tracked.closeOnce.Do(func() {
		tracked.reason = reason
		tracked.closeErr = tracked.Transport.Close()
	})
	return tracked.closeErr
}

type grantTimer interface {
	Stop() bool
}

type grantTimerEntry struct {
	expiresAt time.Time
	timer     grantTimer
}

// grantOperation is the only coordination state shared by operations for one
// canonical GrantID. Grant cleanup uses grantOperationsMu -> operation.mu, while
// transport finish uses the operation.mu -> server.mu suffix. Admission uses the
// separate wgMu -> server.mu barrier before reaching grant state. Store calls and
// transport Close calls run without any of these locks held.
type grantOperation struct {
	grantID    string
	mu         sync.Mutex
	epoch      uint64
	transports map[*trackedTransport]struct{}
	timer      *grantTimerEntry
	refs       int // protected by Server.grantOperationsMu
}

func (server *Server) beginTrackedTransport(connection transport.Transport) (*trackedTransport, error) {
	tracked := &trackedTransport{Transport: connection}
	server.wgMu.Lock()
	server.mu.Lock()
	if server.closed.Load() {
		server.mu.Unlock()
		server.wgMu.Unlock()
		_ = tracked.closeWithReason(transportCloseShutdown)
		return nil, ErrServerClosed
	}
	server.transports[tracked] = struct{}{}
	server.wg.Add(1)
	server.mu.Unlock()
	server.wgMu.Unlock()
	return tracked, nil
}

func (server *Server) admitTransport(ctx context.Context, tracked *trackedTransport, scope TransportScope) error {
	if scope.LocalOwner {
		return nil
	}

	operation := server.acquireGrantOperation(scope.GrantID)
	defer server.releaseGrantOperation(operation)
	operation.mu.Lock()
	epoch := operation.epoch
	operation.mu.Unlock()

	queryNow := server.grantNow().UTC()
	service := server.ClientAccessService()
	active := queryNow.Before(scope.GrantExpiresAt) && service != nil &&
		service.GrantActive(ctx, scope.GrantID, scope.GrantExpiresAt, queryNow)

	var detached []*trackedTransport
	closeReason := transportCloseRevoked
	operation.mu.Lock()
	finalNow := server.grantNow().UTC()
	switch {
	case server.closed.Load():
		closeReason = transportCloseShutdown
	case !finalNow.Before(scope.GrantExpiresAt):
		operation.epoch++
		detached = detachGrantTransportsLocked(operation)
		closeReason = transportCloseExpired
	case operation.epoch != epoch || !active:
		closeReason = transportCloseRevoked
	default:
		tracked.grantOperation.Store(operation)
		operation.transports[tracked] = struct{}{}
		scheduleGrantExpiryLocked(server, operation, scope.GrantExpiresAt, finalNow)
		operation.mu.Unlock()
		return nil
	}
	operation.mu.Unlock()
	closeTrackedTransports(detached, transportCloseExpired)
	_ = tracked.closeWithReason(closeReason)
	if closeReason == transportCloseShutdown {
		return ErrServerClosed
	}
	return ErrGrantTransportInactive
}

func (server *Server) untrackTransport(tracked *trackedTransport) {
	operation := tracked.grantOperation.Swap(nil)
	if operation != nil {
		operation.mu.Lock()
		delete(operation.transports, tracked)
		if len(operation.transports) == 0 && operation.timer != nil {
			operation.timer.timer.Stop()
			operation.timer = nil
		}
		operation.mu.Unlock()
		server.pruneGrantOperation(operation)
	}
	server.mu.Lock()
	delete(server.transports, tracked)
	server.mu.Unlock()
}

func (server *Server) finishTrackedTransport(tracked *trackedTransport) {
	if tracked == nil {
		return
	}
	tracked.finishOnce.Do(func() {
		server.untrackTransport(tracked)
		_ = tracked.Close()
		server.wg.Done()
	})
}

func scheduleGrantExpiryLocked(server *Server, operation *grantOperation, expiresAt, now time.Time) {
	if operation.timer != nil {
		return
	}
	entry := &grantTimerEntry{expiresAt: expiresAt}
	operation.timer = entry
	entry.timer = server.grantAfterFunc(expiresAt.Sub(now), func() {
		server.expireGrantOperation(operation, entry)
	})
}

func (server *Server) expireGrantOperation(operation *grantOperation, entry *grantTimerEntry) {
	now := server.grantNow().UTC()
	operation.mu.Lock()
	if operation.timer != entry {
		operation.mu.Unlock()
		return
	}
	if now.Before(entry.expiresAt) {
		entry.timer = server.grantAfterFunc(entry.expiresAt.Sub(now), func() {
			server.expireGrantOperation(operation, entry)
		})
		operation.mu.Unlock()
		return
	}
	operation.epoch++
	transports := detachGrantTransportsLocked(operation)
	operation.mu.Unlock()
	closeTrackedTransports(transports, transportCloseExpired)
	server.pruneGrantOperation(operation)
}

func (server *Server) revokeClientAccess(ctx context.Context, grantID string) (ClientAccessRecord, error) {
	service := server.ClientAccessService()
	if service == nil {
		return ClientAccessRecord{}, ErrClientAccessServiceUnavailable
	}
	grantID = strings.TrimSpace(grantID)
	operation := server.acquireGrantOperation(grantID)
	defer server.releaseGrantOperation(operation)

	record, err := service.Revoke(ctx, grantID)
	if err != nil {
		return ClientAccessRecord{}, err
	}
	operation.mu.Lock()
	operation.epoch++
	transports := detachGrantTransportsLocked(operation)
	operation.mu.Unlock()
	closeTrackedTransports(transports, transportCloseRevoked)
	return record, nil
}

func detachGrantTransportsLocked(operation *grantOperation) []*trackedTransport {
	transports := make([]*trackedTransport, 0, len(operation.transports))
	for tracked := range operation.transports {
		transports = append(transports, tracked)
		tracked.grantOperation.CompareAndSwap(operation, nil)
		delete(operation.transports, tracked)
	}
	if operation.timer != nil {
		operation.timer.timer.Stop()
		operation.timer = nil
	}
	return transports
}

func (server *Server) acquireGrantOperation(grantID string) *grantOperation {
	server.grantOperationsMu.Lock()
	operation := server.grantOperations[grantID]
	if operation == nil {
		operation = &grantOperation{grantID: grantID, transports: make(map[*trackedTransport]struct{})}
		server.grantOperations[grantID] = operation
	}
	operation.refs++
	server.grantOperationsMu.Unlock()
	return operation
}

func (server *Server) releaseGrantOperation(operation *grantOperation) {
	server.grantOperationsMu.Lock()
	operation.refs--
	server.pruneGrantOperationLocked(operation)
	server.grantOperationsMu.Unlock()
}

func (server *Server) pruneGrantOperation(operation *grantOperation) {
	server.grantOperationsMu.Lock()
	server.pruneGrantOperationLocked(operation)
	server.grantOperationsMu.Unlock()
}

func (server *Server) pruneGrantOperationLocked(operation *grantOperation) {
	if operation.refs != 0 || server.grantOperations[operation.grantID] != operation {
		return
	}
	operation.mu.Lock()
	idle := len(operation.transports) == 0 && operation.timer == nil
	operation.mu.Unlock()
	if idle {
		delete(server.grantOperations, operation.grantID)
	}
}

func (server *Server) stopGrantOperations() {
	server.grantOperationsMu.Lock()
	operations := make([]*grantOperation, 0, len(server.grantOperations))
	for _, operation := range server.grantOperations {
		operation.refs++
		operations = append(operations, operation)
	}
	server.grantOperationsMu.Unlock()

	var transports []*trackedTransport
	for _, operation := range operations {
		operation.mu.Lock()
		operation.epoch++
		transports = append(transports, detachGrantTransportsLocked(operation)...)
		operation.mu.Unlock()
		server.releaseGrantOperation(operation)
	}
	closeTrackedTransports(transports, transportCloseShutdown)
}

func (server *Server) pruneGrantOperations() {
	server.grantOperationsMu.Lock()
	for _, operation := range server.grantOperations {
		server.pruneGrantOperationLocked(operation)
	}
	server.grantOperationsMu.Unlock()
}

func closeTrackedTransports(transports []*trackedTransport, reason transportCloseReason) {
	for _, tracked := range transports {
		_ = tracked.closeWithReason(reason)
	}
}
