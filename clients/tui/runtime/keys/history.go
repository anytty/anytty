package keys

import "sync"

// DefaultLimit is the PROTOCOL §9.3 retention: the host keeps the last 64
// key/paste events per view so a program can call input.forward on them.
const DefaultLimit = 64

// History retains the most recent key/paste events per view by event id, in
// delivery order. The zero value is not usable; use NewHistory. All methods
// are safe for concurrent use.
type History struct {
	mu     sync.Mutex
	limit  int
	order  []string
	events map[string]Event
}

// NewHistory returns a history bounded to limit entries (<=0 means
// DefaultLimit).
func NewHistory(limit int) *History {
	if limit <= 0 {
		limit = DefaultLimit
	}
	return &History{limit: limit, events: map[string]Event{}}
}

// Add stores one event under id and reports whether it was retained. An
// empty id is rejected; re-adding an existing id refreshes its payload
// without changing its position. The oldest entry is evicted past the limit.
func (h *History) Add(id string, ev Event) bool {
	if id == "" {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.events[id]; !ok {
		h.order = append(h.order, id)
	}
	h.events[id] = ev
	for len(h.order) > h.limit {
		oldest := h.order[0]
		h.order = h.order[1:]
		delete(h.events, oldest)
	}
	return true
}

// Get returns the event stored under id; ok is false once it has expired.
func (h *History) Get(id string) (Event, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ev, ok := h.events[id]
	return ev, ok
}

// Len returns how many events are currently retained.
func (h *History) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.order)
}
