package state

import "strings"

// TerminalPickerGroups 返回 terminal picker 的 endpoint 分组投影。
// 分组元数据只来自 reducer-owned EndpointStore；terminal 行仍由 TerminalPickerItems 过滤和选择，避免 renderer 自行读取配置。
func TerminalPickerGroups(root Root) []EndpointPickerGroup {
	if !root.Endpoints.HasItems() {
		return nil
	}
	counts := terminalCountsByEndpoint(root.TerminalPool.Items)
	groups := makeEndpointPickerGroups(root.Endpoints.Normalize(), counts)
	indexByEndpoint := map[EndpointID]int{}
	for index, group := range groups {
		indexByEndpoint[group.EndpointID] = index
	}
	query := strings.ToLower(strings.TrimSpace(root.Shell.ReadonlyDefaults().Overlay.Query))
	for _, poolItem := range root.TerminalPool.Items {
		poolItem = normalizeTerminalPoolItem(poolItem)
		if poolItem.TerminalID == "" {
			continue
		}
		active := terminalPickerPoolItemActive(root, poolItem)
		item := terminalPickerItemWithEndpoint(root, TerminalPickerItem{
			EndpointID: poolItem.EndpointID, Title: terminalPoolTitle(poolItem), Kind: PaneTerminalLive,
			TerminalID: poolItem.TerminalID, Active: active, FromPool: true,
			PoolState: terminalPickerPoolState(poolItem, active), Tags: cloneStringMap(poolItem.Tags),
			Cols: poolItem.Cols, Rows: poolItem.Rows, LastOutputAt: poolItem.LastOutputAt,
		})
		if !matchesTerminalPickerQuery(item, query) {
			continue
		}
		endpointID := NormalizeEndpointID(item.EndpointID)
		index, ok := indexByEndpoint[endpointID]
		if !ok {
			groups = append(groups, endpointPickerGroupFromEndpoint(UnregisteredEndpoint(endpointID), counts[endpointID], false))
			index = len(groups) - 1
			indexByEndpoint[endpointID] = index
		}
		groups[index].VisibleTerminalRows = append(groups[index].VisibleTerminalRows, item)
	}
	return filterEndpointPickerGroups(groups, query)
}

// TerminalPoolPageGroups 返回 Terminal Manager 的 endpoint 分组投影。
// action/selection 继续使用 TerminalPoolPageItems 的 flat row index；该结构只负责展示 endpoint header 和局部失败状态。
func TerminalPoolPageGroups(root Root) []TerminalPoolPageGroup {
	if !root.Endpoints.HasItems() {
		return nil
	}
	rows := TerminalPoolPageItems(root)
	groups := endpointPoolGroupsForRows(root, rows)
	query := strings.ToLower(strings.TrimSpace(root.Shell.ReadonlyDefaults().Overlay.Query))
	return filterEndpointPoolGroups(groups, query)
}

func terminalPickerItemWithEndpoint(root Root, item TerminalPickerItem) TerminalPickerItem {
	item.EndpointID = NormalizeEndpointID(item.EndpointID)
	if endpoint, ok := root.Endpoints.DisplayEndpoint(item.EndpointID); ok {
		item.EndpointLabel = endpoint.DisplayLabel()
		item.EndpointTransport = endpoint.Transport
		item.EndpointConnectMode = endpoint.ConnectMode
		item.EndpointStatus = endpoint.DisplayStatus()
		item.EndpointLastError = endpoint.LastError
		item.EndpointErrorKind = endpoint.LastErrorKind
	}
	return item
}

func terminalPoolPageItemWithEndpoint(root Root, item TerminalPoolPageItem) TerminalPoolPageItem {
	item.EndpointID = NormalizeEndpointID(item.EndpointID)
	if endpoint, ok := root.Endpoints.DisplayEndpoint(item.EndpointID); ok {
		item.EndpointLabel = endpoint.DisplayLabel()
		item.EndpointTransport = endpoint.Transport
		item.EndpointConnectMode = endpoint.ConnectMode
		item.EndpointStatus = endpoint.DisplayStatus()
		item.EndpointLastError = endpoint.LastError
		item.EndpointErrorKind = endpoint.LastErrorKind
	}
	return item
}

func endpointPickerGroupsForRows(root Root, rows []TerminalPickerItem) []EndpointPickerGroup {
	counts := terminalCountsByEndpoint(root.TerminalPool.Items)
	groups := makeEndpointPickerGroups(root.Endpoints.Normalize(), counts)
	indexByEndpoint := map[EndpointID]int{}
	for index, group := range groups {
		indexByEndpoint[group.EndpointID] = index
	}
	for _, row := range rows {
		if row.CreateNew {
			continue
		}
		endpointID := NormalizeEndpointID(row.EndpointID)
		index, ok := indexByEndpoint[endpointID]
		if !ok {
			endpoint := UnregisteredEndpoint(endpointID)
			group := endpointPickerGroupFromEndpoint(endpoint, counts[endpointID], false)
			groups = append(groups, group)
			index = len(groups) - 1
			indexByEndpoint[endpointID] = index
		}
		groups[index].VisibleTerminalRows = append(groups[index].VisibleTerminalRows, row)
	}
	return groups
}

func endpointPoolGroupsForRows(root Root, rows []TerminalPoolPageItem) []TerminalPoolPageGroup {
	counts := terminalCountsByEndpoint(root.TerminalPool.Items)
	groups := makeEndpointPoolGroups(root.Endpoints.Normalize(), counts)
	indexByEndpoint := map[EndpointID]int{}
	for index, group := range groups {
		indexByEndpoint[group.EndpointID] = index
	}
	for _, row := range rows {
		endpointID := NormalizeEndpointID(row.EndpointID)
		index, ok := indexByEndpoint[endpointID]
		if !ok {
			endpoint := UnregisteredEndpoint(endpointID)
			group := endpointPoolGroupFromEndpoint(endpoint, counts[endpointID], false)
			groups = append(groups, group)
			index = len(groups) - 1
			indexByEndpoint[endpointID] = index
		}
		groups[index].VisibleTerminalRows = append(groups[index].VisibleTerminalRows, row)
	}
	return groups
}

func makeEndpointPickerGroups(endpoints EndpointStore, counts map[EndpointID]int) []EndpointPickerGroup {
	groups := make([]EndpointPickerGroup, 0, len(endpoints.Items))
	for _, endpoint := range endpoints.Items {
		endpoint = endpoint.withDefaults()
		groups = append(groups, endpointPickerGroupFromEndpoint(endpoint, counts[endpoint.ID], true))
	}
	return groups
}

func makeEndpointPoolGroups(endpoints EndpointStore, counts map[EndpointID]int) []TerminalPoolPageGroup {
	groups := make([]TerminalPoolPageGroup, 0, len(endpoints.Items))
	for _, endpoint := range endpoints.Items {
		endpoint = endpoint.withDefaults()
		groups = append(groups, endpointPoolGroupFromEndpoint(endpoint, counts[endpoint.ID], true))
	}
	return groups
}

func endpointPickerGroupFromEndpoint(endpoint EndpointItem, terminalCount int, configured bool) EndpointPickerGroup {
	endpoint = endpoint.withDefaults()
	return EndpointPickerGroup{
		EndpointID:           endpoint.ID,
		Label:                endpoint.DisplayLabel(),
		Transport:            endpoint.Transport,
		ObservedPath:         endpoint.ObservedPath,
		RouteSelectionReason: endpoint.RouteSelectionReason,
		ConnectionPhase:      endpoint.ConnectionPhase,
		ConnectMode:          endpoint.ConnectMode,
		Status:               endpoint.DisplayStatus(),
		LastError:            endpoint.LastError,
		ErrorKind:            endpoint.LastErrorKind,
		Configured:           configured && !endpoint.Unregistered,
		TerminalCount:        terminalCount,
	}
}

func endpointPoolGroupFromEndpoint(endpoint EndpointItem, terminalCount int, configured bool) TerminalPoolPageGroup {
	endpoint = endpoint.withDefaults()
	return TerminalPoolPageGroup{
		EndpointID:           endpoint.ID,
		Label:                endpoint.DisplayLabel(),
		Transport:            endpoint.Transport,
		ObservedPath:         endpoint.ObservedPath,
		RouteSelectionReason: endpoint.RouteSelectionReason,
		ConnectionPhase:      endpoint.ConnectionPhase,
		ConnectMode:          endpoint.ConnectMode,
		Status:               endpoint.DisplayStatus(),
		LastError:            endpoint.LastError,
		ErrorKind:            endpoint.LastErrorKind,
		Configured:           configured && !endpoint.Unregistered,
		TerminalCount:        terminalCount,
	}
}

func filterEndpointPickerGroups(groups []EndpointPickerGroup, query string) []EndpointPickerGroup {
	if query == "" {
		return groups
	}
	filtered := make([]EndpointPickerGroup, 0, len(groups))
	for _, group := range groups {
		if len(group.VisibleTerminalRows) > 0 || endpointGroupMatchesQuery(group.EndpointID, group.Label, group.Transport, group.ConnectMode, group.Status, query) {
			filtered = append(filtered, group)
		}
	}
	return filtered
}

func filterEndpointPoolGroups(groups []TerminalPoolPageGroup, query string) []TerminalPoolPageGroup {
	if query == "" {
		return groups
	}
	filtered := make([]TerminalPoolPageGroup, 0, len(groups))
	for _, group := range groups {
		if len(group.VisibleTerminalRows) > 0 || endpointGroupMatchesQuery(group.EndpointID, group.Label, group.Transport, group.ConnectMode, group.Status, query) {
			filtered = append(filtered, group)
		}
	}
	return filtered
}

func endpointGroupMatchesQuery(endpointID EndpointID, label string, transport EndpointTransportKind, connectMode EndpointConnectMode, status EndpointStatusKind, query string) bool {
	return strings.Contains(strings.ToLower(string(endpointID)), query) ||
		strings.Contains(strings.ToLower(label), query) ||
		strings.Contains(strings.ToLower(string(transport)), query) ||
		strings.Contains(strings.ToLower(string(connectMode)), query) ||
		strings.Contains(strings.ToLower(string(status)), query)
}

func terminalCountsByEndpoint(items []TerminalPoolItem) map[EndpointID]int {
	counts := map[EndpointID]int{}
	for _, item := range items {
		item = normalizeTerminalPoolItem(item)
		if item.TerminalID == "" {
			continue
		}
		counts[item.EndpointID]++
	}
	return counts
}
