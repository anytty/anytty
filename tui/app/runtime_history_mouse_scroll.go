package app

import (
	"time"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

const historyMouseScrollInterval = 100 * time.Millisecond

// Gesture identity prevents a queued timer from scrolling after release or
// after a different selection has started. Timers live in the runtime loop.
type CopyModeMouseAutoScrollMsg struct {
	GestureID uint64
	ViewID    string
	Direction int
}

func (CopyModeMouseAutoScrollMsg) isMsg() {}

func (runtime *AppRuntime) updateHistoryMouseScrollEdge(event input.InputEvent) {
	direction := 0
	root := rootWithCopyHistorySessionForView(runtime.state, runtime.mouseDrag.ViewID)
	if rect, ok := copyModeContentRect(root); ok && rect.W > 0 && rect.H > 0 {
		x, y := event.Col-1, event.Row-1
		if x >= rect.X && x < rect.X+rect.W {
			if y <= rect.Y {
				direction = -1
			} else if y >= rect.Y+rect.H-1 {
				direction = 1
			}
		}
	}
	if direction != runtime.mouseDrag.HistoryScrollDirection {
		runtime.mouseDrag.HistoryScrollDirection = direction
		runtime.mouseDrag.NextHistoryScroll = runtime.currentTime().Add(historyMouseScrollInterval)
	}
	if direction == 0 {
		runtime.mouseDrag.NextHistoryScroll = time.Time{}
	}
}

func (runtime *AppRuntime) historyMouseScrollAvailable() bool {
	drag := runtime.mouseDrag
	if !drag.Active || drag.Kind != mouseDragHistorySelect || drag.HistoryScrollDirection == 0 || runtime.state.Shell.Overlay.Open {
		return false
	}
	history, copyMode := runtime.state.CopyHistorySessionForView(drag.ViewID)
	if !copyMode.CanSelect() || copyMode.Mark == nil || copyMode.BoundToken != drag.HistoryToken || history.Pending != nil {
		return false
	}
	if drag.HistoryScrollDirection < 0 {
		return copyMode.Cursor.Row > 0 || copyMode.CanPageHistory() && history.OlderRequestState() == state.OlderRequestReady
	}
	return !copyMode.AtFrozenBottom(history)
}

func (runtime *AppRuntime) nextHistoryMouseScrollWakeDelay() time.Duration {
	if !runtime.historyMouseScrollAvailable() || runtime.mouseDrag.NextHistoryScroll.IsZero() {
		return -1
	}
	delay := runtime.mouseDrag.NextHistoryScroll.Sub(runtime.currentTime())
	if delay < 0 {
		return 0
	}
	return delay
}

func (runtime *AppRuntime) enqueueDueHistoryMouseScroll() {
	if runtime.nextHistoryMouseScrollWakeDelay() != 0 {
		return
	}
	drag := runtime.mouseDrag
	// Never accumulate missed ticks while the backend is slow or the UI is busy.
	runtime.mouseDrag.NextHistoryScroll = runtime.currentTime().Add(historyMouseScrollInterval)
	runtime.enqueue(CopyModeMouseAutoScrollMsg{GestureID: drag.HistoryGestureID, ViewID: drag.ViewID, Direction: drag.HistoryScrollDirection})
}
