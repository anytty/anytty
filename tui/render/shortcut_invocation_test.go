package render

import (
	"strconv"
	"strings"
	"testing"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

func TestShortcutActionLabelPriorityUsesCanonicalDomainSpec(t *testing.T) {
	action := FooterActionVM{Label: "render fallback", ActionID: "menu.panel"}
	cases := []struct {
		name  string
		entry input.ShortcutEntry
		cfg   state.TUIConfigStore
		want  string
	}{
		{name: "binding label", entry: input.ShortcutEntry{ActionID: "menu.panel", Label: "binding"}, cfg: state.TUIConfigStore{Shortcuts: state.TUIShortcutConfig{Actions: map[string]state.TUIShortcutActionConfig{"menu.panel": {Label: "action"}}}}, want: "binding"},
		{name: "action label", entry: input.ShortcutEntry{ActionID: "menu.panel"}, cfg: state.TUIConfigStore{Shortcuts: state.TUIShortcutConfig{Actions: map[string]state.TUIShortcutActionConfig{"menu.panel": {Label: "action"}}}}, want: "action"},
		{name: "canonical alias action label", entry: input.ShortcutEntry{ActionID: "menu.pane"}, cfg: state.TUIConfigStore{Shortcuts: state.TUIShortcutConfig{Actions: map[string]state.TUIShortcutActionConfig{"menu.panel": {Label: "canonical action"}}}}, want: "canonical action"},
		{name: "domain default", entry: input.ShortcutEntry{ActionID: "panel.close"}, want: "CLOSE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shortcutActionLabel(tc.entry, tc.cfg, action); got != tc.want {
				t.Fatalf("label priority mismatch: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestActionOnlyShortcutConfigKeepsDefaultFooterBindings(t *testing.T) {
	root := state.Root{Config: state.TUIConfigStore{Shortcuts: state.TUIShortcutConfig{
		Configured: true,
		Actions: map[string]state.TUIShortcutActionConfig{
			"menu.panel": {Label: "custom panel", Style: string(StyleFooterKeyWorkspace)},
		},
	}}}
	actions := footerActionCatalogFromShortcuts("live", root)
	if !containsFooterAction(actions, "^P", "custom panel", "menu.panel") {
		t.Fatalf("action-only config must retain default ctrl-p with overridden label, got %#v", actions)
	}
	if !containsFooterActionID(actions, "menu.resize") {
		t.Fatalf("action-only config must retain other default bindings, got %#v", actions)
	}
	panelMenu, ok := footerActionForID(actions, "menu.panel")
	if !ok || panelMenu.Style != StyleFooterKeyWorkspace {
		t.Fatalf("action-only style must override the built-in projection style, got %#v", actions)
	}

	root.Config.Shortcuts.Actions["panel.close"] = state.TUIShortcutActionConfig{Label: "dismiss pane"}
	panelActions := footerActionCatalogFromShortcuts("panel", root)
	if !containsFooterAction(panelActions, "x/w", "dismiss pane", "panel.close") {
		t.Fatalf("aggregated default bindings must preserve action-only label override, got %#v", panelActions)
	}
}

func TestShortcutShowPolicyOnlyFiltersFooter(t *testing.T) {
	hide := false
	show := true
	root := state.Root{Config: state.TUIConfigStore{Shortcuts: state.TUIShortcutConfig{
		Configured: true,
		Scenes: map[string]state.TUIShortcutSceneConfig{
			"floating": {Bindings: map[string]state.TUIShortcutBindingConfig{
				"n": {Action: "floating.new", Show: &hide},
				"o": {Action: "floating.overview"},
			}},
			"copy": {Bindings: map[string]state.TUIShortcutBindingConfig{
				"h": {Action: "copy.cursor_left", Show: &show},
			}},
		},
	}}}
	footerActions := footerActionCatalogFromShortcuts("floating", root)
	if containsFooterActionID(footerActions, "floating.new") || !containsFooterActionID(footerActions, "menu.floating_overview") {
		t.Fatalf("show false must only remove the selected footer action: %#v", footerActions)
	}
	copyFooter := footerActionCatalogFromShortcuts("copy", root)
	if !containsFooterActionID(copyFooter, "copy.cursor_left") {
		t.Fatalf("show true must expose a domain-default hidden action without a render ActionID mapping: %#v", copyFooter)
	}
	helpActions := helpActionCatalogFromShortcuts("floating", root)
	if !containsFooterActionID(helpActions, "floating.new") || !containsFooterActionID(helpActions, "menu.floating_overview") {
		t.Fatalf("help must retain all effective bindings regardless of footer show: %#v", helpActions)
	}
	copyHelp := helpActionCatalogFromShortcuts("copy", root)
	if !containsFooterActionID(copyHelp, "copy.cursor_left") {
		t.Fatalf("help must include Help-visible actions without a render ActionID mapping: %#v", copyHelp)
	}
}

func TestEnhancedShortcutVisibilityUsesHostCapability(t *testing.T) {
	show := true
	root := state.Root{Config: state.TUIConfigStore{Shortcuts: state.TUIShortcutConfig{
		Configured: true,
		Scenes: map[string]state.TUIShortcutSceneConfig{
			"global": {Bindings: map[string]state.TUIShortcutBindingConfig{
				"ctrl-1": {Action: "tab.jump.1", Show: &show},
				"ctrl-t": {Action: "menu.tab", Show: &show},
			}},
		},
	}}}
	withoutCapability := footerActionCatalogFromShortcuts("live", root)
	if containsFooterActionID(withoutCapability, "tab.jump") || !containsFooterActionID(withoutCapability, "menu.tab") {
		t.Fatalf("unavailable enhanced binding must be hidden while stable fallback remains: %#v", withoutCapability)
	}
	root.HostCapabilities = state.HostCapabilityStore{KeyboardProbed: true, KeyboardDisambiguation: true}
	withCapability := footerActionCatalogFromShortcuts("live", root)
	if !containsFooterActionID(withCapability, "tab.jump") {
		t.Fatalf("confirmed enhanced binding should enter footer catalog: %#v", withCapability)
	}
}

func TestFooterHitRegionCarriesExactShortcutInvocation(t *testing.T) {
	invocation, _, err := actiondomain.ParseInvocation("panel.close")
	if err != nil {
		t.Fatal(err)
	}
	footer := FooterVM{Visible: true, Mode: "pane", ActionTokens: []FooterActionVM{{Key: "q/w", Label: "CLOSE", ActionID: "panel.close", Invocation: invocation, Click: ClickClickable}}}
	regions := appendFooterHitRegions(nil, footer, Rect{W: 80, H: 1}, Rect{}, Rect{W: 80, H: 1})
	if len(regions) != 1 || regions[0].Invocation.Signature() != invocation.Signature() {
		t.Fatalf("footer hit region lost invocation: %#v", regions)
	}
}

func TestEveryExecutableHitRegionCarriesCanonicalInvocation(t *testing.T) {
	roots := []state.Root{
		{Shell: state.DefaultShell()},
		{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeGlobal)},
		{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModePane)},
		{Shell: state.DefaultShell().OpenTerminalPicker()},
		{Shell: state.DefaultShell().OpenTerminalPool(), TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", State: "running"}}}},
	}
	for rootIndex, root := range roots {
		for _, width := range []int{32, 120} {
			vm := NewRenderVMBuilder().Build(root)
			plan := MeasureLayout(vm.Shell, Rect{W: width, H: 24})
			for _, region := range plan.HitRegions {
				if region.ActionID == "" {
					continue
				}
				if region.Invocation.ID == "" {
					t.Fatalf("root=%d width=%d action=%q has no canonical invocation: %#v", rootIndex, width, region.ActionID, region)
				}
				if _, ok := actiondomain.SpecByID(region.Invocation.ID); !ok {
					t.Fatalf("root=%d width=%d action=%q references unknown invocation %q", rootIndex, width, region.ActionID, region.Invocation.ID)
				}
			}
		}
	}
}

func TestAppendRegionRejectsActionWithoutProducerInvocation(t *testing.T) {
	region := HitRegion{Kind: HitRegionContentAction, Rect: Rect{W: 10, H: 1}, ActionID: "empty.attach"}
	if got := appendRegion(nil, region, Rect{W: 80, H: 24}); len(got) != 0 {
		t.Fatalf("append layer must fail closed instead of deriving invocation: %#v", got)
	}
}

func TestAggregatedDifferentInvocationsAreHintOnly(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeTab)}
	actions := footerActionCatalogFromShortcuts("tab", root)
	for _, action := range actions {
		if action.ActionID == ActionTabSwitch.String() && (action.Click == ClickClickable || action.Invocation.ID != "") {
			t.Fatalf("tab jump aggregation must be hint-only: %#v", action)
		}
	}
	footer := FooterVM{Visible: true, Mode: "tab", ActionTokens: actions}
	regions := appendFooterHitRegions(nil, footer, Rect{W: 200, H: 1}, Rect{}, Rect{W: 200, H: 1})
	for _, region := range regions {
		if region.ActionID == ActionTabSwitch.String() {
			t.Fatalf("hint-only tab jump must not create hit region: %#v", region)
		}
	}
}

func TestFooterZeroClickPolicyDoesNotCreateHitRegion(t *testing.T) {
	footer := FooterVM{Visible: true, ActionTokens: []FooterActionVM{{
		Key:        "x",
		Label:      "ACTION",
		ActionID:   "legacy.action",
		Invocation: actiondomain.Invocation{ID: "panel.close"},
	}}}
	regions := appendFooterHitRegions(nil, footer, Rect{W: 80, H: 1}, Rect{}, Rect{W: 80, H: 1})
	if len(regions) != 0 {
		t.Fatalf("zero click policy must be non-clickable: %#v", regions)
	}
}

func TestEmptyFooterDoesNotInventGlobalShortcut(t *testing.T) {
	footer := FooterVM{Visible: true, Mode: "live"}
	if segments := footerLeftSegments(footer, 80); len(segments) != 0 {
		t.Fatalf("empty footer must not invent ctrl-g display or click target: %#v", segments)
	}
	if regions := appendFooterHitRegions(nil, footer, Rect{W: 80, H: 1}, Rect{}, Rect{W: 80, H: 1}); len(regions) != 0 {
		t.Fatalf("empty footer must not invent hit regions: %#v", regions)
	}
}

func TestEmptyHelpSceneRemovesCloseContentAction(t *testing.T) {
	root := state.Root{Config: state.TUIConfigStore{Shortcuts: state.TUIShortcutConfig{Configured: true, Scenes: map[string]state.TUIShortcutSceneConfig{
		"help": {Bindings: map[string]state.TUIShortcutBindingConfig{}},
	}}}}
	content := buildHelpContent(root)
	for _, line := range content.Lines {
		if strings.Contains(line.PlainString(), "Close Help") {
			t.Fatalf("empty help scene must remove close action text: %#v", content.Lines)
		}
	}
	if len(content.HitRegions) != 0 {
		t.Fatalf("empty help scene must remove close hit region: %#v", content.HitRegions)
	}
}

func TestOverlayShortcutBindingPreservesTargetContextAndHonorsCatalog(t *testing.T) {
	regions := []HitRegion{{
		Kind:     HitRegionContentAction,
		Rect:     Rect{W: 20, H: 1},
		PaneID:   "pane-2",
		Row:      3,
		ActionID: ActionWorkbenchOpen.String(),
		Invocation: actiondomain.Invocation{ID: "workbench_tree.open",
			SourceActionID: "workbench_tree.open"},
		TargetMode: HitTargetExplicit,
	}}
	bound := bindOverlayShortcutInvocations(OverlayWorkbenchTree, regions, state.TUIShortcutConfig{})
	if len(bound) != 1 || bound[0].Invocation.ID != "workbench_tree.open" || bound[0].PaneID != "pane-2" || bound[0].Row != 3 {
		t.Fatalf("overlay shortcut lost invocation or target context: %#v", bound)
	}

	emptyCatalog := state.TUIShortcutConfig{Configured: true, Scenes: map[string]state.TUIShortcutSceneConfig{
		"workbench_tree": {Bindings: map[string]state.TUIShortcutBindingConfig{}},
	}}
	if got := bindOverlayShortcutInvocations(OverlayWorkbenchTree, regions, emptyCatalog); len(got) != 0 {
		t.Fatalf("unconfigured overlay shortcut must not retain legacy click fallback: %#v", got)
	}
}

func TestFloatingOverviewRowsUseOpenInvocationBeyondNumericSummonRange(t *testing.T) {
	regions := make([]HitRegion, 10)
	for index := range regions {
		regions[index] = HitRegion{
			Kind:     HitRegionContentAction,
			Rect:     Rect{Y: index, W: 20, H: 1},
			PaneID:   "floating-" + strconv.Itoa(index+1),
			Row:      index,
			ActionID: ActionFloatingSummon.String(),
			Invocation: actiondomain.Invocation{ID: "floating_overview.open",
				SourceActionID: "floating_overview.open"},
			TargetMode: HitTargetExplicit,
		}
	}
	shortcuts := state.TUIShortcutConfig{Configured: true, Scenes: map[string]state.TUIShortcutSceneConfig{
		"floating_overview": {Bindings: map[string]state.TUIShortcutBindingConfig{
			"enter": {Action: "floating_overview.open"},
		}},
	}}
	bound := bindOverlayShortcutInvocations(OverlayFloatingOverview, regions, shortcuts)
	if len(bound) != 10 {
		t.Fatalf("all overview rows must remain clickable without numeric summon bindings: %d", len(bound))
	}
	if bound[9].Invocation.ID != "floating_overview.open" || bound[9].Row != 9 || bound[9].PaneID != "floating-10" {
		t.Fatalf("tenth overview row lost open invocation or context: %#v", bound[9])
	}
}
