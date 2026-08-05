package app

import (
	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/tui/state"
)

func (runtime *AppRuntime) enqueue(msg Msg) {
	if msg == nil {
		return
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.coalesceQueuedHostResize(msg) {
		runtime.signalWakeLocked()
		return
	}
	if runtime.coalesceQueuedLiveUpdate(msg) {
		runtime.signalWakeLocked()
		return
	}
	if runtime.coalesceQueuedWorkbenchStorage(msg) {
		runtime.signalWakeLocked()
		return
	}
	if runtime.prioritizeQueuedInputLocked(msg) {
		runtime.signalWakeLocked()
		return
	}
	runtime.queue = append(runtime.queue, msg)
	runtime.signalWakeLocked()
}

func (runtime *AppRuntime) prioritizeQueuedInputLocked(msg Msg) bool {
	if _, ok := msg.(InputMsg); !ok {
		return false
	}
	index := firstOrdinaryLiveUpdateIndex(runtime.queue)
	if index < 0 {
		return false
	}
	runtime.insertAtLocked(index, msg)
	return true
}

func (runtime *AppRuntime) coalesceQueuedHostResize(msg Msg) bool {
	incoming, ok := msg.(HostResizeMsg)
	if !ok {
		return false
	}
	first := -1
	original := runtime.queue
	filtered := original[:0]
	for _, queued := range original {
		if _, ok := queued.(HostResizeMsg); ok {
			if first < 0 {
				first = len(filtered)
			}
			continue
		}
		filtered = append(filtered, queued)
	}
	runtime.setQueueFilteredLocked(original, filtered)
	if first < 0 {
		return false
	}
	runtime.insertAtLocked(first, incoming)
	return true
}

func (runtime *AppRuntime) dropQueuedHostResizeLocked() {
	original := runtime.queue
	filtered := original[:0]
	for _, queued := range original {
		if _, ok := queued.(HostResizeMsg); ok {
			continue
		}
		filtered = append(filtered, queued)
	}
	runtime.setQueueFilteredLocked(original, filtered)
}

func (runtime *AppRuntime) insertAtLocked(index int, msg Msg) {
	if index < 0 {
		index = 0
	}
	if index >= len(runtime.queue) {
		runtime.queue = append(runtime.queue, msg)
		return
	}
	runtime.queue = append(runtime.queue, nil)
	copy(runtime.queue[index+1:], runtime.queue[index:])
	runtime.queue[index] = msg
}

func (runtime *AppRuntime) coalesceQueuedLiveUpdate(msg Msg) bool {
	incoming, ok := queuedOrdinaryLiveUpdate(msg)
	if !ok {
		return false
	}
	for i := len(runtime.queue) - 1; i >= 0; i-- {
		queued := runtime.queue[i]
		// 普通 live 帧只能在同一语义基线内合并，不能跨过 resize/exit/attach 等边界。
		if liveQueueBoundary(queued) {
			return false
		}
		existing, ok := queuedOrdinaryLiveUpdate(queued)
		if !ok {
			return false
		}
		// 中文说明：TerminalID 只在 owning endpoint 内唯一；
		// 普通 live 帧合并必须按 TerminalRef，不能吞掉远端同名 terminal 的后续输出。
		if !existing.ref.Equal(incoming.ref) {
			continue
		}
		if existing.kind != incoming.kind {
			// 中文说明：refresh wake 和 surface result 虽然都不要求逐帧渲染，
			// 但它们释放不同 reducer-owned 背压状态，不能互相 latest-only 替换。
			return false
		}
		perftrace.Count("tui.queue_live_coalesce", messageApproxBytes(queued))
		runtime.removeAtLocked(i)
		runtime.queue = append(runtime.queue, msg)
		return true
	}
	return false
}

func (runtime *AppRuntime) coalesceQueuedWorkbenchStorage(msg Msg) bool {
	switch msg := msg.(type) {
	case WorkbenchStorageLoadRequestMsg:
		return runtime.replaceQueuedWorkbenchStorageLoadLocked(msg)
	case WorkbenchStoragePersistRequestMsg:
		return runtime.replaceQueuedWorkbenchStoragePersistLocked(msg)
	default:
		return false
	}
}

func (runtime *AppRuntime) replaceQueuedWorkbenchStorageLoadLocked(msg WorkbenchStorageLoadRequestMsg) bool {
	first := -1
	original := runtime.queue
	filtered := original[:0]
	for _, queued := range original {
		if _, ok := queued.(WorkbenchStorageLoadRequestMsg); ok {
			if first < 0 {
				first = len(filtered)
			}
			continue
		}
		filtered = append(filtered, queued)
	}
	if first < 0 {
		return false
	}
	runtime.setQueueFilteredLocked(original, filtered)
	runtime.queue = append(runtime.queue, nil)
	copy(runtime.queue[first+1:], runtime.queue[first:])
	runtime.queue[first] = msg
	return true
}

func (runtime *AppRuntime) replaceQueuedWorkbenchStoragePersistLocked(msg WorkbenchStoragePersistRequestMsg) bool {
	first := -1
	original := runtime.queue
	filtered := original[:0]
	for _, queued := range original {
		if _, ok := queued.(WorkbenchStoragePersistRequestMsg); ok {
			if first < 0 {
				first = len(filtered)
			}
			continue
		}
		filtered = append(filtered, queued)
	}
	if first < 0 {
		return false
	}
	// persist request 不携带 snapshot；真正保存时读取当时 root，因此保留最后一个请求即可。
	runtime.setQueueFilteredLocked(original, filtered)
	runtime.queue = append(runtime.queue, nil)
	copy(runtime.queue[first+1:], runtime.queue[first:])
	runtime.queue[first] = msg
	return true
}

type queuedLiveUpdate struct {
	ref  state.TerminalRef
	kind queuedLiveUpdateKind
}

type queuedLiveUpdateKind uint8

const (
	queuedLiveUpdateRefresh queuedLiveUpdateKind = iota + 1
	queuedLiveUpdateSurface
)

func queuedOrdinaryLiveUpdate(msg Msg) (queuedLiveUpdate, bool) {
	switch msg := msg.(type) {
	case LiveEventMsg:
		// 中文说明：runtime 队列只合并 daemon 事件总线的普通 refresh hint；
		if ordinaryLiveRefreshEvent(msg.Event) {
			if msg.Event.TerminalID == "" {
				return queuedLiveUpdate{}, false
			}
			return queuedLiveUpdate{ref: state.NewTerminalRef(msg.Event.EndpointID, msg.Event.TerminalID), kind: queuedLiveUpdateRefresh}, true
		}
	case LiveSurfaceMsg:
		// 中文说明：普通 native screen result 是 live projection，不是语义帧；
		// 压力输出下同 terminal 已返回的旧 projection 必须 latest-only 合并，
		// 生命周期/错误 surface 仍是不可丢的语义边界。
		if ordinaryLiveSurfaceResult(msg) {
			return queuedLiveUpdate{ref: msg.Snapshot.TerminalRef(), kind: queuedLiveUpdateSurface}, true
		}
	default:
		return queuedLiveUpdate{}, false
	}
	return queuedLiveUpdate{}, false
}

func liveQueueBoundary(msg Msg) bool {
	switch msg := msg.(type) {
	case LiveAttachMsg, LiveAttachResultMsg, LiveInputAttachResultMsg, TerminalPoolAttachResultMsg, LiveExitMsg, LiveResizeMsg, LiveResizeResultMsg, HostResizeMsg, QuitMsg:
		return true
	case LiveSurfaceMsg:
		_, ok := queuedOrdinaryLiveUpdate(msg)
		return !ok
	case LiveEventMsg:
		_, ok := queuedOrdinaryLiveUpdate(msg)
		return !ok
	default:
		return false
	}
}

func firstOrdinaryLiveUpdateIndex(queue []Msg) int {
	for i, queued := range queue {
		if _, ok := queuedOrdinaryLiveUpdate(queued); ok {
			return i
		}
		if liveQueueBoundary(queued) {
			return -1
		}
	}
	return -1
}

func ordinaryLiveSurfaceResult(msg LiveSurfaceMsg) bool {
	return msg.Err == nil && !msg.LifecycleKnown && msg.Snapshot.TerminalID != ""
}

func messageApproxBytes(msg Msg) int {
	switch msg := msg.(type) {
	case LiveSurfaceMsg:
		return liveSnapshotApproxBytes(msg.Snapshot)
	case LiveEventMsg:
		return liveEventApproxBytes(msg.Event)
	default:
		return 0
	}
}

func (runtime *AppRuntime) removeAtLocked(index int) {
	if index < 0 || index >= len(runtime.queue) {
		return
	}
	copy(runtime.queue[index:], runtime.queue[index+1:])
	last := len(runtime.queue) - 1
	runtime.queue[last] = nil
	if last == 0 {
		runtime.queue = nil
		return
	}
	runtime.queue = runtime.queue[:last]
}

func (runtime *AppRuntime) setQueueFilteredLocked(original []Msg, filtered []Msg) {
	for i := len(filtered); i < len(original); i++ {
		original[i] = nil
	}
	if len(filtered) == 0 {
		runtime.queue = nil
		return
	}
	runtime.queue = filtered
}

func (runtime *AppRuntime) prepend(msg Msg) {
	if msg == nil {
		return
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if _, ok := msg.(HostResizeMsg); ok {
		runtime.dropQueuedHostResizeLocked()
	}
	runtime.queue = append([]Msg{msg}, runtime.queue...)
	runtime.signalWakeLocked()
}

func (runtime *AppRuntime) dequeue() (Msg, bool) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if len(runtime.queue) == 0 {
		return nil, false
	}
	msg := runtime.queue[0]
	copy(runtime.queue, runtime.queue[1:])
	last := len(runtime.queue) - 1
	runtime.queue[last] = nil
	if last == 0 {
		runtime.queue = nil
		return msg, true
	}
	runtime.queue = runtime.queue[:last]
	return msg, true
}

func (runtime *AppRuntime) signalWakeLocked() {
	if runtime.wake == nil {
		return
	}
	select {
	case runtime.wake <- struct{}{}:
	default:
	}
}
