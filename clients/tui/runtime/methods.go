package runtime

import pb "github.com/anytty/anytty/proto/ui/protobuf"

// DataKind classifies the RESPONSE.data payload of a method (PROTOCOL §4).
type DataKind uint8

const (
	// DataNone means the method answers with an empty data field.
	DataNone DataKind = iota
	// DataCreate answers {endpoint,id}.
	DataCreate
	// DataRows answers historical rows.
	DataRows
	// DataText answers a text blob.
	DataText
)

// Method is one row of the PROTOCOL §4 authoritative method registry. The
// registry is the single source for HELLO.methods and for the whitelist the
// runtime accepts RESULT calls against.
type Method struct {
	Name string
	// Confirm marks destructive methods the host must confirm first.
	Confirm bool
	Data    DataKind
}

var methodRegistry = []Method{
	{Name: "terminal.attach", Data: DataNone},
	{Name: "terminal.create", Data: DataCreate},
	{Name: "terminal.restart", Data: DataNone},
	{Name: "terminal.kill", Confirm: true, Data: DataNone},
	{Name: "terminal.remove", Confirm: true, Data: DataNone},
	{Name: "terminal.scroll", Data: DataRows},
	{Name: "terminal.scrollEnd", Data: DataNone},
	{Name: "terminal.copy", Data: DataNone},
	{Name: "history.window", Data: DataRows},
	{Name: "clipboard.read", Confirm: true, Data: DataText},
	{Name: "input.forward", Data: DataNone},
	{Name: "system.quit", Confirm: true, Data: DataNone},
	// endpoint.sync registers one configured endpoint with the host and asks
	// it to connect in the background (ENDPOINTS.zh-CN.md §3). It is
	// append-only after system.quit so v1 programs keep their registry order.
	{Name: "endpoint.sync", Data: DataNone},
}

// Methods returns a copy of the §4 registry.
func Methods() []Method {
	return append([]Method(nil), methodRegistry...)
}

// MethodNames returns the registry names in table order. HELLO.methods is
// generated from it.
func MethodNames() []string {
	names := make([]string, len(methodRegistry))
	for i, m := range methodRegistry {
		names[i] = m.Name
	}
	return names
}

// LookupMethod returns the registry entry for name.
func LookupMethod(name string) (Method, bool) {
	for _, m := range methodRegistry {
		if m.Name == name {
			return m, true
		}
	}
	return Method{}, false
}

// validateParams checks the required fields of one method call. An empty
// return means the params are acceptable.
func validateParams(m Method, p *pb.MethodParams) string {
	switch m.Name {
	case "clipboard.read", "system.quit":
		return ""
	case "terminal.create":
		if p.GetEndpoint() == "" {
			return "missing params.endpoint"
		}
		return ""
	case "endpoint.sync":
		if p.GetEndpoint() == "" {
			return "missing params.endpoint"
		}
		return ""
	case "input.forward":
		if p.GetEventId() == "" {
			return "missing params.event_id"
		}
		if p.GetSource() == "" {
			return "missing params.source"
		}
		return ""
	default:
		if p.GetEndpoint() == "" {
			return "missing params.endpoint"
		}
		if p.GetId() == "" {
			return "missing params.id"
		}
		return ""
	}
}

// eventNames lists the EVENT frame payloads the host can send (PROTOCOL §1).
func eventNames() []string {
	return []string{"key", "paste", "mouse", "wheel", "resize", "sources", "notice", "component", "view_rejected"}
}
