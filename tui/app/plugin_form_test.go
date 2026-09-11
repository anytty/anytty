package app

import (
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
	"testing"
)

func pluginFormFixture(t *testing.T) (state.Root, PluginDeps, *pluginTestService) {
	root, deps, service := pluginFixture(t)
	m := root.Plugins.Mounts["agents"]
	m.Actions = nil
	m.Nodes = []state.PluginNode{{ID: "form", Kind: "form", Action: "save", Children: []state.PluginNode{
		{ID: "name", Kind: "input", Text: "Name", Value: "old"},
		{ID: "mode", Kind: "select", Text: "Mode", Value: "one", Children: []state.PluginNode{{ID: "one", Kind: "option", Text: "First", Value: "one"}, {ID: "two", Kind: "option", Text: "Second", Value: "two"}}},
		{ID: "enabled", Kind: "checkbox", Text: "Enabled", Value: "false"},
		{ID: "submit", Kind: "button", Text: "Save", Action: "save"},
	}}}
	m.SelectedID = "name"
	root.Plugins = root.Plugins.Set(m)
	return root, deps, service
}
func TestPluginFormInputEditingCancelAndSubmission(t *testing.T) {
	root, deps, service := pluginFormFixture(t)
	reducer := NewPluginReducer(deps)
	send := func(event input.InputEvent) {
		t.Helper()
		var effects []Effect
		root, effects = reducer(root, InputMsg{Event: event})
		runPluginEffects(t, effects)
	}
	send(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter})
	if service.messages[len(service.messages)-1].GetInteraction().Kind != "focus" {
		t.Fatal("field focus not routed")
	}
	send(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "中"})
	send(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyLeft})
	send(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"})
	change := service.messages[len(service.messages)-1].GetInteraction()
	if change.Kind != "change" || change.Value != "oldx中" || change.Values["name"] != "oldx中" {
		t.Fatalf("edit/cursor values not routed: %v", change)
	}
	send(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc})
	if root.Plugins.Mounts["agents"].FormValues()["name"] != "old" || service.messages[len(service.messages)-1].GetInteraction().Kind != "cancel" {
		t.Fatal("Escape must restore initial value and notify")
	}
	send(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter})
	send(input.InputEvent{Kind: input.EventKindPaste, Paste: "\nnew\x1b"})
	send(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter})
	submit := service.messages[len(service.messages)-1].GetInteraction()
	if submit.Kind != "submit" || submit.ActionId != "save" || submit.Values["name"] != "oldnew" || submit.Values["mode"] != "one" || submit.Context.GetContextId() == "" {
		t.Fatalf("submit lacks complete typed values/context: %v", submit)
	}
	binding, _ := root.TerminalViews.PaneBinding(state.DefaultPaneID)
	if binding.TerminalID != "old" {
		t.Fatal("form submit must await daemon operation before mutation")
	}
}
func TestPluginFormMouseSelectCheckboxButtonAndKeyboardAgree(t *testing.T) {
	root, deps, service := pluginFormFixture(t)
	reducer := NewPluginReducer(deps)
	for _, id := range []string{"mode", "enabled", "submit"} {
		var effects []Effect
		root, effects = reducer(root, PluginInputMsg{MountID: "agents", NodeID: id, Event: input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseLeft}})
		runPluginEffects(t, effects)
	}
	result := service.messages[len(service.messages)-1].GetInteraction()
	if result.Kind != "submit" || result.Values["mode"] != "two" || result.Values["enabled"] != "true" {
		t.Fatalf("mouse form controls failed: %v", result)
	}
	root, deps, service = pluginFormFixture(t)
	reducer = NewPluginReducer(deps)
	for _, key := range []input.Key{input.KeyTab, input.KeyEnter, input.KeyRight, input.KeyTab, input.KeyEnter, input.KeyTab, input.KeyEnter} {
		var effects []Effect
		root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: key}})
		runPluginEffects(t, effects)
	}
	result = service.messages[len(service.messages)-1].GetInteraction()
	if result.Kind != "submit" || result.Values["mode"] != "two" || result.Values["enabled"] != "true" {
		t.Fatalf("keyboard form controls differ: %v", result)
	}
}
func TestPluginTreeExpansionUsesRoutedSemanticEvent(t *testing.T) {
	root, deps, service := pluginFixture(t)
	m := root.Plugins.Mounts["agents"]
	m.Nodes = []state.PluginNode{{ID: "branch", Kind: "tree", Text: "Project", Children: []state.PluginNode{{ID: "child", Kind: "row", Text: "Agent", Action: "open"}}}}
	m.SelectedID = "branch"
	root.Plugins = root.Plugins.Set(m)
	root, effects := NewPluginReducer(deps)(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyLeft}})
	runPluginEffects(t, effects)
	if len(root.Plugins.Mounts["agents"].Rows()) != 1 || service.messages[0].GetInteraction().Kind != "collapse" {
		t.Fatal("tree collapse must preserve branch and notify daemon")
	}
	root, effects = NewPluginReducer(deps)(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyRight}})
	runPluginEffects(t, effects)
	if len(root.Plugins.Mounts["agents"].Rows()) != 2 {
		t.Fatal("tree expansion failed")
	}
}
func TestPluginFormProtobufRegistrationSupported(t *testing.T) {
	n := &apipb.PluginUiNode{Id: "form", Kind: "form", Children: []*apipb.PluginUiNode{{Id: "input", Kind: "input", Text: "Name", Value: "hello", Placeholder: "Enter name"}, {Id: "submit", Kind: "button", Text: "Save", ActionId: "save"}}}
	if err := validatePluginMount(&apipb.PluginUiMountUpdate{Root: n}); err != nil {
		t.Fatal(err)
	}
	nodes := pluginNodes(n, PluginDeps{})
	if nodes[0].Children[0].Value != "hello" || nodes[0].Children[0].Placeholder != "Enter name" {
		t.Fatal("generated protobuf value fields not projected")
	}
}
