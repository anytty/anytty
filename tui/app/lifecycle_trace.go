package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/anytty/anytty/tui/state"
)

func lifecycleTerminalViewsSummary(store state.TerminalViewStore) string {
	return lifecycleTerminalViewBindingsSummary(store.Bindings())
}

func lifecycleTerminalViewBindingsSummary(bindings []state.TerminalViewBinding) string {
	if len(bindings) == 0 {
		return ""
	}
	ordered := append([]state.TerminalViewBinding(nil), bindings...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].TerminalID != ordered[j].TerminalID {
			return ordered[i].TerminalID < ordered[j].TerminalID
		}
		return ordered[i].ViewID < ordered[j].ViewID
	})
	parts := make([]string, 0, len(ordered))
	for _, binding := range ordered {
		parts = append(parts, fmt.Sprintf(
			"%s term=%s pane=%s floating=%s ch=%d attached=%t role=%s size=%dx%d surface=%s can_resize=%t",
			binding.ViewID,
			binding.TerminalID,
			binding.PaneID,
			binding.FloatingID,
			binding.Channel,
			binding.Attached,
			binding.ResizeRole,
			binding.DesiredCols,
			binding.DesiredRows,
			binding.SurfaceID,
			binding.CanResize,
		))
	}
	return strings.Join(parts, " | ")
}

func lifecycleInputChannelsSummary(channels map[string]uint16) string {
	if len(channels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(channels))
	for terminalID := range channels {
		keys = append(keys, terminalID)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, terminalID := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", terminalID, channels[terminalID]))
	}
	return strings.Join(parts, ",")
}

func lifecycleTimeSummary(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
