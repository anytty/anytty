package render

import (
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

func TestConnectionsOverlayProjectsActualRouteGenerationAndPriorityAtNarrowWidths(t *testing.T) {
	zero := 0
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24},
		Shell:    state.DefaultShell().OpenConnections(),
		Endpoints: (state.EndpointStore{}).Upsert(state.EndpointItem{
			ID: "studio", Label: "Studio", Enabled: true, Status: state.EndpointStatusConnected,
			RoutePreference: endpoint.RoutePreferenceAuto, ActiveRouteID: "cloud", ConnectionGeneration: 12,
			ObservedPath: "single_relay", RouteSelectionReason: "first_ready",
			Routes: []state.EndpointRouteItem{
				{ID: "cloud", Kind: state.EndpointTransportHubP2P, Enabled: true, Priority: &zero, AvailabilityKnown: true, Available: true, AvailabilityReason: endpoint.RouteAvailabilityAvailable, RelayMode: endpoint.RelayAuto, RelayTransport: endpoint.RelayTransportTCP},
				{ID: "ssh", Kind: state.EndpointTransportSSH, Enabled: true, Priority: &zero, AvailabilityKnown: true, Available: false, AvailabilityReason: endpoint.RouteAvailabilityCredentialUnavailable},
			},
		}),
	}
	for _, cols := range []int{80, 56} {
		root.Viewport.Cols = cols
		vm := NewRenderVMBuilder().Build(root)
		if vm.Shell.Overlay.Kind != OverlayConnections || vm.Shell.Overlay.Content.Kind != ContentConnections || !vm.Shell.Overlay.Opaque {
			t.Fatalf("connections overlay projection = %#v", vm.Shell.Overlay)
		}
		frame := NewRenderer(DefaultTheme()).Render(vm)
		text := frameText(frame)
		for _, want := range []string{"Connections", "CURRENT CONNECTION", "NEXT CONNECTION POLICY", "cloud", "single_relay", "12", "priority 0", "available", "credential missing"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%d-column frame missing %q:\n%s", cols, want, text)
			}
		}
		if strings.Contains(text, "connected") {
			t.Fatalf("%d-column frame should render endpoint status as an icon:\n%s", cols, text)
		}
	}
}

func TestConnectionsOverlayShowsSelectedPairAddressesAndRTT(t *testing.T) {
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 36},
		Shell:    state.DefaultShell().OpenConnections(),
		Endpoints: (state.EndpointStore{}).Upsert(state.EndpointItem{
			ID: "studio", Label: "Studio", Enabled: true, Status: state.EndpointStatusConnected,
			ActiveRouteID: "cloud", ConnectionGeneration: 12, ObservedPath: "direct",
			ConnectionSnapshot: state.EndpointConnectionSnapshot{
				RoundTrip:    42*time.Millisecond + 500*time.Microsecond,
				LocalAddress: "192.0.2.10", LocalPort: 41000, RemoteAddress: "2001:db8::20", RemotePort: 41121,
				LocalCandidateType: "srflx", RemoteCandidateType: "host", LocalProtocol: "udp", RemoteProtocol: "udp",
			},
		}),
	}
	text := frameText(NewRenderer(DefaultTheme()).Render(NewRenderVMBuilder().Build(root)))
	for _, want := range []string{"192.0.2.10:41000", "[2001:db8::20]:41121", "42.5 ms", "srflx / host", "udp / udp"} {
		if !strings.Contains(text, want) {
			t.Fatalf("connections frame missing %q:\n%s", want, text)
		}
	}
}

func TestConnectionsOverlayShowsEnabledCheckboxAndDrainingState(t *testing.T) {
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		Shell:    state.DefaultShell().OpenConnections(),
		Endpoints: (state.EndpointStore{}).Upsert(state.EndpointItem{
			ID: "studio", Label: "Studio", Enabled: false, Status: state.EndpointStatusConnected,
		}),
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView(
		"studio", "pane-1", "term-1", 7, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-1"), false,
	))
	text := frameText(NewRenderer(DefaultTheme()).Render(NewRenderVMBuilder().Build(root)))
	for _, want := range []string{"[ ] Studio ○", "Enabled: no", "Drain: 1 active view(s)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("connections frame missing %q:\n%s", want, text)
		}
	}
	for _, hidden := range []string{"connected", "disabled", "draining"} {
		if strings.Contains(text, hidden) {
			t.Fatalf("connections frame should hide endpoint status text %q:\n%s", hidden, text)
		}
	}
}

func TestConnectionsOverlayUsesConfiguredEndpointStatusAppearance(t *testing.T) {
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		Shell:    state.DefaultShell().OpenConnections(),
		Config: state.TUIConfigStore{Chrome: state.TUIChromeConfig{Picker: state.TUIPickerChromeConfig{
			EndpointStatus: state.TUIPickerEndpointStatusConfig{
				Connected: state.TUIEndpointStatusAppearanceConfig{Glyph: "+", Style: "danger"},
			},
		}}},
		Endpoints: (state.EndpointStore{}).Upsert(state.EndpointItem{
			ID: "studio", Label: "Studio", Enabled: true, Status: state.EndpointStatusConnected,
		}),
	}

	content := NewRenderVMBuilder().Build(root).Shell.Overlay.Content
	styledStatusCount := 0
	for _, line := range content.Lines {
		for _, cell := range line.Cells {
			if cell.Text == "+" && cell.Style == StyleDanger {
				styledStatusCount++
			}
		}
	}
	if styledStatusCount < 2 || strings.Contains(plainLines(content.Lines), "connected") {
		t.Fatalf("connections list and details should use configured status icons, lines=%#v", content.Lines)
	}
}

func TestWorkbenchNavigatorUsesEndpointStatusIcons(t *testing.T) {
	statusConfig := state.TUIPickerEndpointStatusConfig{
		Connected: state.TUIEndpointStatusAppearanceConfig{Glyph: "+", Style: "success"},
		Offline:   state.TUIEndpointStatusAppearanceConfig{Glyph: "!", Style: "danger"},
	}
	root := state.Root{Config: state.TUIConfigStore{Chrome: state.TUIChromeConfig{Picker: state.TUIPickerChromeConfig{
		EndpointStatus: statusConfig,
	}}}}

	connected := state.WorkbenchTreeItem{
		Kind: state.WorkbenchTreeKindPane, EndpointID: "studio", EndpointLabel: "Studio", TerminalID: "term-1",
		EndpointStatus: state.EndpointStatusConnected,
	}
	detail := workbenchNavigatorEndpointLine(root, connected, state.TerminalPoolPageItem{})
	if !lineHasStyledCell(detail, "+", StyleSuccess) || strings.Contains(detail.PlainString(), "connected") {
		t.Fatalf("workbench endpoint detail should use the configured status icon, got %#v", detail)
	}

	offline := connected
	offline.EndpointStatus = state.EndpointStatusOffline
	badges := workbenchEndpointBadges(root, offline)
	if len(badges) != 1 || badges[0].Text != "!" || badges[0].Style != StyleDanger {
		t.Fatalf("workbench endpoint badge should use the configured status icon, got %#v", badges)
	}
	meta := workbenchTreeInlineMetaCells(offline, statusConfig)
	if len(meta) != 1 || meta[0].Text != "!" || meta[0].Style != StyleDanger {
		t.Fatalf("workbench inline endpoint state should use the configured status icon, got %#v", meta)
	}
}

func TestConnectionsEntryIsDiscoverableInNarrowSystemFooterAndHelp(t *testing.T) {
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 56, Rows: 24},
		Shell:    state.DefaultShell().SetInteractionMode(state.InteractionModeGlobal),
	}
	frame := NewRenderer(DefaultTheme()).Render(NewRenderVMBuilder().Build(root))
	if text := frameText(frame); !strings.Contains(text, "CONNECTIONS") {
		t.Fatalf("56-column system footer hides Connections entry:\n%s", text)
	}
	root.Shell = state.DefaultShell().OpenHelp("most-used")
	entries := input.ShortcutEntriesForHelp(root.Config.Shortcuts, root.HostCapabilities.KeyboardDisambiguation)
	for index, entry := range entries {
		if entry.ActionID == "system.open_connections" {
			root.Shell = root.Shell.SetHelpSelection(index, len(entries))
			break
		}
	}
	help := plainLines(NewRenderVMBuilder().Build(root).Shell.Overlay.Content.Lines)
	if !strings.Contains(help, "Shell") || !strings.Contains(help, "[e]") || !strings.Contains(help, "CONNECTIONS") {
		t.Fatalf("help does not expose Connections entry: %q", help)
	}
}

func frameText(frame Frame) string {
	lines := make([]string, 0, len(frame.Lines))
	for _, line := range frame.Lines {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
