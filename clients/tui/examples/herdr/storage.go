package main

import (
	"encoding/json"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk/app"
	apipb "github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	gproto "google.golang.org/protobuf/proto"
)

// layoutDoc is the persisted layout (SPEC §7): workspaces/tabs/panes with
// names, split axis, weights, focus and the sidebar collapsed flag. It is
// deliberately plain JSON so a future version can migrate it.
type layoutDoc struct {
	Version          int               `json:"version"`
	ActiveWorkspace  string            `json:"active_workspace"`
	SidebarCollapsed bool              `json:"sidebar_collapsed"`
	Workspaces       []layoutWorkspace `json:"workspaces"`
}

type layoutWorkspace struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	ActiveTab string      `json:"active_tab"`
	Tabs      []layoutTab `json:"tabs"`
}

type layoutTab struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Axis    string       `json:"axis"`
	Weights []int        `json:"weights"`
	Focus   string       `json:"focus"`
	Panes   []layoutPane `json:"panes"`
}

type layoutPane struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Source string `json:"source,omitempty"`
}

// layoutDoc snapshots the in-memory model.
func (m *model) layoutDoc() layoutDoc {
	doc := layoutDoc{
		Version:          1,
		ActiveWorkspace:  m.activeWorkspace,
		SidebarCollapsed: m.sidebarCollapsed,
		Workspaces:       make([]layoutWorkspace, 0, len(m.workspaces)),
	}
	for _, ws := range m.workspaces {
		entry := layoutWorkspace{ID: ws.ID, Name: ws.Name, ActiveTab: ws.ActiveTab}
		for _, t := range ws.Tabs {
			tabEntry := layoutTab{
				ID:      t.ID,
				Name:    t.Name,
				Axis:    t.Axis,
				Weights: append([]int(nil), t.Weights...),
				Focus:   t.Focus,
			}
			for _, p := range t.Panes {
				tabEntry.Panes = append(tabEntry.Panes, layoutPane{ID: p.ID, Name: p.Name, Source: p.Source})
			}
			entry.Tabs = append(entry.Tabs, tabEntry)
		}
		doc.Workspaces = append(doc.Workspaces, entry)
	}
	return doc
}

// applyLayoutDoc replaces the model layout from a persisted document. It
// validates and normalizes every entry, then re-binds nothing: the next
// sources snapshot resolves pane sources by id (unbound panes show gone).
func (m *model) applyLayoutDoc(doc layoutDoc) bool {
	workspaces := make([]*workspace, 0, len(doc.Workspaces))
	for _, wsEntry := range doc.Workspaces {
		if wsEntry.ID == "" {
			continue
		}
		ws := &workspace{ID: wsEntry.ID, Name: wsEntry.Name, ActiveTab: wsEntry.ActiveTab}
		for _, tabEntry := range wsEntry.Tabs {
			if tabEntry.ID == "" {
				continue
			}
			t := &tab{
				ID:      tabEntry.ID,
				Name:    tabEntry.Name,
				Axis:    tabEntry.Axis,
				Weights: append([]int(nil), tabEntry.Weights...),
				Focus:   tabEntry.Focus,
			}
			if t.Axis != "col" {
				t.Axis = "row"
			}
			for _, paneEntry := range tabEntry.Panes {
				if paneEntry.ID == "" {
					continue
				}
				t.Panes = append(t.Panes, &pane{ID: paneEntry.ID, Name: paneEntry.Name, Source: paneEntry.Source})
			}
			if len(t.Panes) == 0 {
				continue
			}
			if len(t.Weights) != len(t.Panes) {
				t.Weights = equalWeights(len(t.Panes))
			} else {
				t.Weights = normalizeTo100(t.Weights)
			}
			if paneByID(t, t.Focus) == nil {
				t.Focus = t.Panes[0].ID
			}
			ws.Tabs = append(ws.Tabs, t)
		}
		if len(ws.Tabs) == 0 {
			continue
		}
		if tabByID(ws, ws.ActiveTab) == nil {
			ws.ActiveTab = ws.Tabs[0].ID
		}
		workspaces = append(workspaces, ws)
	}
	if len(workspaces) == 0 {
		return false
	}
	m.workspaces = workspaces
	if m.workspaceByID(doc.ActiveWorkspace) != nil {
		m.activeWorkspace = doc.ActiveWorkspace
	} else {
		m.activeWorkspace = workspaces[0].ID
	}
	m.sidebarCollapsed = doc.SidebarCollapsed
	m.zoom = false
	m.navWS = m.currentWSIndex()
	m.touch()
	return true
}

// saveLayoutCmd persists the layout through access.call storage (AppId
// "herdr", scope PRIVATE, key "layout"). Failures only toast; a save that
// lands while another one is in flight is coalesced.
func (m *model) saveLayoutCmd() app.Cmd {
	if m.client == nil || !m.layoutReady || m.savePending || !m.layoutDirty {
		return nil
	}
	value, err := json.Marshal(m.layoutDoc())
	if err != nil {
		return nil
	}
	m.layoutDirty = false
	m.savePending = true
	return m.storageSetCmd(layoutKey, value)
}

// --- access.call storage (SDK-GUIDE §5.2) ---

func (m *model) storageGetCmd(key string) app.Cmd {
	if m.client == nil {
		return app.None
	}
	command, err := gproto.Marshal(&apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_StorageGet{StorageGet: &apipb.StorageGetCommand{Key: storageKey(key)}},
	})
	if err != nil {
		return app.None
	}
	return m.storageCall(command, "get", key, decodeStorageGet)
}

func (m *model) storageSetCmd(key string, value []byte) app.Cmd {
	if m.client == nil {
		return app.None
	}
	command, err := gproto.Marshal(&apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_StoragePut{StoragePut: &apipb.StoragePutCommand{
			Key:   storageKey(key),
			Value: value,
		}},
	})
	if err != nil {
		return app.None
	}
	return m.storageCall(command, "set", key, decodeStorageSet)
}

// storageCall runs one access.call and keeps the host's RESPONSE error text.
// app.Emit only hands the decode callback the access_result bytes, which are
// empty whenever the host rejected the call (for example while the local
// access stack is still starting), so the toast would lose the reason.
func (m *model) storageCall(command []byte, op, key string, decode func([]byte) app.Msg) app.Cmd {
	client := m.client
	params := &pb.MethodParams{Endpoint: "local", AccessCommand: command}
	return func() app.Msg {
		responses := make(chan *pb.Response, 1)
		if _, err := client.Emit("access.call", params, func(resp *pb.Response) { responses <- resp }); err != nil {
			return storageMsg{op: op, key: key, err: err.Error()}
		}
		resp := <-responses
		if !resp.GetOk() {
			errText := resp.GetError()
			if errText == "" {
				errText = "access unavailable"
			}
			// A missing entry is not a failure: the host may reject the GET
			// itself with the storage not-found text (or answer an ApiError
			// NOT_FOUND envelope, decoded below). First run has no saved
			// layout and must stay silent (SPEC §7).
			if op == "get" && isStorageNotFound(errText) {
				return storageMsg{op: op, key: key, ok: true, empty: true}
			}
			return storageMsg{op: op, key: key, err: errText}
		}
		msg := decode(resp.GetData().GetAccessResult())
		if result, ok := msg.(storageMsg); ok {
			result.key = key
			return result
		}
		return msg
	}
}

// isStorageNotFound reports whether a rejected storage GET means "the key is
// not set". The host surfaces the access layer's not-found text on the
// RESPONSE itself when the storage stack rejects the call.
func isStorageNotFound(errText string) bool {
	return strings.Contains(strings.ToLower(errText), "not found")
}

// decodeStorageGet turns data.access_result into a storageMsg. An empty
// payload means the host could not reach access (RESPONSE not ok); a
// NOT_FOUND envelope means the key is not set — neither is fatal.
func decodeStorageGet(payload []byte) app.Msg {
	if len(payload) == 0 {
		return storageMsg{op: "get", err: "access unavailable"}
	}
	var envelope apipb.ResultEnvelope
	if err := gproto.Unmarshal(payload, &envelope); err != nil {
		return storageMsg{op: "get", err: "bad storage result"}
	}
	if apiErr := envelope.GetError(); apiErr != nil {
		if apiErr.GetCode() == apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND {
			return storageMsg{op: "get", ok: true, empty: true}
		}
		return storageMsg{op: "get", err: apiErr.GetMessage()}
	}
	return storageMsg{op: "get", ok: true, value: string(envelope.GetStorageGet().GetEntry().GetValue())}
}

func decodeStorageSet(payload []byte) app.Msg {
	if len(payload) == 0 {
		return storageMsg{op: "set", err: "access unavailable"}
	}
	var envelope apipb.ResultEnvelope
	if err := gproto.Unmarshal(payload, &envelope); err != nil {
		return storageMsg{op: "set", err: "bad storage result"}
	}
	if apiErr := envelope.GetError(); apiErr != nil {
		return storageMsg{op: "set", err: apiErr.GetMessage()}
	}
	return storageMsg{op: "set", ok: true}
}

// markLayoutReady opens the save gate and flushes a change made while the
// initial load was still in flight.
func (m *model) markLayoutReady() app.Cmd {
	m.layoutReady = true
	if m.layoutDirty {
		return m.saveLayoutCmd()
	}
	return nil
}

func (m *model) onStorage(v storageMsg) app.Cmd {
	switch v.op {
	case "get":
		switch v.key {
		case layoutKey:
			return m.onLayoutGet(v)
		case legacyPinKey:
			return m.onPinGet(v)
		}
	case "set":
		m.savePending = false
		if !v.ok {
			// Keep the change pending so the next structural edit retries.
			m.layoutDirty = true
			m.setToast("layout save failed: "+v.err, true)
			return app.None
		}
		if m.layoutDirty {
			return m.saveLayoutCmd()
		}
	}
	return app.None
}

func (m *model) onLayoutGet(v storageMsg) app.Cmd {
	if v.err != "" {
		m.setToast("layout restore failed: "+v.err, true)
		return m.markLayoutReady()
	}
	if !v.empty && v.value != "" {
		if m.layoutUserTouched {
			m.setToast("layout restore skipped: local changes", false)
			return m.markLayoutReady()
		}
		var doc layoutDoc
		if err := json.Unmarshal([]byte(v.value), &doc); err != nil {
			m.setToast("layout restore failed: bad json", true)
			return m.markLayoutReady()
		}
		if m.applyLayoutDoc(doc) {
			m.setToast("layout restored", false)
		}
		return m.markLayoutReady()
	}
	// No persisted layout: fall back to the legacy `pinned` hint, then let
	// the default seed stay until the hint binds (SPEC §7).
	if m.client != nil {
		return m.storageGetCmd(legacyPinKey)
	}
	return m.markLayoutReady()
}

func (m *model) onPinGet(v storageMsg) app.Cmd {
	if v.err != "" {
		m.setToast("pin restore failed: "+v.err, true)
		return m.markLayoutReady()
	}
	if !v.empty && v.value != "" {
		m.pendingPin = v.value
		m.bindPendingPin()
	}
	return m.markLayoutReady()
}

// bindPendingPin binds the legacy pinned source to the first empty pane once
// it appears in the sources snapshot.
func (m *model) bindPendingPin() bool {
	if m.pendingPin == "" {
		return false
	}
	if _, ok := m.sources[m.pendingPin]; !ok {
		return false
	}
	ws := m.currentWS()
	if ws == nil {
		return false
	}
	for _, t := range ws.Tabs {
		for _, p := range t.Panes {
			if p.Source == "" {
				p.Source = m.pendingPin
				m.pendingPin = ""
				m.structural()
				return true
			}
		}
	}
	return false
}
