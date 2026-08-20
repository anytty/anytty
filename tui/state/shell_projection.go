package state

import (
	"sort"
	"strconv"
	"strings"
)

// TerminalPickerItems 从 reducer-owned root 推导 picker 列表；服务端 Terminal Pool 必须先回投到 TerminalPoolStore。
func TerminalPickerItems(root Root) []TerminalPickerItem {
	shell := root.Shell.ReadonlyDefaults()
	query := strings.ToLower(strings.TrimSpace(shell.Overlay.Query))
	items := []TerminalPickerItem{}
	items = append(items, terminalPickerCreateItems(root, query)...)
	for _, poolItem := range root.TerminalPool.Items {
		poolItem = normalizeTerminalPoolItem(poolItem)
		if poolItem.TerminalID == "" {
			continue
		}
		active := terminalPickerPoolItemActive(root, poolItem)
		item := TerminalPickerItem{
			EndpointID:   poolItem.EndpointID,
			Title:        terminalPoolTitle(poolItem),
			Kind:         PaneTerminalLive,
			TerminalID:   poolItem.TerminalID,
			Active:       active,
			FromPool:     true,
			PoolState:    terminalPickerPoolState(poolItem, active),
			Cols:         poolItem.Cols,
			Rows:         poolItem.Rows,
			LastOutputAt: poolItem.LastOutputAt,
		}
		item = terminalPickerItemWithEndpoint(root, item)
		if !matchesTerminalPickerQuery(item, query) {
			continue
		}
		items = append(items, item)
	}
	sortTerminalPickerItemsByName(items)
	if len(items) > 0 {
		selected := shell.Overlay.SelectedIndex
		if selected < 0 {
			selected = 0
		}
		if selected >= len(items) {
			selected = len(items) - 1
		}
		items[selected].Selected = true
	}
	return items
}

func terminalPickerPoolItemActive(root Root, poolItem TerminalPoolItem) bool {
	ref := poolItem.TerminalRef()
	if root.Session.Attached && root.Session.TerminalRef().Equal(ref) {
		return true
	}
	for _, binding := range root.TerminalViews.BindingsForTerminalRef(ref) {
		if binding.Attached {
			return true
		}
	}
	return false
}

func terminalPickerPoolState(poolItem TerminalPoolItem, active bool) string {
	if active {
		return string(TerminalLiveAttached)
	}
	stateText := strings.TrimSpace(poolItem.State)
	// 中文说明：pool/list 里的 attached 不是当前 TUI view binding 的真值；
	// picker 只在本地 Session/TerminalView 精确命中 TerminalRef 时展示 attached。
	if strings.EqualFold(stateText, string(TerminalLiveAttached)) {
		return "running"
	}
	return stateText
}

func TerminalPoolPageItems(root Root) []TerminalPoolPageItem {
	shell := root.Shell.ReadonlyDefaults()
	query := strings.ToLower(strings.TrimSpace(shell.Overlay.Query))
	items := make([]TerminalPoolPageItem, 0, len(root.TerminalPool.Items))
	for _, poolItem := range root.TerminalPool.Items {
		poolItem = normalizeTerminalPoolItem(poolItem)
		if poolItem.TerminalID == "" {
			continue
		}
		attachmentCount := terminalPoolAttachmentCount(root, poolItem)
		item := TerminalPoolPageItem{
			EndpointID:      poolItem.EndpointID,
			TerminalID:      poolItem.TerminalID,
			Title:           terminalPoolTitle(poolItem),
			State:           poolItem.State,
			CWD:             poolItem.CWD,
			Command:         append([]string(nil), poolItem.Command...),
			Tags:            cloneStringMap(poolItem.Tags),
			ExitCode:        cloneIntPointer(poolItem.ExitCode),
			ExitedAt:        poolItem.ExitedAt,
			LastOutputAt:    poolItem.LastOutputAt,
			Cols:            poolItem.Cols,
			Rows:            poolItem.Rows,
			AttachmentCount: attachmentCount,
			Resources:       poolItem.Resources,
			ResourceHistory: root.TerminalPool.ResourceSamples(poolItem.TerminalRef()),
			Attached:        poolItem.Attached || attachmentCount > 0,
		}
		item = terminalPoolPageItemWithEndpoint(root, item)
		if !matchesTerminalPoolPageQuery(item, query) {
			continue
		}
		items = append(items, item)
	}
	sortTerminalPoolPageItemsByName(items)
	if len(items) > 0 {
		selected := shell.Overlay.SelectedIndex
		if selected < 0 {
			selected = 0
		}
		if selected >= len(items) {
			selected = len(items) - 1
		}
		items[selected].Selected = true
	}
	return items
}

func sortTerminalPickerItemsByName(items []TerminalPickerItem) {
	// 中文说明：排序只改变 picker 的展示投影；TerminalRef 仍是 action 路由真值，
	// 不把 endpoint label 或 terminal title 当成跨 daemon 身份。
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.CreateNew || right.CreateNew {
			return left.CreateNew && !right.CreateNew
		}
		return terminalPickerSortKey(left) < terminalPickerSortKey(right)
	})
}

func terminalPickerSortKey(item TerminalPickerItem) string {
	return strings.ToLower(strings.Join([]string{
		item.Title,
		item.EndpointLabel,
		string(item.EndpointID),
		item.TerminalID,
	}, "\x00"))
}

func sortTerminalPoolPageItemsByName(items []TerminalPoolPageItem) {
	// 中文说明：Terminal Manager 与 picker 使用同一展示排序语义，
	// 但 manager action 仍通过 row 中的 EndpointID + TerminalID 回到 owning daemon。
	sort.SliceStable(items, func(i, j int) bool {
		return terminalPoolPageSortKey(items[i]) < terminalPoolPageSortKey(items[j])
	})
}

func terminalPoolPageSortKey(item TerminalPoolPageItem) string {
	return strings.ToLower(strings.Join([]string{
		item.Title,
		item.EndpointLabel,
		string(item.EndpointID),
		item.TerminalID,
	}, "\x00"))
}

func terminalPoolAttachmentCount(root Root, poolItem TerminalPoolItem) int {
	count := poolItem.AttachmentCount
	localAttached := 0
	for _, binding := range root.TerminalViews.BindingsForTerminalRef(poolItem.TerminalRef()) {
		if binding.Attached && binding.Channel != 0 {
			localAttached++
		}
	}
	if localAttached > count {
		count = localAttached
	}
	return count
}

func matchesTerminalPickerQuery(item TerminalPickerItem, query string) bool {
	if query == "" {
		return true
	}
	if item.CreateNew {
		return TerminalPickerQueryMatchIndexes(item.Title, query) != nil ||
			TerminalPickerQueryMatchIndexes("create terminal", query) != nil ||
			TerminalPickerQueryMatchIndexes("new terminal", query) != nil ||
			TerminalPickerQueryMatchIndexes(string(item.EndpointID), query) != nil ||
			TerminalPickerQueryMatchIndexes(item.EndpointLabel, query) != nil ||
			TerminalPickerQueryMatchIndexes(item.EndpointSearchText, query) != nil
	}
	return TerminalPickerQueryMatchIndexes(item.Title, query) != nil ||
		TerminalPickerQueryMatchIndexes(item.TerminalID, query) != nil ||
		TerminalPickerQueryMatchIndexes(item.PoolState, query) != nil ||
		TerminalPickerQueryMatchIndexes(string(item.EndpointID), query) != nil ||
		TerminalPickerQueryMatchIndexes(item.EndpointLabel, query) != nil ||
		TerminalPickerQueryMatchIndexes(terminalPickerSizeText(item), query) != nil
}

func terminalPickerSizeText(item TerminalPickerItem) string {
	if item.Cols <= 0 || item.Rows <= 0 {
		return ""
	}
	return strconv.Itoa(item.Cols) + "x" + strconv.Itoa(item.Rows)
}

func TerminalPickerQueryMatchIndexes(value string, query string) []int {
	query = strings.TrimSpace(query)
	if query == "" {
		return []int{}
	}
	valueRunes := []rune(value)
	queryRunes := []rune(query)
	matches := make([]int, 0, len(queryRunes))
	valueAt := 0
	for _, queryRune := range queryRunes {
		queryLower := []rune(strings.ToLower(string(queryRune)))
		if len(queryLower) == 0 {
			continue
		}
		found := false
		for valueAt < len(valueRunes) {
			valueLower := []rune(strings.ToLower(string(valueRunes[valueAt])))
			if len(valueLower) > 0 && valueLower[0] == queryLower[0] {
				matches = append(matches, valueAt)
				valueAt++
				found = true
				break
			}
			valueAt++
		}
		if !found {
			return nil
		}
	}
	return matches
}

func matchesTerminalPoolPageQuery(item TerminalPoolPageItem, query string) bool {
	if query == "" {
		return true
	}
	if strings.Contains(strings.ToLower(item.Title), query) ||
		strings.Contains(strings.ToLower(item.TerminalID), query) ||
		strings.Contains(strings.ToLower(item.State), query) ||
		strings.Contains(strings.ToLower(item.CWD), query) ||
		strings.Contains(strings.ToLower(string(item.EndpointID)), query) ||
		strings.Contains(strings.ToLower(item.EndpointLabel), query) ||
		strings.Contains(strings.ToLower(string(item.EndpointTransport)), query) ||
		strings.Contains(strings.ToLower(string(item.EndpointConnectMode)), query) ||
		strings.Contains(strings.ToLower(string(item.EndpointStatus)), query) ||
		strings.Contains(strings.ToLower(strings.Join(item.Command, " ")), query) {
		return true
	}
	for key, value := range item.Tags {
		if strings.Contains(strings.ToLower(key), query) || strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}
