package core

import (
	"sync"
	"time"
)

const (
	liveScreenBaselineTTL             = 2 * time.Second
	maxLiveScreenBaselineEntries      = 64
	maxLiveScreenBaselineSessionBytes = 1 << 20
	maxLiveScreenBaselineServerBytes  = 64 << 20
)

type cachedLiveScreenBaseline struct {
	screen    *nativeScreenBaseline
	bytes     int64
	pins      int
	expiresAt time.Time
}

type sessionLiveScreenBaselines struct {
	confirmed *cachedLiveScreenBaseline
	offered   *cachedLiveScreenBaseline
}

func (session *protocolSession) acquireLiveScreenBaseline(terminalID string, observed LiveRevision) (*nativeScreenBaseline, func()) {
	if session == nil || terminalID == "" || observed == 0 {
		return nil, func() {}
	}
	now := time.Now()
	session.liveBaselineMu.Lock()
	session.pruneLiveScreenBaselinesLocked(now)
	entry := session.liveBaselines[terminalID]
	if entry == nil {
		session.liveBaselineMu.Unlock()
		return nil, func() {}
	}
	if entry.offered != nil && entry.offered.screen.revision == observed {
		if entry.confirmed != nil && entry.confirmed.pins > 0 {
			session.liveBaselineMu.Unlock()
			return nil, func() {}
		}
		session.dropCachedLiveScreenBaselineLocked(entry.confirmed)
		entry.confirmed = entry.offered
		entry.offered = nil
	}
	cached := entry.confirmed
	if cached == nil || cached.screen.revision != observed {
		session.liveBaselineMu.Unlock()
		return nil, func() {}
	}
	cached.pins++
	cached.expiresAt = time.Time{}
	screen := cached.screen
	session.ensureLiveScreenBaselineTimerLocked()
	session.liveBaselineMu.Unlock()

	var once sync.Once
	return screen, func() {
		once.Do(func() {
			session.liveBaselineMu.Lock()
			if cached.pins > 0 {
				cached.pins--
			}
			if cached.pins == 0 {
				cached.expiresAt = time.Now().Add(liveScreenBaselineTTL)
			}
			session.ensureLiveScreenBaselineTimerLocked()
			session.liveBaselineMu.Unlock()
		})
	}
}

func (session *protocolSession) offerLiveScreenBaseline(terminalID string, screen *nativeScreenBaseline) {
	if session == nil || session.server == nil || terminalID == "" || screen == nil {
		return
	}
	bytes := int64(len(screen.rowHashes))*8 + 64
	if bytes <= 0 || bytes > maxLiveScreenBaselineSessionBytes {
		return
	}
	now := time.Now()
	session.liveBaselineMu.Lock()
	defer session.liveBaselineMu.Unlock()
	if session.liveBaselineClosed {
		return
	}
	session.pruneLiveScreenBaselinesLocked(now)
	entry := session.liveBaselines[terminalID]
	if entry == nil {
		if len(session.liveBaselines) >= maxLiveScreenBaselineEntries {
			return
		}
		entry = &sessionLiveScreenBaselines{}
		session.liveBaselines[terminalID] = entry
	}
	if entry.offered != nil {
		if entry.offered.pins > 0 || entry.offered.screen.revision > screen.revision {
			return
		}
		session.dropCachedLiveScreenBaselineLocked(entry.offered)
		entry.offered = nil
	}
	if session.liveBaselineBytes+bytes > maxLiveScreenBaselineSessionBytes || !session.server.reserveLiveScreenBaselineBytes(bytes) {
		if entry.confirmed == nil {
			delete(session.liveBaselines, terminalID)
		}
		return
	}
	session.liveBaselineBytes += bytes
	entry.offered = &cachedLiveScreenBaseline{screen: screen, bytes: bytes, expiresAt: now.Add(liveScreenBaselineTTL)}
	session.ensureLiveScreenBaselineTimerLocked()
}

func (server *Server) reserveLiveScreenBaselineBytes(bytes int64) bool {
	if server == nil || bytes <= 0 {
		return false
	}
	for {
		current := server.liveBaselineBytes.Load()
		if current+bytes > maxLiveScreenBaselineServerBytes {
			return false
		}
		if server.liveBaselineBytes.CompareAndSwap(current, current+bytes) {
			return true
		}
	}
}

func (session *protocolSession) dropCachedLiveScreenBaselineLocked(cached *cachedLiveScreenBaseline) {
	if cached == nil || cached.bytes <= 0 {
		return
	}
	session.liveBaselineBytes -= cached.bytes
	if session.liveBaselineBytes < 0 {
		session.liveBaselineBytes = 0
	}
	if session.server != nil {
		session.server.liveBaselineBytes.Add(-cached.bytes)
	}
	cached.bytes = 0
}

func (session *protocolSession) pruneLiveScreenBaselinesLocked(now time.Time) {
	for terminalID, entry := range session.liveBaselines {
		if cachedLiveScreenBaselineExpired(entry.confirmed, now) {
			session.dropCachedLiveScreenBaselineLocked(entry.confirmed)
			entry.confirmed = nil
		}
		if cachedLiveScreenBaselineExpired(entry.offered, now) {
			session.dropCachedLiveScreenBaselineLocked(entry.offered)
			entry.offered = nil
		}
		if entry.confirmed == nil && entry.offered == nil {
			delete(session.liveBaselines, terminalID)
		}
	}
}

func cachedLiveScreenBaselineExpired(cached *cachedLiveScreenBaseline, now time.Time) bool {
	return cached != nil && cached.pins == 0 && !cached.expiresAt.IsZero() && !now.Before(cached.expiresAt)
}

func (session *protocolSession) ensureLiveScreenBaselineTimerLocked() {
	if session.liveBaselineClosed || session.liveBaselineTimer != nil || len(session.liveBaselines) == 0 {
		return
	}
	session.liveBaselineTimer = time.AfterFunc(liveScreenBaselineTTL, session.expireLiveScreenBaselines)
}

func (session *protocolSession) expireLiveScreenBaselines() {
	session.liveBaselineMu.Lock()
	session.liveBaselineTimer = nil
	if !session.liveBaselineClosed {
		session.pruneLiveScreenBaselinesLocked(time.Now())
		session.ensureLiveScreenBaselineTimerLocked()
	}
	session.liveBaselineMu.Unlock()
}

func (session *protocolSession) clearLiveScreenBaselines() {
	if session == nil {
		return
	}
	session.liveBaselineMu.Lock()
	session.liveBaselineClosed = true
	if session.liveBaselineTimer != nil {
		session.liveBaselineTimer.Stop()
		session.liveBaselineTimer = nil
	}
	for terminalID, entry := range session.liveBaselines {
		session.dropCachedLiveScreenBaselineLocked(entry.confirmed)
		session.dropCachedLiveScreenBaselineLocked(entry.offered)
		delete(session.liveBaselines, terminalID)
	}
	session.liveBaselineMu.Unlock()
}
