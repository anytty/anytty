package core

import (
	"context"
	"strings"
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
	EventStorageChanged          EventType = "storage.changed"
)

type Event struct {
	Type       EventType
	TerminalID string
	Terminal   *TerminalInfo
	Storage    *StorageChanged
	Live       *LiveScreenInvalidated
	// 中文说明：true 表示该事件承载 terminal lifecycle 变化，而不是普通 live 输出刷新。
	LifecycleKnown bool
	SocketPath     string
	OldSize        Size
	NewSize        Size
	Timestamp      time.Time
}

type EventFilter struct {
	TerminalID       string
	Types            []EventType
	StorageAppID     string
	StorageScope     StorageScope
	StorageOwnerID   string
	StorageKeyPrefix string
}

type eventBroker struct {
	mu          sync.RWMutex
	buffer      int
	subscribers map[uint64]eventSubscription
	nextID      uint64
	closed      bool
}

type eventSubscription struct {
	filter EventFilter
	ch     chan Event
}

func newEventBroker(buffer int) *eventBroker {
	if buffer <= 0 {
		buffer = 1
	}
	return &eventBroker{
		buffer:      buffer,
		subscribers: make(map[uint64]eventSubscription),
	}
}

func (broker *eventBroker) subscribe(ctx context.Context, filter EventFilter) <-chan Event {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	ch := make(chan Event, broker.buffer)
	if broker.closed {
		close(ch)
		return ch
	}
	broker.nextID++
	id := broker.nextID
	broker.subscribers[id] = eventSubscription{filter: filter, ch: ch}
	go func() {
		<-ctx.Done()
		broker.unsubscribe(id)
	}()
	return ch
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
		select {
		case sub.ch <- cloneEvent(event):
		default:
			// 事件流是通知边界，满队列不阻塞 daemon 主路径。
		}
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
	close(sub.ch)
}

func (broker *eventBroker) close() {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if broker.closed {
		return
	}
	broker.closed = true
	for id, sub := range broker.subscribers {
		delete(broker.subscribers, id)
		close(sub.ch)
	}
}

func eventMatchesFilter(event Event, filter EventFilter) bool {
	if filter.TerminalID != "" && filter.TerminalID != event.TerminalID {
		return false
	}
	if event.Type == EventStorageChanged && !storageEventMatchesFilter(event.Storage, filter) {
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

func storageEventMatchesFilter(storage *StorageChanged, filter EventFilter) bool {
	if storage == nil {
		return filter.StorageAppID == "" && filter.StorageScope == "" && filter.StorageOwnerID == "" && filter.StorageKeyPrefix == ""
	}
	if filter.StorageAppID != "" && filter.StorageAppID != storage.AppID {
		return false
	}
	if filter.StorageScope != "" && filter.StorageScope != storage.Scope {
		return false
	}
	if filter.StorageOwnerID != "" && filter.StorageOwnerID != storage.OwnerID {
		return false
	}
	if filter.StorageKeyPrefix != "" && !strings.HasPrefix(storage.Key, filter.StorageKeyPrefix) {
		return false
	}
	return true
}

func cloneEvent(event Event) Event {
	if event.Terminal != nil {
		terminal := event.Terminal.Clone()
		event.Terminal = &terminal
	}
	if event.Storage != nil {
		storage := *event.Storage
		event.Storage = &storage
	}
	if event.Live != nil {
		live := *event.Live
		event.Live = &live
	}
	return event
}
