package core

import (
	"context"
	"sync"
	"time"
)

type EventType string

const (
	EventServerListening         EventType = "server.listening"
	EventServerStopped           EventType = "server.stopped"
	EventTerminalCreated         EventType = "terminal.created"
	EventTerminalExited          EventType = "terminal.exited"
	EventTerminalResized         EventType = "terminal.resized"
	EventTerminalMetadataChanged EventType = "terminal.metadata_changed"
	EventTerminalRemoved         EventType = "terminal.removed"
	EventTerminalChanged         EventType = "terminal.changed"
	EventTerminalLiveInvalidated EventType = "terminal.live.invalidated"
)

type Event struct {
	Type       EventType
	TerminalID string
	Terminal   *TerminalInfo
	Attachment *TerminalAttachmentProjection
	Live       *LiveScreenInvalidated
	// 中文说明：true 表示该事件承载 terminal lifecycle 变化，而不是普通 live 输出刷新。
	LifecycleKnown bool
	SocketPath     string
	OldSize        Size
	NewSize        Size
	Timestamp      time.Time
}

// TerminalAttachmentProjection is the pool-authoritative attached-view count
// and resize-owner snapshot published when the attachment registry changes.
type TerminalAttachmentProjection struct {
	AttachmentCount int
	ResizeEpoch     uint64
	ResizeControl   *TerminalResizeControl
}

type EventFilter struct {
	TerminalID string
	Types      []EventType
}

type eventBroker struct {
	mu          sync.RWMutex
	buffer      int
	subscribers map[uint64]*eventSubscription
	nextID      uint64
	closed      bool
}

type eventSubscription struct {
	filter   EventFilter
	ch       chan Event
	wake     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	closed   bool
	inFlight bool
	pending  map[string]Event
	order    []string
}

func newEventBroker(buffer int) *eventBroker {
	if buffer <= 0 {
		buffer = 1
	}
	return &eventBroker{
		buffer:      buffer,
		subscribers: make(map[uint64]*eventSubscription),
	}
}

func (broker *eventBroker) subscribe(ctx context.Context, filter EventFilter) <-chan Event {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if broker.closed {
		ch := make(chan Event)
		close(ch)
		return ch
	}
	broker.nextID++
	id := broker.nextID
	sub := &eventSubscription{
		filter:  filter,
		ch:      make(chan Event, broker.buffer),
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
		pending: make(map[string]Event),
	}
	broker.subscribers[id] = sub
	go sub.run()
	go func() {
		select {
		case <-ctx.Done():
			broker.unsubscribe(id)
		case <-sub.done:
		}
	}()
	return sub.ch
}

func (broker *eventBroker) publish(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	broker.mu.RLock()
	defer broker.mu.RUnlock()
	if broker.closed {
		return
	}
	for _, sub := range broker.subscribers {
		if !eventMatchesFilter(event, sub.filter) {
			continue
		}
		sub.enqueue(cloneEvent(event))
	}
}

func (broker *eventBroker) unsubscribe(id uint64) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	sub, ok := broker.subscribers[id]
	if !ok {
		return
	}
	delete(broker.subscribers, id)
	sub.stop()
}

func (broker *eventBroker) close() {
	broker.mu.Lock()
	if broker.closed {
		broker.mu.Unlock()
		return
	}
	broker.closed = true
	subs := make([]*eventSubscription, 0, len(broker.subscribers))
	for id, sub := range broker.subscribers {
		delete(broker.subscribers, id)
		subs = append(subs, sub)
	}
	broker.mu.Unlock()
	for _, sub := range subs {
		sub.stop()
	}
}

func (sub *eventSubscription) enqueue(event Event) {
	key := eventDeliveryKey(event)
	sub.mu.Lock()
	if sub.closed {
		sub.mu.Unlock()
		return
	}
	if !sub.inFlight && len(sub.order) == 0 {
		select {
		case sub.ch <- event:
			sub.mu.Unlock()
			return
		default:
		}
	}
	if key == "" {
		sub.mu.Unlock()
		return
	}
	if _, exists := sub.pending[key]; !exists {
		sub.order = append(sub.order, key)
	}
	sub.pending[key] = event
	sub.mu.Unlock()
	select {
	case sub.wake <- struct{}{}:
	default:
	}
}

func (sub *eventSubscription) run() {
	defer close(sub.ch)
	for {
		event, ok := sub.pop()
		if !ok {
			select {
			case <-sub.wake:
				continue
			case <-sub.done:
				return
			}
		}
		select {
		case sub.ch <- event:
			sub.deliveryComplete()
		case <-sub.done:
			return
		}
	}
}

func (sub *eventSubscription) pop() (Event, bool) {
	sub.mu.Lock()
	defer sub.mu.Unlock()
	if len(sub.order) == 0 {
		return Event{}, false
	}
	key := sub.order[0]
	sub.order = sub.order[1:]
	event := sub.pending[key]
	delete(sub.pending, key)
	sub.inFlight = true
	return event, true
}

func (sub *eventSubscription) deliveryComplete() {
	sub.mu.Lock()
	sub.inFlight = false
	sub.mu.Unlock()
}

func (sub *eventSubscription) stop() {
	sub.mu.Lock()
	if sub.closed {
		sub.mu.Unlock()
		return
	}
	sub.closed = true
	sub.pending = nil
	sub.order = nil
	close(sub.done)
	sub.mu.Unlock()
}

func eventDeliveryKey(event Event) string {
	if event.Attachment != nil {
		return "terminal.attachment\x00" + event.TerminalID
	}
	return ""
}

func eventMatchesFilter(event Event, filter EventFilter) bool {
	if filter.TerminalID != "" && filter.TerminalID != event.TerminalID {
		return false
	}
	if len(filter.Types) == 0 {
		return true
	}
	for _, typ := range filter.Types {
		if typ == event.Type {
			return true
		}
	}
	return false
}

func cloneEvent(event Event) Event {
	if event.Terminal != nil {
		terminal := event.Terminal.Clone()
		event.Terminal = &terminal
	}
	if event.Attachment != nil {
		attachment := *event.Attachment
		if event.Attachment.ResizeControl != nil {
			control := *event.Attachment.ResizeControl
			if event.Attachment.ResizeControl.ResizeOwnership != nil {
				ownership := *event.Attachment.ResizeControl.ResizeOwnership
				control.ResizeOwnership = &ownership
			}
			attachment.ResizeControl = &control
		}
		event.Attachment = &attachment
	}
	if event.Live != nil {
		live := *event.Live
		event.Live = &live
	}
	return event
}
