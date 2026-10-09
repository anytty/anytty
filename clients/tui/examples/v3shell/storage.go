package main

import (
	"encoding/json"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk/app"
	apipb "github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	gproto "google.golang.org/protobuf/proto"
)

// v3shell persists its workbench through the access storage API via access.call
// (SDK-GUIDE §5.2), the same mechanism herdr uses. The legacy TUI kept this in
// a host-side workbench store; the v2 contract has no typed workbench method,
// so a program-owned storage partition is the portable equivalent. AppId/scope
// are the program identity, the key is fixed, and the value is a versioned
// plain-JSON document so a future version can migrate it.
const (
	v3shellAppID    = "v3shell"
	workbenchKey    = "workbench"
	workbenchDocVer = 1
)

// workbenchDoc is the persisted workbench: workspaces/tabs/panes, the recursive
// split tree with its size hints, focus, and the header/footer visibility.
type workbenchDoc struct {
	Version         int                  `json:"version"`
	ActiveWorkspace int                  `json:"active_workspace"`
	HeaderVisible   bool                 `json:"header_visible"`
	FooterVisible   bool                 `json:"footer_visible"`
	Workspaces      []workbenchWorkspace `json:"workspaces"`
}

type workbenchWorkspace struct {
	Name      string         `json:"name"`
	ActiveTab int            `json:"active_tab"`
	Tabs      []workbenchTab `json:"tabs"`
}

type workbenchTab struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Focus    int             `json:"focus"`
	SplitSeq int             `json:"split_seq"`
	Root     *workbenchNode  `json:"root,omitempty"`
	Panes    []workbenchPane `json:"panes"`
}

type workbenchPane struct {
	ID               string `json:"id"`
	Title            string `json:"title,omitempty"`
	SourceID         string `json:"source_id,omitempty"`
	DetachedSourceID string `json:"detached_source_id,omitempty"`
	// Layout is the pane's view-local content layout. It is omitempty and every
	// field defaults to the legacy auto/start/start/0/0, so older documents
	// (and default panes) stay byte-identical and restore compatibly.
	Layout *workbenchLayout `json:"layout,omitempty"`
}

// workbenchLayout is the persisted form of contentLayout. Empty strings and
// zero pan encode the legacy defaults, so a default pane serializes to nothing.
type workbenchLayout struct {
	Mode   string `json:"mode,omitempty"`
	AlignX string `json:"align_x,omitempty"`
	AlignY string `json:"align_y,omitempty"`
	PanX   int    `json:"pan_x,omitempty"`
	PanY   int    `json:"pan_y,omitempty"`
}

// workbenchNode is one recursive split-tree node: a split (orient/ratio/bias
// and its two children) or a leaf naming a pane id. seq is preserved so the
// restored tree keeps the same divider identities.
type workbenchNode struct {
	Orient string         `json:"orient,omitempty"`
	Ratio  float64        `json:"ratio,omitempty"`
	Bias   int            `json:"bias,omitempty"`
	Seq    int            `json:"seq,omitempty"`
	A      *workbenchNode `json:"a,omitempty"`
	B      *workbenchNode `json:"b,omitempty"`
	PaneID string         `json:"pane,omitempty"`
}

// workbenchDoc snapshots the in-memory model. Panes carry their source binding
// so a restart can re-bind by id once the next sources snapshot arrives.
func (m *model) workbenchDoc() workbenchDoc {
	doc := workbenchDoc{
		Version:         workbenchDocVer,
		ActiveWorkspace: m.space,
		HeaderVisible:   m.headerVisible,
		FooterVisible:   m.footerVisible,
		Workspaces:      make([]workbenchWorkspace, 0, len(m.spaces)),
	}
	for _, ws := range m.spaces {
		entry := workbenchWorkspace{Name: ws.name, ActiveTab: ws.active, Tabs: make([]workbenchTab, 0, len(ws.tabs))}
		for _, t := range ws.tabs {
			tabEntry := workbenchTab{ID: t.id, Title: t.title, Focus: t.focus, SplitSeq: t.splitSeq, Root: workbenchNodeOf(t.root)}
			for _, p := range t.panes {
				tabEntry.Panes = append(tabEntry.Panes, workbenchPane{
					ID: p.id, Title: p.title, SourceID: p.sourceID, DetachedSourceID: p.detachedSourceID,
					Layout: workbenchLayoutOf(p.layout),
				})
			}
			entry.Tabs = append(entry.Tabs, tabEntry)
		}
		doc.Workspaces = append(doc.Workspaces, entry)
	}
	return doc
}

// workbenchLayoutOf encodes a pane's content layout, returning nil for the
// default so a default pane serializes without a layout field (backward
// compatible documents).
func workbenchLayoutOf(layout contentLayout) *workbenchLayout {
	layout = layout.normalized()
	if layout.isDefault() {
		return nil
	}
	return &workbenchLayout{
		Mode: layout.mode, AlignX: layout.alignX, AlignY: layout.alignY,
		PanX: layout.panX, PanY: layout.panY,
	}
}

// contentLayoutFromWorkbench decodes a persisted layout, defaulting every
// missing/zero field to the legacy auto/start/start/0/0.
func contentLayoutFromWorkbench(doc *workbenchLayout) contentLayout {
	if doc == nil {
		return contentLayout{}.normalized()
	}
	return contentLayout{
		mode: doc.Mode, alignX: doc.AlignX, alignY: doc.AlignY, panX: doc.PanX, panY: doc.PanY,
	}.normalized()
}

func workbenchNodeOf(node treeNode) *workbenchNode {
	switch n := node.(type) {
	case *leaf:
		if n == nil || n.pane == nil {
			return nil
		}
		return &workbenchNode{PaneID: n.pane.id}
	case *split:
		if n == nil {
			return nil
		}
		return &workbenchNode{Orient: n.orient, Ratio: n.ratio, Bias: n.bias, Seq: n.seq, A: workbenchNodeOf(n.a), B: workbenchNodeOf(n.b)}
	}
	return nil
}

// applyWorkbenchDoc replaces the model layout from a persisted document. It
// validates and normalizes every entry and rebuilds the split tree; sources
// are re-bound by id on the next sources snapshot (an unbound pane shows as an
// empty panel). It reports false when the document has nothing usable, so the
// caller keeps the default seed.
func (m *model) applyWorkbenchDoc(doc workbenchDoc) bool {
	spaces := make([]*workspace, 0, len(doc.Workspaces))
	for _, wsEntry := range doc.Workspaces {
		ws := &workspace{name: wsEntry.Name}
		for _, tabEntry := range wsEntry.Tabs {
			if tabEntry.ID == "" {
				continue
			}
			panes := make(map[string]*pane, len(tabEntry.Panes))
			var ordered []*pane
			for _, pe := range tabEntry.Panes {
				if pe.ID == "" {
					continue
				}
				p := &pane{id: pe.ID, title: pe.Title, sourceID: pe.SourceID, detachedSourceID: pe.DetachedSourceID, layout: contentLayoutFromWorkbench(pe.Layout)}
				panes[pe.ID] = p
				ordered = append(ordered, p)
			}
			if len(ordered) == 0 {
				continue
			}
			root := buildWorkbenchNode(tabEntry.Root, panes)
			if root == nil {
				root = &leaf{pane: ordered[0]}
			}
			t := &tab{id: tabEntry.ID, title: tabEntry.Title, root: root, splitSeq: maxInt(tabEntry.SplitSeq, maxSplitSeq(root))}
			t.rebuild()
			if len(t.panes) == 0 {
				continue
			}
			t.focus = clampInt(tabEntry.Focus, 0, len(t.panes)-1)
			ws.tabs = append(ws.tabs, t)
		}
		if len(ws.tabs) == 0 {
			continue
		}
		ws.active = clampInt(wsEntry.ActiveTab, 0, len(ws.tabs)-1)
		spaces = append(spaces, ws)
	}
	if len(spaces) == 0 {
		return false
	}
	m.spaces = spaces
	m.space = clampInt(doc.ActiveWorkspace, 0, len(spaces)-1)
	m.headerVisible = doc.HeaderVisible
	m.footerVisible = doc.FooterVisible
	m.zoomPane = ""
	m.tabSeq = maxInt(m.tabSeq, maxTabSeq(spaces))
	m.paneSeq = maxInt(m.paneSeq, maxPaneSeq(spaces))
	return true
}

// buildWorkbenchNode reconstructs one tree node from its document form. A leaf
// whose pane id is unknown, or a split with no surviving children, is dropped
// so the restored tree never references a missing pane.
func buildWorkbenchNode(node *workbenchNode, panes map[string]*pane) treeNode {
	if node == nil {
		return nil
	}
	if node.PaneID != "" {
		if p := panes[node.PaneID]; p != nil {
			return &leaf{pane: p}
		}
		return nil
	}
	a := buildWorkbenchNode(node.A, panes)
	b := buildWorkbenchNode(node.B, panes)
	switch {
	case a == nil && b == nil:
		return nil
	case a == nil:
		return b
	case b == nil:
		return a
	}
	orient := node.Orient
	if orient != "col" {
		orient = "row"
	}
	return &split{orient: orient, ratio: node.Ratio, bias: node.Bias, seq: node.Seq, a: a, b: b}
}

func maxSplitSeq(node treeNode) int {
	sp, ok := node.(*split)
	if !ok || sp == nil {
		return 0
	}
	return maxInt(sp.seq, maxInt(maxSplitSeq(sp.a), maxSplitSeq(sp.b)))
}

func maxTabSeq(spaces []*workspace) int {
	max := 0
	for _, ws := range spaces {
		for _, t := range ws.tabs {
			if n := atoiNode(t.id, "tab-"); n > max {
				max = n
			}
		}
	}
	return max
}

func maxPaneSeq(spaces []*workspace) int {
	max := 0
	for _, ws := range spaces {
		for _, t := range ws.tabs {
			for _, p := range t.panes {
				if n := atoiNode(p.id, "pane-"); n > max {
					max = n
				}
			}
		}
	}
	return max
}

// --- access.call storage ---

// workbenchStorageKey is the program identity + fixed key for the persisted
// workbench (SDK-GUIDE §5.2): AppId "v3shell", PRIVATE scope, key "workbench".
func workbenchStorageKey() *apipb.StorageKey {
	return &apipb.StorageKey{AppId: v3shellAppID, Scope: apipb.StorageScope_STORAGE_SCOPE_PRIVATE, Key: workbenchKey}
}

// storageGetCmd asks the host for the saved workbench through access.call.
func (m *model) storageGetCmd() app.Cmd {
	command, err := gproto.Marshal(&apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_StorageGet{StorageGet: &apipb.StorageGetCommand{Key: workbenchStorageKey()}},
	})
	if err != nil {
		return nil
	}
	return m.emit("access.call", &pb.MethodParams{Endpoint: "local", AccessCommand: command}, opMsg{op: "workbench.get"})
}

// storageSetCmd writes the workbench document through access.call.
func (m *model) storageSetCmd(value []byte) app.Cmd {
	command, err := gproto.Marshal(&apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_StoragePut{StoragePut: &apipb.StoragePutCommand{Key: workbenchStorageKey(), Value: value}},
	})
	if err != nil {
		return nil
	}
	return m.emit("access.call", &pb.MethodParams{Endpoint: "local", AccessCommand: command}, opMsg{op: "workbench.set"})
}

// loadWorkbenchCmd runs once at HELLO: it asks for the saved workbench and
// applies it in onWorkbenchGet. Demo/offline runs and hosts without a client
// keep the default seed and never touch storage.
func (m *model) loadWorkbenchCmd() app.Cmd {
	if !m.host || m.demo || m.client == nil {
		return nil
	}
	return m.storageGetCmd()
}

// saveWorkbenchCmd persists the workbench, coalescing bursts: while a save is
// in flight the flag stays dirty and the next structural edit retries (herdr's
// layout save gate). It is scheduled by Update's tail.
func (m *model) saveWorkbenchCmd() app.Cmd {
	if m.client == nil || !m.workbenchReady || m.savePending || !m.workbenchDirty {
		return nil
	}
	value, err := json.Marshal(m.workbenchDoc())
	if err != nil {
		return nil
	}
	m.workbenchDirty = false
	m.savePending = true
	return m.storageSetCmd(value)
}

// markWorkbenchDirty flags a structural change for the next save. The save is
// scheduled by Update's tail; a flag keeps a burst of edits to one write.
func (m *model) markWorkbenchDirty() {
	if m.demo || m.client == nil {
		return
	}
	m.workbenchDirty = true
}

// onWorkbenchGet applies a saved workbench. A host rejection that means "not
// set" (or a NOT_FOUND ApiError envelope) keeps the default seed; any other
// failure is only logged to the message log.
func (m *model) onWorkbenchGet(v opMsg) app.Cmd {
	if !v.ok {
		if !isStorageNotFound(v.err) {
			if v.err == "" {
				m.notice("workbench restore failed: access unavailable")
			} else {
				m.notice("workbench restore failed: " + v.err)
			}
		}
		return m.markWorkbenchReady()
	}
	if len(v.accessResult) == 0 {
		m.notice("workbench restore failed: access unavailable")
		return m.markWorkbenchReady()
	}
	var envelope apipb.ResultEnvelope
	if err := gproto.Unmarshal(v.accessResult, &envelope); err != nil {
		m.notice("workbench restore failed: bad storage result")
		return m.markWorkbenchReady()
	}
	if apiErr := envelope.GetError(); apiErr != nil {
		if apiErr.GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND {
			m.notice("workbench restore failed: " + apiErr.GetMessage())
		}
		return m.markWorkbenchReady()
	}
	value := envelope.GetStorageGet().GetEntry().GetValue()
	if len(value) == 0 {
		return m.markWorkbenchReady()
	}
	var doc workbenchDoc
	if err := json.Unmarshal(value, &doc); err != nil {
		m.notice("workbench restore failed: bad json")
		return m.markWorkbenchReady()
	}
	if m.applyWorkbenchDoc(doc) {
		m.notice("workbench restored")
	}
	return m.markWorkbenchReady()
}

// onWorkbenchSet clears the in-flight flag and retries when a newer change
// landed during the save; a failure keeps the change dirty for the next edit.
func (m *model) onWorkbenchSet(v opMsg) app.Cmd {
	m.savePending = false
	if !v.ok {
		m.workbenchDirty = true
		if v.err != "" {
			m.notice("workbench save failed: " + v.err)
		}
		return nil
	}
	if len(v.accessResult) > 0 {
		var envelope apipb.ResultEnvelope
		if err := gproto.Unmarshal(v.accessResult, &envelope); err == nil {
			if apiErr := envelope.GetError(); apiErr != nil {
				m.workbenchDirty = true
				m.notice("workbench save failed: " + apiErr.GetMessage())
				return nil
			}
		}
	}
	if m.workbenchDirty {
		return m.saveWorkbenchCmd()
	}
	return nil
}

// markWorkbenchReady opens the save gate and flushes a change made while the
// initial load was still in flight.
func (m *model) markWorkbenchReady() app.Cmd {
	m.workbenchReady = true
	if m.workbenchDirty {
		return m.saveWorkbenchCmd()
	}
	return nil
}

// isStorageNotFound reports whether a rejected storage call means "the key is
// not set". The host surfaces the access layer's not-found text on the RESPONSE
// itself when the storage stack rejects the call.
func isStorageNotFound(errText string) bool {
	return strings.Contains(strings.ToLower(errText), "not found")
}
