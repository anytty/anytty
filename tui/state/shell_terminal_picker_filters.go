package state

import (
	"sort"
	"strconv"
	"strings"
)

func TerminalPickerEndpointTabs(root Root) []TerminalPickerEndpointTab {
	counts := terminalCountsByEndpoint(root.TerminalPool.Items)
	groups := makeEndpointPickerGroups(root.Endpoints.Normalize(), counts)
	known := make(map[EndpointID]struct{}, len(groups))
	for _, group := range groups {
		known[group.EndpointID] = struct{}{}
	}
	if len(groups) == 0 {
		endpoint := DefaultLocalEndpoint()
		groups = append(groups, endpointPickerGroupFromEndpoint(endpoint, counts[endpoint.ID], true))
		known[endpoint.ID] = struct{}{}
	}
	orphanIDs := make([]string, 0)
	for endpointID := range counts {
		if _, ok := known[endpointID]; ok {
			continue
		}
		orphanIDs = append(orphanIDs, string(endpointID))
	}
	sort.Strings(orphanIDs)
	for _, rawEndpointID := range orphanIDs {
		endpointID := EndpointID(rawEndpointID)
		count := counts[endpointID]
		groups = append(groups, endpointPickerGroupFromEndpoint(UnregisteredEndpoint(endpointID), count, false))
	}
	selectedID := terminalPickerActiveEndpointFromGroups(root, groups)
	tabs := make([]TerminalPickerEndpointTab, 0, len(groups))
	for _, group := range groups {
		label := strings.TrimSpace(group.Label)
		if label == "" {
			label = string(group.EndpointID)
		}
		tabs = append(tabs, TerminalPickerEndpointTab{
			EndpointID:  group.EndpointID,
			Label:       label,
			Count:       group.TerminalCount,
			Selected:    group.EndpointID == selectedID,
			Transport:   group.Transport,
			ConnectMode: group.ConnectMode,
			Status:      group.Status,
			LastError:   group.LastError,
			ErrorKind:   group.ErrorKind,
		})
	}
	return tabs
}

// TerminalPickerVisibleEndpointOptions returns endpoint choices filtered only by
// the endpoint subview query. Terminal search remains in Overlay.Query.
func TerminalPickerVisibleEndpointOptions(root Root) []TerminalPickerEndpointTab {
	tabs := TerminalPickerEndpointTabs(root)
	query := strings.TrimSpace(root.Shell.ReadonlyDefaults().Overlay.TerminalPickerEndpointQuery)
	if query == "" {
		return tabs
	}
	filtered := make([]TerminalPickerEndpointTab, 0, len(tabs))
	for _, tab := range tabs {
		values := []string{
			tab.Label,
			string(tab.EndpointID),
			string(tab.Transport),
			string(tab.ConnectMode),
			string(tab.Status),
			strings.ReplaceAll(string(tab.Status), "_", "-"),
			string(tab.ErrorKind),
			tab.LastError,
		}
		for _, value := range values {
			if TerminalPickerQueryMatchIndexes(value, query) != nil {
				filtered = append(filtered, tab)
				break
			}
		}
	}
	return filtered
}

func TerminalPickerActiveEndpointID(root Root) EndpointID {
	for _, tab := range TerminalPickerEndpointTabs(root) {
		if tab.Selected {
			return tab.EndpointID
		}
	}
	return DefaultEndpointID
}

func TerminalPickerStatusOptions(root Root) []TerminalPickerStatusOption {
	endpointID := TerminalPickerActiveEndpointID(root)
	counts := map[TerminalPickerStatusFilter]int{
		TerminalPickerStatusAll:     0,
		TerminalPickerStatusRunning: 0,
		TerminalPickerStatusExited:  0,
	}
	for _, poolItem := range root.TerminalPool.Items {
		poolItem = normalizeTerminalPoolItem(poolItem)
		if poolItem.TerminalID == "" || poolItem.EndpointID != endpointID {
			continue
		}
		counts[TerminalPickerStatusAll]++
		stateText := strings.ToLower(strings.TrimSpace(poolItem.State))
		if stateText == string(TerminalPickerStatusExited) {
			counts[TerminalPickerStatusExited]++
		} else if stateText == string(TerminalPickerStatusRunning) || stateText == string(TerminalLiveAttached) {
			counts[TerminalPickerStatusRunning]++
		}
	}
	selected := normalizeTerminalPickerStatus(root.Shell.ReadonlyDefaults().Overlay.TerminalPickerStatus)
	return []TerminalPickerStatusOption{
		{Status: TerminalPickerStatusRunning, Label: "Running", Count: counts[TerminalPickerStatusRunning], Selected: selected == TerminalPickerStatusRunning},
		{Status: TerminalPickerStatusExited, Label: "Exited", Count: counts[TerminalPickerStatusExited], Selected: selected == TerminalPickerStatusExited},
		{Status: TerminalPickerStatusAll, Label: "All", Count: counts[TerminalPickerStatusAll], Selected: selected == TerminalPickerStatusAll},
	}
}

func TerminalPickerTagOptions(root Root) []TerminalPickerTagOption {
	endpointID := TerminalPickerActiveEndpointID(root)
	counts := map[string]int{}
	for _, poolItem := range root.TerminalPool.Items {
		poolItem = normalizeTerminalPoolItem(poolItem)
		if poolItem.TerminalID == "" || poolItem.EndpointID != endpointID {
			continue
		}
		for _, label := range PublicTerminalTagLabels(poolItem.Tags) {
			counts[label]++
		}
	}
	selected := stringSet(root.Shell.ReadonlyDefaults().Overlay.TerminalPickerTagFilters)
	for label := range selected {
		if _, ok := counts[label]; !ok {
			counts[label] = 0
		}
	}
	labels := make([]string, 0, len(counts))
	for label := range counts {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	options := make([]TerminalPickerTagOption, 0, len(labels))
	for _, label := range labels {
		_, checked := selected[label]
		options = append(options, TerminalPickerTagOption{Label: label, Count: counts[label], Selected: checked})
	}
	return options
}

func TerminalPickerVisibleTagOptions(root Root) []TerminalPickerTagOption {
	options := TerminalPickerTagOptions(root)
	query := strings.ToLower(strings.TrimSpace(root.Shell.ReadonlyDefaults().Overlay.TerminalPickerTagQuery))
	if query == "" {
		return options
	}
	filtered := make([]TerminalPickerTagOption, 0, len(options))
	for _, option := range options {
		if strings.Contains(strings.ToLower(option.Label), query) {
			filtered = append(filtered, option)
		}
	}
	return filtered
}

func PublicTerminalTagLabels(tags map[string]string) []string {
	if len(tags) == 0 {
		return nil
	}
	labels := map[string]struct{}{}
	for rawKey, rawValue := range tags {
		key := strings.TrimSpace(rawKey)
		value := strings.TrimSpace(rawValue)
		if key == "" || isInternalTerminalTagKey(key) {
			continue
		}
		label := key
		if isPositionalTerminalTagKey(key) && value != "" {
			label = value
		} else if value != "" {
			label = key + "=" + value
		}
		if label != "" {
			labels[label] = struct{}{}
		}
	}
	out := make([]string, 0, len(labels))
	for label := range labels {
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

func ReplacePublicTerminalTags(tags map[string]string, labels []string) map[string]string {
	next := map[string]string{}
	for key, value := range tags {
		if isInternalTerminalTagKey(strings.TrimSpace(key)) {
			next[key] = value
		}
	}
	seen := map[string]struct{}{}
	position := 1
	for _, rawLabel := range labels {
		label := strings.TrimSpace(rawLabel)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		next["tag"+strconv.Itoa(position)] = label
		position++
	}
	if len(next) == 0 {
		return nil
	}
	return next
}

func terminalPickerActiveEndpointFromGroups(root Root, groups []EndpointPickerGroup) EndpointID {
	available := make(map[EndpointID]struct{}, len(groups))
	for _, group := range groups {
		available[group.EndpointID] = struct{}{}
	}
	preferred := []EndpointID{
		root.Shell.ReadonlyDefaults().Overlay.TerminalPickerEndpointID,
		root.Shell.ReadonlyDefaults().TerminalCreateDraft.EndpointID,
	}
	if root.Session.Attached {
		preferred = append(preferred, root.Session.TerminalRef().EndpointID)
	}
	for _, endpointID := range preferred {
		if strings.TrimSpace(string(endpointID)) == "" {
			continue
		}
		endpointID = NormalizeEndpointID(endpointID)
		if _, ok := available[endpointID]; ok {
			return endpointID
		}
	}
	for _, group := range groups {
		return group.EndpointID
	}
	return DefaultEndpointID
}

func terminalPickerMatchesStatus(item TerminalPickerItem, status TerminalPickerStatusFilter) bool {
	status = normalizeTerminalPickerStatus(status)
	if status == TerminalPickerStatusAll {
		return true
	}
	stateText := strings.ToLower(strings.TrimSpace(item.PoolState))
	if status == TerminalPickerStatusExited {
		return stateText == string(TerminalPickerStatusExited)
	}
	return stateText == string(TerminalPickerStatusRunning) || stateText == string(TerminalLiveAttached)
}

func terminalPickerMatchesTags(item TerminalPickerItem, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	labels := stringSet(PublicTerminalTagLabels(item.Tags))
	for _, filter := range filters {
		if _, ok := labels[filter]; !ok {
			return false
		}
	}
	return true
}

func isInternalTerminalTagKey(key string) bool {
	return key == "cwd" || strings.HasPrefix(key, "anytty.")
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
