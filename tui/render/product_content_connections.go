package render

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/tui/state"
)

// Connections Page 只渲染 reducer 已持有的 Endpoint registry/planner/runtime 投影。
// 当前 Route、Path 和 generation 缺失时显示空值，不按 priority 或 transport kind 猜测连接事实。
func buildConnectionsContent(root state.Root, shell state.ShellStore) ContentVM {
	shell = shell.ReadonlyDefaults()
	items := root.Endpoints.Normalize().Items
	selectedIndex := shell.Overlay.SelectedIndex
	if selectedIndex < 0 || selectedIndex >= len(items) {
		selectedIndex = 0
	}
	layout := terminalManagerLayoutForViewport(chromeSafeViewportForShell(root.Viewport, shell))
	rightHeader := "CONNECTION DETAILS"
	if layout.DetailWidth < 30 {
		rightHeader = "DETAILS"
	}
	lines := []Line{
		terminalManagerFullLine(NewLine(fmt.Sprintf("%d endpoints", len(items))), layout),
		terminalManagerDividerLine(layout),
		terminalManagerBodyLine(terminalManagerHeaderLine("ENDPOINTS"), terminalManagerHeaderLine(rightHeader), layout),
	}
	var selected state.EndpointItem
	selectedOK := len(items) > 0
	selectedActiveViews := 0
	if selectedOK {
		selected = items[selectedIndex]
		selectedActiveViews = root.TerminalViews.AttachedBindingCountForEndpoint(selected.ID)
	}
	details := connectionsDetailLines(selected, selectedOK, layout.DetailWidth, selectedActiveViews)
	for row := 0; row < layout.BodyRows; row++ {
		left := Line{}
		if row < len(items) {
			activeViews := root.TerminalViews.AttachedBindingCountForEndpoint(items[row].ID)
			left = connectionsEndpointLine(items[row], row == selectedIndex, activeViews, endpointLastOutputActivityLabel(root, items[row].ID))
		}
		right := Line{}
		if row < len(details) {
			right = details[row]
		}
		lines = append(lines, terminalManagerBodyLine(left, right, layout))
	}
	return ContentVM{
		Kind: ContentConnections, Lines: lines, Meta: ContentMetaVM{SplitPageLeftWidth: layout.ListWidth},
		Status: fmt.Sprintf("connections: %d endpoints", len(items)), Empty: len(items) == 0, Cursor: Cursor{Visible: false},
	}
}

func connectionsEndpointLine(item state.EndpointItem, selected bool, activeViews int, activity string) Line {
	marker, markerStyle, labelStyle := "  ", StyleMuted, StyleForeground
	if selected {
		marker, markerStyle, labelStyle = "▸ ", StyleAccent, StyleAccent
	}
	checkbox, checkboxStyle := "[ ] ", StyleMuted
	if item.Enabled {
		checkbox, checkboxStyle = "[x] ", StyleSuccess
	}
	status := string(item.DisplayStatus())
	statusStyle := endpointStatusStyle(item.DisplayStatus())
	if !item.Enabled && activeViews > 0 {
		status = fmt.Sprintf("draining %d view(s)", activeViews)
		statusStyle = StyleWarning
	}
	if status == "" {
		status = "unknown"
	}
	cells := []Cell{
		styledCell(marker, markerStyle), styledCell(checkbox, checkboxStyle), styledCell(item.DisplayLabel(), labelStyle), NewCell(" "), tokenCell(status, statusStyle),
	}
	if activity != "" {
		cells = append(cells, NewCell(" "), styledCell(activity, TerminalOutputActivityStyle(activity)))
	}
	return Line{Cells: cells}
}

// endpointLastOutputActivityLabel 返回该 endpoint 下所有 terminal 中最近一次非空 PTY 输出
// 的活跃度标签；没有输出历史时返回空字符串。
func endpointLastOutputActivityLabel(root state.Root, endpointID state.EndpointID) string {
	endpointID = state.NormalizeEndpointID(endpointID)
	var latest time.Time
	for _, item := range root.TerminalPool.Items {
		if item.EndpointID != endpointID || item.LastOutputAt.IsZero() {
			continue
		}
		if item.LastOutputAt.After(latest) {
			latest = item.LastOutputAt
		}
	}
	return TerminalOutputActivityLabel(latest, time.Now())
}

func connectionsDetailLines(item state.EndpointItem, ok bool, width int, activeViews int) []Line {
	if !ok {
		return []Line{NewLine("No endpoint configured")}
	}
	value := func(label, text string) Line {
		if strings.TrimSpace(text) == "" {
			text = "-"
		}
		return Line{Cells: []Cell{styledCell(label+": ", StyleMuted), NewCell(text)}}
	}
	generation := "-"
	if item.ConnectionGeneration > 0 {
		generation = fmt.Sprintf("%d", item.ConnectionGeneration)
	}
	preference := string(item.RoutePreference)
	if preference == "" {
		preference = "auto"
	}
	routeHeader := "ROUTE PRIORITY / AVAILABILITY"
	if width < 30 {
		routeHeader = "ROUTES"
	}
	snapshot := item.ConnectionSnapshot
	lines := []Line{
		terminalManagerHeaderLine("CURRENT CONNECTION"),
		value("Enabled", map[bool]string{true: "yes", false: "no"}[item.Enabled]),
		value("Status", string(item.DisplayStatus())),
		value("Route", string(item.ActiveRouteID)),
		value("Path", item.ObservedPath),
		value("Generation", generation),
		value("Reason", item.RouteSelectionReason),
	}
	if !item.Enabled && activeViews > 0 {
		lines = append(lines, value("Drain", fmt.Sprintf("%d active view(s)", activeViews)))
	}
	if local := candidateEndpoint(snapshot.LocalAddress, snapshot.LocalPort); local != "" {
		lines = append(lines, value("Local", local))
	}
	if remote := candidateEndpoint(snapshot.RemoteAddress, snapshot.RemotePort); remote != "" {
		lines = append(lines, value("Remote", remote))
	}
	if snapshot.RoundTrip > 0 {
		lines = append(lines, value("RTT", fmt.Sprintf("%.1f ms", float64(snapshot.RoundTrip)/float64(time.Millisecond))))
	}
	if candidates := strings.Trim(strings.Join([]string{snapshot.LocalCandidateType, snapshot.RemoteCandidateType}, " / "), " /"); candidates != "" {
		lines = append(lines, value("Candidates", candidates))
	}
	if protocols := strings.Trim(strings.Join([]string{snapshot.LocalProtocol, snapshot.RemoteProtocol}, " / "), " /"); protocols != "" {
		lines = append(lines, value("ICE", protocols))
	}
	if snapshot.RelayTransport != "" {
		lines = append(lines, value("Relay", snapshot.RelayTransport))
	}
	if snapshot.NetworkClass != "" {
		lines = append(lines, value("Network", snapshot.NetworkClass))
	}
	lines = append(lines, Line{}, terminalManagerHeaderLine("NEXT CONNECTION POLICY"), value("Preference", preference), terminalManagerHeaderLine(routeHeader))
	for _, route := range item.Routes {
		lines = append(lines, connectionsRouteLines(route, width)...)
	}
	return lines
}

func candidateEndpoint(address string, port uint16) string {
	address = strings.TrimSpace(address)
	if address == "" || port == 0 {
		return address
	}
	if strings.Contains(address, ":") {
		return fmt.Sprintf("[%s]:%d", address, port)
	}
	return fmt.Sprintf("%s:%d", address, port)
}

func connectionsRouteLines(route state.EndpointRouteItem, width int) []Line {
	priority := "Full race"
	if route.Priority != nil {
		priority = strconv.Itoa(*route.Priority)
	}
	availability := "unknown"
	style := StyleMuted
	if !route.Enabled {
		availability = "disabled"
	} else if route.ManualOnly {
		availability = "manual"
	} else if route.AvailabilityKnown && route.Available {
		availability, style = "available", StyleSuccess
	} else if route.AvailabilityKnown {
		availability = strings.ReplaceAll(string(route.AvailabilityReason), "_", " ")
		if route.AvailabilityReason == endpointdomain.RouteAvailabilityCredentialUnavailable {
			availability = "credential missing"
		}
		style = StyleWarning
	}
	relay := ""
	if route.Kind == state.EndpointTransportHubP2P {
		relay = strings.Trim(strings.Join([]string{string(route.RelayMode), string(route.RelayTransport)}, "/"), "/")
	}
	if width < 34 {
		availabilityText := availability
		if relay != "" {
			availabilityText += " " + relay
		}
		return []Line{
			{Cells: []Cell{NewCell(fmt.Sprintf("%s %s", route.ID, route.Kind))}},
			{Cells: []Cell{styledCell("priority ", StyleMuted), NewCell(priority)}},
			{Cells: []Cell{tokenCell(availabilityText, style)}},
		}
	}
	if width < 56 {
		summary := fmt.Sprintf("%s  %s  priority %s", route.ID, route.Kind, priority)
		availabilityText := availability
		if relay != "" {
			availabilityText += "  " + relay
		}
		return []Line{
			{Cells: []Cell{NewCell(summary)}},
			{Cells: []Cell{tokenCell(availabilityText, style)}},
		}
	}
	text := fmt.Sprintf("%s  %s  priority %s", route.ID, route.Kind, priority)
	if relay != "" {
		text += "  " + relay
	}
	return []Line{{Cells: []Cell{NewCell(text), NewCell(" "), tokenCell(availability, style)}}}
}
