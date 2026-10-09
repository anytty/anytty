package main

import (
	"encoding/json"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk/app"
	apipb "github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	gproto "google.golang.org/protobuf/proto"
)

// TestWorkbenchPersistRoundTrip pins the workbench document: workspaces, tabs,
// panes, the split tree with its size hints and focus, and the header/footer
// flags survive a serialize/apply cycle.
func TestWorkbenchPersistRoundTrip(t *testing.T) {
	m := newModel(nil, false)
	m.createWorkspace() // second workspace, active
	m.ws().name = "build"
	if clone := m.splitLeafFor("col", nil); clone == nil {
		t.Fatal("split failed")
	}
	tab := m.activeTab()
	sp, ok := tab.root.(*split)
	if !ok {
		t.Fatalf("root = %T, want split", tab.root)
	}
	sp.ratio = 0.25
	sp.bias = 3
	tab.focus = 1
	m.headerVisible = false
	m.footerVisible = true

	doc := m.workbenchDoc()
	if doc.Version != workbenchDocVer || len(doc.Workspaces) != 2 {
		t.Fatalf("doc = %+v", doc)
	}

	fresh := newModel(nil, false)
	if !fresh.applyWorkbenchDoc(doc) {
		t.Fatal("applyWorkbenchDoc returned false")
	}
	if len(fresh.spaces) != 2 || fresh.space != 1 {
		t.Fatalf("spaces=%d active=%d, want 2/1", len(fresh.spaces), fresh.space)
	}
	ws := fresh.spaces[1]
	if ws.name != "build" {
		t.Fatalf("workspace name = %q", ws.name)
	}
	gotTab := ws.tabs[0]
	if gotTab.id != tab.id || len(gotTab.panes) != 2 {
		t.Fatalf("tab = %+v, want id %q and 2 panes", gotTab, tab.id)
	}
	gotSplit, ok := gotTab.root.(*split)
	if !ok || gotSplit.orient != "col" || gotSplit.ratio != 0.25 || gotSplit.bias != 3 {
		t.Fatalf("split round-trip = %+v", gotTab.root)
	}
	if gotTab.focus != 1 {
		t.Fatalf("focus = %d, want 1", gotTab.focus)
	}
	if fresh.headerVisible || !fresh.footerVisible {
		t.Fatalf("header/footer = %v/%v, want false/true", fresh.headerVisible, fresh.footerVisible)
	}
}

// TestWorkbenchSaveEmitsStoragePut pins the access.call wire shape: AppId
// "v3shell", PRIVATE scope, key "workbench", and a decodable document value.
func TestWorkbenchSaveEmitsStoragePut(t *testing.T) {
	var captured *apipb.CommandEnvelope
	m := newModel(nil, false)
	m.client = emitterFunc(func(params *pb.MethodParams) *pb.Response {
		env := &apipb.CommandEnvelope{}
		if err := gproto.Unmarshal(params.GetAccessCommand(), env); err != nil {
			t.Fatalf("unmarshal access command: %v", err)
		}
		captured = env
		return &pb.Response{Ok: true}
	})
	m.host = true
	m.workbenchReady = true
	m.workbenchDirty = true

	runCmd(t, m, m.saveWorkbenchCmd())
	if captured == nil {
		t.Fatal("no access.call captured")
	}
	put := captured.GetStoragePut()
	if put == nil {
		t.Fatalf("command = %T, want StoragePut", captured.GetCommand())
	}
	key := put.GetKey()
	if key.GetAppId() != v3shellAppID || key.GetScope() != apipb.StorageScope_STORAGE_SCOPE_PRIVATE || key.GetKey() != workbenchKey {
		t.Fatalf("storage key = %+v", key)
	}
	var doc workbenchDoc
	if err := json.Unmarshal(put.GetValue(), &doc); err != nil {
		t.Fatalf("saved value is not a workbench doc: %v", err)
	}
	if len(doc.Workspaces) != 1 {
		t.Fatalf("saved workspaces = %d, want 1", len(doc.Workspaces))
	}
}

// TestWorkbenchLoadNotFoundKeepsSeed pins the first-run path: a NOT_FOUND on
// the initial get keeps the default seed, stays silent, and opens the save gate.
func TestWorkbenchLoadNotFoundKeepsSeed(t *testing.T) {
	m, _ := boundModel(t)
	m.logLines = nil
	before := len(m.spaces)
	payload, err := gproto.Marshal(&apipb.ResultEnvelope{
		Result: &apipb.ResultEnvelope_Error{Error: &apipb.ApiError{
			Code: apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, Message: "entry not found",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m.onOp(opMsg{op: "workbench.get", ok: true, accessResult: payload})
	if len(m.spaces) != before {
		t.Fatalf("not-found changed the seed: %d -> %d", before, len(m.spaces))
	}
	if len(m.logLines) != 0 {
		t.Fatalf("not-found must stay silent, log = %q", m.logLines)
	}
	if !m.workbenchReady {
		t.Fatal("not-found must open the save gate")
	}
}

// TestWorkbenchRestoreAppliesDoc pins the happy path: a saved doc is applied
// on the get response.
func TestWorkbenchRestoreAppliesDoc(t *testing.T) {
	m, _ := boundModel(t)
	doc := workbenchDoc{
		Version: workbenchDocVer, ActiveWorkspace: 0,
		HeaderVisible: true, FooterVisible: true,
		Workspaces: []workbenchWorkspace{{
			Name: "restored", ActiveTab: 0,
			Tabs: []workbenchTab{{
				ID: "tab-9", Title: "restored-tab", Focus: 0, SplitSeq: 1,
				Panes: []workbenchPane{{ID: "pane-7", Title: "term"}},
				Root:  &workbenchNode{PaneID: "pane-7"},
			}},
		}},
	}
	value, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := gproto.Marshal(&apipb.ResultEnvelope{
		Result: &apipb.ResultEnvelope_StorageGet{StorageGet: &apipb.StorageGetResult{
			Entry: &apipb.StorageEntry{Value: value},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m.onOp(opMsg{op: "workbench.get", ok: true, accessResult: payload})
	if len(m.spaces) != 1 || m.spaces[0].name != "restored" {
		t.Fatalf("restore failed: %+v", m.spaces)
	}
	if m.spaces[0].tabs[0].id != "tab-9" {
		t.Fatalf("tab id = %q", m.spaces[0].tabs[0].id)
	}
}

// emitterFunc adapts a response function to the model's emitter interface.
type emitterFunc func(params *pb.MethodParams) *pb.Response

func (f emitterFunc) Emit(_ string, params *pb.MethodParams, onResponse func(*pb.Response)) (uint64, error) {
	if onResponse != nil {
		onResponse(f(params))
	}
	return 1, nil
}

// TestWorkbenchUpdateTailSaves pins that a structural mutation reached through
// Update schedules exactly one coalesced save.
func TestWorkbenchUpdateTailSaves(t *testing.T) {
	var puts int
	m := newModel(nil, false)
	m.client = emitterFunc(func(params *pb.MethodParams) *pb.Response {
		env := &apipb.CommandEnvelope{}
		if err := gproto.Unmarshal(params.GetAccessCommand(), env); err == nil && env.GetStoragePut() != nil {
			puts++
		}
		return &pb.Response{Ok: true}
	})
	m.host = true
	m.workbenchReady = true

	// Enter PANE, then split: splitLeafFor marks the workbench dirty and the
	// Update tail schedules the coalesced save.
	runCmd(t, m, m.Update(app.KeyMsg{Key: &pb.KeyEvent{Key: "ctrl-p"}}))
	runCmd(t, m, m.Update(app.KeyMsg{Key: &pb.KeyEvent{Key: "%", Char: "%"}}))
	if puts == 0 {
		t.Fatal("Update tail did not schedule a save after a structural edit")
	}
	// A no-op Update does not save again while nothing is dirty.
	before := puts
	runCmd(t, m, m.Update(app.KeyMsg{Key: &pb.KeyEvent{Key: "esc"}}))
	if puts != before {
		t.Fatalf("clean Update saved again: %d -> %d", before, puts)
	}
}
