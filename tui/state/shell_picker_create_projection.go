package state

import "strings"

func terminalPickerCreateItems(root Root, query string) []TerminalPickerItem {
	item := TerminalPickerItem{
		EndpointID: terminalPickerCreateDefaultEndpoint(root),
		Title:      "new terminal",
		Kind:       PaneTerminalLive,
		CreateNew:  true,
	}
	item = terminalPickerItemWithEndpoint(root, item)
	item.EndpointSearchText = terminalPickerCreateEndpointSearchText(root, item.EndpointLabel)
	if matchesTerminalPickerQuery(item, query) {
		return []TerminalPickerItem{item}
	}
	return nil
}

func terminalPickerCreateDefaultEndpoint(root Root) EndpointID {
	draftEndpointID := NormalizeEndpointID(root.Shell.ReadonlyDefaults().TerminalCreateDraft.EndpointID)
	if terminalPickerCreateEndpointAvailable(root, draftEndpointID) {
		return draftEndpointID
	}
	for _, endpoint := range TerminalCreateEndpointItems(root) {
		return endpoint.ID
	}
	return DefaultEndpointID
}

func terminalPickerCreateEndpointAvailable(root Root, endpointID EndpointID) bool {
	endpointID = NormalizeEndpointID(endpointID)
	if endpointID == "" {
		return false
	}
	for _, endpoint := range TerminalCreateEndpointItems(root) {
		if endpoint.ID == endpointID {
			return true
		}
	}
	return false
}

func terminalPickerCreateEndpointSearchText(root Root, fallback string) string {
	parts := []string{}
	if strings.TrimSpace(fallback) != "" {
		parts = append(parts, strings.TrimSpace(fallback))
	}
	for _, endpoint := range TerminalCreateEndpointItems(root) {
		label := endpoint.DisplayLabel()
		if strings.TrimSpace(label) != "" {
			parts = append(parts, strings.TrimSpace(label))
		}
		if endpoint.ID != "" {
			parts = append(parts, string(endpoint.ID))
		}
	}
	return strings.Join(parts, " ")
}
