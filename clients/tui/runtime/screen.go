package runtime

import "time"

// ScreenPollInterval is how often WaitScreen re-checks the revision while it
// has not changed yet.
const ScreenPollInterval = 2 * time.Millisecond

// MarkOutput records that a content source produced output. The screen
// revision advances so a host loop can compose and flush a frame even when
// no input arrived (PTY output alone must repaint).
func (s *Session) MarkOutput() {
	s.mu.Lock()
	s.screenRev++
	s.mu.Unlock()
}

// ScreenRevision returns the number of output notifications observed so far.
// Compare two revisions to detect that the screen changed.
func (s *Session) ScreenRevision() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.screenRev
}

// WaitScreen blocks until the screen revision differs from seen or timeout
// elapses, then returns the current revision (== seen on timeout).
//
// It polls every ScreenPollInterval instead of using sync.Cond: Cond has no
// timed wait, and a lost wakeup would stall the painter until the next input
// event. A 2 ms poll is far below a repaint budget and needs no extra
// goroutine or close/broadcast channel bookkeeping.
func (s *Session) WaitScreen(seen uint64, timeout time.Duration) uint64 {
	deadline := time.Now().Add(timeout)
	for {
		if rev := s.ScreenRevision(); rev != seen {
			return rev
		}
		if !time.Now().Before(deadline) {
			return seen
		}
		time.Sleep(ScreenPollInterval)
	}
}
