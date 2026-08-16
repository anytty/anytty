package render

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/state"
)

func TestFooterInformationAndActionsDeriveFromFocusedFloating(t *testing.T) {
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-tiled")
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "cloud", Kind: state.PaneTerminalLive, TerminalID: "term-floating"},
		Rect:     state.FloatingRect{X: 8, Y: 4, W: 42, H: 10},
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating fixture: %#v", result)
	}
	shell = shell.SetInteractionMode(state.InteractionModePane)
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-tiled", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-tiled", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindFloating(state.NewFloatingTerminalView("floating-1", "floating-pane-1", "term-floating", 8, 40, 8, state.TerminalResizeRoleFollower, "surface-floating", state.TerminalFloatingViewID("floating-1"), false))

	footer := NewRenderVMBuilder().Build(root).Shell.Footer

	if footer.ActiveTarget != "float:cloud attached" {
		t.Fatalf("footer target information must come from the focused floating, got %q", footer.ActiveTarget)
	}
	for _, id := range []string{"panel.close", "panel.detach", "panel.reconnect", "panel.restart", "panel.take_owner", "panel.size_lock", "panel.kill"} {
		if !containsFooterActionID(footer.ActionTokens, id) {
			t.Fatalf("focused floating must expose supported action %s, actions=%#v", id, footer.ActionTokens)
		}
	}
	for _, id := range []string{"panel.split_right", "panel.split_down", "panel.toggle_zoom", "panel.balance", "panel.presentation_card", "panel.presentation_split_line", "panel.focus_next", "panel.focus_prev"} {
		if containsFooterActionID(footer.ActionTokens, id) {
			t.Fatalf("focused floating must hide tiled-only action %s, actions=%#v", id, footer.ActionTokens)
		}
	}
}

func TestResizeFooterProjectsCompleteFloatingPositionActions(t *testing.T) {
	shell := state.DefaultShell()
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action: state.FloatingCommandCreate, TargetID: "floating-1",
		Pane: state.PaneState{ID: "floating-pane-1", Title: "cloud", Kind: state.PaneEmpty},
	})
	shell = shell.SetInteractionMode(state.InteractionModeResize)
	root := state.Root{Shell: shell, Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 24}}
	footer := NewRenderVMBuilder().Build(root).Shell.Footer
	for _, want := range []struct {
		key   string
		label string
		id    string
	}{
		{key: "←↑→↓", label: "MOVE", id: "floating.position"},
		{key: "0$^B", label: "ALIGN", id: "resize.align_left"},
		{key: "m|_", label: "CENTER", id: "resize.center"},
		{key: "a", label: "OWNER", id: "panel.take_owner"},
		{key: "s", label: "LOCK", id: "panel.size_lock"},
		{key: "r", label: "RESET", id: "resize.layout_reset"},
	} {
		if !containsFooterAction(footer.ActionTokens, want.key, want.label, want.id) {
			t.Fatalf("focused floating resize footer must expose %#v, actions=%#v", want, footer.ActionTokens)
		}
	}
	for _, id := range []string{"resize.layout_toggle", "panel.balance"} {
		if containsFooterActionID(footer.ActionTokens, id) {
			t.Fatalf("focused floating position footer must hide unsupported action %s, actions=%#v", id, footer.ActionTokens)
		}
	}
	frame := NewRenderer(DefaultTheme()).Render(NewRenderVMBuilder().Build(root))
	footerLine := frame.Lines[len(frame.Lines)-1]
	for _, want := range []string{"[←↑→↓] MOVE", "[0$^B] ALIGN", "[m|_] CENTER"} {
		if !strings.Contains(footerLine, want) {
			t.Fatalf("80-column floating position footer must render %q, line=%q", want, footerLine)
		}
	}
	root.Viewport.Cols = 48
	narrowFrame := NewRenderer(DefaultTheme()).Render(NewRenderVMBuilder().Build(root))
	narrowFooter := narrowFrame.Lines[len(narrowFrame.Lines)-1]
	for _, want := range []string{"[←↑→↓]", "[0$^B]", "[m|_]", "[a]", "[s]", "[r]"} {
		if !strings.Contains(narrowFooter, want) {
			t.Fatalf("48-column floating position footer must keep key token %q, line=%q", want, narrowFooter)
		}
	}
}

func TestFooterUsesActionColorsWhenKeyTemplateIsHidden(t *testing.T) {
	actions := []FooterActionVM{
		{Key: "^P", Label: "P  PANE", Style: StyleFooterKeyPane},
		{Key: "^R", Label: "R 󰙖 SIZE", Style: StyleFooterKeyResize},
		{Key: "^T", Label: "T 󰓩 TAB", Style: StyleFooterKeyTab},
	}
	segments := appendFooterActionSegments(nil, actions, 100, "", defaultFooterActionTemplate, " · ")

	for _, want := range []struct {
		text  string
		style StyleToken
	}{
		{text: "P  PANE", style: StyleFooterKeyPane},
		{text: "R 󰙖 SIZE", style: StyleFooterKeyResize},
		{text: "T 󰓩 TAB", style: StyleFooterKeyTab},
	} {
		found := false
		for _, segment := range segments {
			if strings.Contains(segment.text, want.text) && segment.style == want.style {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("hidden-key footer must render %q with %s, segments=%#v", want.text, want.style, segments)
		}
	}
}
