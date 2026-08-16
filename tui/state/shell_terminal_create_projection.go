package state

// TerminalCreateEndpointItems 返回当前允许新建 terminal 的 endpoint 投影。
// 它只消费 reducer-owned EndpointStore，作为 picker create 行和 app 创建 prompt 的共同真值；
// disabled、未注册、需要 reconnect 和尚未显式连接的 manual endpoint 不会被自动创建流程使用。
func TerminalCreateEndpointItems(root Root) []EndpointItem {
	if !root.Endpoints.HasItems() {
		return []EndpointItem{DefaultLocalEndpoint()}
	}
	endpoints := root.Endpoints.Normalize()
	items := make([]EndpointItem, 0, len(endpoints.Items))
	for _, endpoint := range endpoints.Items {
		endpoint = endpoint.withDefaults()
		if !terminalCreateEndpointAvailable(endpoint) {
			continue
		}
		items = append(items, endpoint)
	}
	return items
}

func terminalCreateEndpointAvailable(endpoint EndpointItem) bool {
	endpoint = endpoint.withDefaults()
	if endpoint.ID == "" || !endpoint.Enabled {
		return false
	}
	switch endpoint.DisplayStatus() {
	case EndpointStatusDisabled, EndpointStatusManual, EndpointStatusReconnectRequired, EndpointStatusUnregistered:
		return false
	default:
		return true
	}
}
