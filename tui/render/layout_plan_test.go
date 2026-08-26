package render

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	actiondomain "github.com/anytty/anytty/tui/action"
	tuiconfig "github.com/anytty/anytty/tui/config"
)

func TestMeasureLayoutPlansBodyPanelOverlayAndToastRects(t *testing.T) {
	shell := ShellVM{
		Header: HeaderVM{Visible: true, Title: "main"},
		Footer: FooterVM{Visible: true, Mode: "live"},
		Layout: LayoutVM{
			Panels: []PanelVM{{
				ID:           "pane-1",
				Title:        "shell",
				Presentation: PanelPresentationCard,
				Active:       true,
			}},
		},
		Overlay: OverlayVM{Kind: OverlayTerminalPicker, Content: ContentVM{Kind: ContentTerminalPicker}},
		Toasts:  []ToastVM{{ID: "toast-1", Title: "notice"}},
	}

	plan := MeasureLayout(shell, Rect{W: 50, H: 14})
	if plan.Viewport != (Rect{W: 50, H: 14}) {
		t.Fatalf("unexpected viewport plan %#v", plan.Viewport)
	}
	if plan.Header != (Rect{X: 0, Y: 0, W: 50, H: 1}) {
		t.Fatalf("unexpected header rect %#v", plan.Header)
	}
	if plan.Footer != (Rect{X: 0, Y: 13, W: 50, H: 1}) {
		t.Fatalf("unexpected footer rect %#v", plan.Footer)
	}
	if plan.Body != (Rect{X: 0, Y: 1, W: 50, H: 12}) {
		t.Fatalf("unexpected body rect %#v", plan.Body)
	}
	if len(plan.Panels) != 1 || plan.Panels[0].Rect != plan.Body {
		t.Fatalf("expected panel to occupy body, got %#v", plan.Panels)
	}
	if want := (Rect{X: 1, Y: 2, W: 48, H: 10}); plan.Panels[0].ContentRect != want {
		t.Fatalf("unexpected card content rect got=%#v want=%#v", plan.Panels[0].ContentRect, want)
	}
	if plan.Overlay.W == 0 || plan.Overlay.H == 0 {
		t.Fatalf("expected overlay rect, got %#v", plan.Overlay)
	}
	if plan.OverlayContentRect.W > plan.Overlay.W || plan.OverlayContentRect.H > plan.Overlay.H {
		t.Fatalf("overlay content must stay inside overlay rect overlay=%#v content=%#v", plan.Overlay, plan.OverlayContentRect)
	}
	if len(plan.Toasts) != 1 || plan.Toasts[0] != (Rect{X: 6, Y: 3, W: 42, H: 5}) {
		t.Fatalf("unexpected toast rects %#v", plan.Toasts)
	}
}

func TestMeasureLayoutAlignsPromptWithTerminalPickerOverlay(t *testing.T) {
	promptShell := ShellVM{Overlay: OverlayVM{Kind: OverlayPrompt, Content: ContentVM{Kind: ContentPrompt, Lines: []Line{NewLine("Create Terminal"), NewLine("name*: shell")}}}}
	pickerShell := ShellVM{Overlay: OverlayVM{Kind: OverlayTerminalPicker, Content: ContentVM{Kind: ContentTerminalPicker, Lines: []Line{NewLine("terminal picker"), NewLine("search:")}}}}
	viewport := Rect{W: 80, H: 24}

	prompt := MeasureLayout(promptShell, viewport)
	picker := MeasureLayout(pickerShell, viewport)
	if prompt.Overlay.X != picker.Overlay.X || prompt.Overlay.W != picker.Overlay.W || prompt.Overlay.H > picker.Overlay.H {
		t.Fatalf("prompt should use compact modal geometry aligned with picker, prompt=%#v picker=%#v", prompt.Overlay, picker.Overlay)
	}
}

func TestMeasureLayoutClipboardHistoryUsesOuterViewportWidth(t *testing.T) {
	shell := ShellVM{Overlay: OverlayVM{Kind: OverlayClipboardHistory, Content: ContentVM{Kind: ContentClipboardHistory, Lines: []Line{NewLine("Search"), NewLine(""), NewLine("› git commit│git commit -m fix terminal")}}}}

	plan := MeasureLayout(shell, Rect{W: 160, H: 40})
	if plan.Overlay.W < 120 || plan.Overlay.W > 136 || plan.Overlay.H < 28 || plan.Overlay.H > 36 {
		t.Fatalf("clipboard history should expand into a large quick panel, overlay=%#v content=%#v", plan.Overlay, plan.OverlayContentRect)
	}
	if plan.OverlayContentRect.W != plan.Overlay.W-2 || plan.OverlayContentRect.H != plan.Overlay.H-2 {
		t.Fatalf("clipboard history should keep thin one-cell modal border, overlay=%#v content=%#v", plan.Overlay, plan.OverlayContentRect)
	}
	if plan.Overlay.X <= 0 || plan.Overlay.Y <= 0 {
		t.Fatalf("clipboard history should stay centered with terminal margin, overlay=%#v", plan.Overlay)
	}
}

func TestMeasureLayoutVisibleHeaderFooterReserveChromeSafeOverlayAndFloatingArea(t *testing.T) {
	shell := ShellVM{
		Header: HeaderVM{Visible: true, Title: "main"},
		Footer: FooterVM{Visible: true, Mode: "live"},
		Layout: LayoutVM{
			Floating: []FloatingVM{{
				ID:      "float-1",
				Title:   "float",
				Rect:    Rect{X: 2, Y: 0, W: 20, H: 24},
				Content: ContentVM{Kind: ContentEmptyPane},
			}},
		},
		Overlay: OverlayVM{
			Kind:   OverlayTerminalPool,
			Opaque: true,
			Content: ContentVM{
				Kind:   ContentTerminalPool,
				Cursor: Cursor{Visible: true, Row: 0, Col: 4, Shape: CursorShapeBar},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 24})
	if plan.Body != (Rect{X: 0, Y: 1, W: 80, H: 22}) {
		t.Fatalf("body should reserve visible header/footer, got %#v", plan.Body)
	}
	if plan.Overlay != plan.Body {
		t.Fatalf("opaque manager overlay should stay inside body, overlay=%#v body=%#v", plan.Overlay, plan.Body)
	}
	if got := plan.CursorRect; got.X != plan.OverlayContentRect.X+4 || got.Y != plan.OverlayContentRect.Y {
		t.Fatalf("overlay cursor should be relative to body content rect, content=%#v cursor=%#v", plan.OverlayContentRect, got)
	}
	if len(plan.Floatings) != 1 || plan.Floatings[0].Rect.Y < plan.Body.Y || plan.Floatings[0].Rect.Y+plan.Floatings[0].Rect.H > plan.Body.Y+plan.Body.H {
		t.Fatalf("floating should be moved inside chrome-safe body, body=%#v floatings=%#v", plan.Body, plan.Floatings)
	}
	if len(plan.HitRegions) < 3 || plan.HitRegions[0].Rect.Y != plan.Header.Y || plan.HitRegions[1].Rect.Y != plan.Header.Y || plan.HitRegions[2].Kind != HitRegionOverlay {
		t.Fatalf("visible header/footer hit regions should stay above opaque overlay, got %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutHiddenHeaderFooterAllowManagerOverlayFullViewport(t *testing.T) {
	shell := ShellVM{
		Overlay: OverlayVM{
			Kind:    OverlayTerminalPool,
			Opaque:  true,
			Content: ContentVM{Kind: ContentTerminalPool},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 24})
	if plan.Header.H != 0 || plan.Footer.H != 0 || plan.Overlay != (Rect{W: 80, H: 24}) {
		t.Fatalf("hidden chrome should allow full viewport manager overlay, header=%#v footer=%#v overlay=%#v", plan.Header, plan.Footer, plan.Overlay)
	}
}

func TestMeasureLayoutUsesKnownNarrowViewportExactly(t *testing.T) {
	shell := ShellVM{
		Header: HeaderVM{Visible: true, Title: "main"},
		Footer: FooterVM{Visible: true, Mode: "live"},
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
		}}},
	}

	plan := MeasureLayout(shell, Rect{W: 12, H: 6})
	if plan.Viewport != (Rect{W: 12, H: 6}) {
		t.Fatalf("known viewport must not be expanded, got %#v", plan.Viewport)
	}
	if plan.Body != (Rect{X: 0, Y: 1, W: 12, H: 4}) {
		t.Fatalf("unexpected narrow body rect %#v", plan.Body)
	}
	if got, want := plan.Panels[0].ContentRect, (Rect{X: 1, Y: 2, W: 10, H: 2}); got != want {
		t.Fatalf("unexpected narrow content rect got=%#v want=%#v", got, want)
	}
}

func TestMeasureLayoutSplitsPanelsFromPurePlan(t *testing.T) {
	panels := []PanelVM{
		{ID: "left", Presentation: PanelPresentationSplitLine},
		{ID: "right", Presentation: PanelPresentationSplitLine, Active: true},
	}
	shell := ShellVM{
		Layout: LayoutVM{
			Panels: panels,
			Split:  SplitVM{Direction: SplitVertical, Children: []SplitVM{{PaneID: "left"}, {PaneID: "right"}}},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 41, H: 10})
	if len(plan.Panels) != 2 {
		t.Fatalf("expected two panel plans, got %#v", plan.Panels)
	}
	if got, want := plan.Panels[0].Rect, (Rect{X: 0, Y: 0, W: 20, H: 10}); got != want {
		t.Fatalf("unexpected left split rect got=%#v want=%#v", got, want)
	}
	if got, want := plan.Panels[1].Rect, (Rect{X: 20, Y: 0, W: 21, H: 10}); got != want {
		t.Fatalf("unexpected right split rect got=%#v want=%#v", got, want)
	}
	if got, want := plan.Panels[0].ContentRect, (Rect{X: 1, Y: 1, W: 19, H: 8}); got != want {
		t.Fatalf("unexpected left split content rect got=%#v want=%#v", got, want)
	}
	if got, want := plan.Panels[1].ContentRect, (Rect{X: 21, Y: 1, W: 19, H: 8}); got != want {
		t.Fatalf("unexpected split content rect got=%#v want=%#v", got, want)
	}
}

func TestMeasureLayoutUsesSplitSizeHints(t *testing.T) {
	panels := []PanelVM{
		{ID: "left", Presentation: PanelPresentationSplitLine},
		{ID: "right", Presentation: PanelPresentationSplitLine, Active: true},
	}
	shell := ShellVM{Layout: LayoutVM{
		Panels: panels,
		Split:  SplitVM{Direction: SplitVertical, Ratio: 0.25, Children: []SplitVM{{PaneID: "left"}, {PaneID: "right"}}},
	}}
	plan := MeasureLayout(shell, Rect{W: 40, H: 10})
	if got, want := plan.Panels[0].Rect.W, 10; got != want {
		t.Fatalf("ratio split should allocate left width got=%d want=%d", got, want)
	}

	shell.Layout.Split = SplitVM{Direction: SplitVertical, BiasCells: 5, Children: []SplitVM{{PaneID: "left"}, {PaneID: "right"}}}
	plan = MeasureLayout(shell, Rect{W: 40, H: 10})
	if got, want := plan.Panels[0].Rect.W, 25; got != want {
		t.Fatalf("bias split should adjust left width got=%d want=%d", got, want)
	}

	shell.Layout.Split = SplitVM{Direction: SplitVertical, FixedPaneID: "right", FixedCols: 12, Children: []SplitVM{{PaneID: "left"}, {PaneID: "right"}}}
	plan = MeasureLayout(shell, Rect{W: 40, H: 10})
	if got, want := plan.Panels[1].Rect.W, 12; got != want {
		t.Fatalf("fixed split should allocate right width got=%d want=%d", got, want)
	}
}

func TestMeasureLayoutHeaderFooterHideReclaimsBodyInPurePlan(t *testing.T) {
	shell := ShellVM{
		Header: HeaderVM{Visible: true},
		Footer: FooterVM{Visible: true},
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
		}}},
	}

	visible := MeasureLayout(shell, Rect{W: 30, H: 10})
	shell.Header.Visible = false
	shell.Footer.Visible = false
	hidden := MeasureLayout(shell, Rect{W: 30, H: 10})

	if hidden.Body.H <= visible.Body.H || hidden.Panels[0].ContentRect.H <= visible.Panels[0].ContentRect.H {
		t.Fatalf("expected hidden chrome to reclaim body visible=%#v hidden=%#v", visible, hidden)
	}
}

func TestMeasureLayoutProducesGlobalHitRegionsAndCursorRect(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content: ContentVM{
				Kind:       ContentCopyHistory,
				Cursor:     Cursor{Visible: true, Row: 1, Col: 2, Shape: CursorShapeBlock},
				HitRegions: []HitRegion{{Kind: HitRegionHistoryRow, Rect: Rect{X: 1, Y: 0, W: 5, H: 1}, LineID: 42}},
			},
		}}},
	}

	plan := MeasureLayout(shell, Rect{W: 30, H: 10})
	var contentRegion HitRegion
	for _, region := range plan.HitRegions {
		if region.Kind == HitRegionHistoryRow {
			contentRegion = region
			break
		}
	}
	if contentRegion.Kind != HitRegionHistoryRow {
		t.Fatalf("expected content hit region, got %#v", plan.HitRegions)
	}
	if got, want := contentRegion.Rect, (Rect{X: 2, Y: 1, W: 5, H: 1}); got != want {
		t.Fatalf("unexpected global hit region got=%#v want=%#v", got, want)
	}
	if !plan.Cursor.Visible || plan.Cursor.Row != 1 || plan.Cursor.Col != 2 {
		t.Fatalf("expected content-local cursor passthrough, got %#v", plan.Cursor)
	}
	if got, want := plan.CursorRect, (Rect{X: 3, Y: 2, W: 1, H: 1}); got != want {
		t.Fatalf("unexpected global cursor rect got=%#v want=%#v", got, want)
	}
}

func TestMeasureLayoutFloatingCoversUnderlyingPaneCursor(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{
			Panels: []PanelVM{{
				ID:           "pane-1",
				Presentation: PanelPresentationCard,
				Active:       true,
				Content: ContentVM{
					Kind:   ContentTerminalLive,
					Cursor: Cursor{Visible: true, Row: 3, Col: 5, Shape: CursorShapeBar},
				},
			}},
			Floating: []FloatingVM{{
				ID:      "float-1",
				PaneID:  "float-pane-1",
				Rect:    Rect{X: 4, Y: 3, W: 18, H: 5},
				Z:       1,
				Active:  false,
				Content: ContentVM{Kind: ContentTerminalLive},
			}},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 40, H: 12})
	if plan.Cursor.Visible || !plan.Cursor.Anchor {
		t.Fatalf("floating should hide covered pane cursor but keep anchor, cursor=%#v rect=%#v", plan.Cursor, plan.CursorRect)
	}
	if plan.CursorRect != (Rect{X: 6, Y: 4, W: 1, H: 1}) {
		t.Fatalf("covered cursor should retain original global anchor rect, got %#v", plan.CursorRect)
	}
}

func TestMeasureLayoutClipsContentHitRegionsToContentRect(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content: ContentVM{
				Kind:       ContentCopyHistory,
				HitRegions: []HitRegion{{Kind: HitRegionHistoryRow, Rect: Rect{X: 2, Y: 0, W: 40, H: 1}, LineID: 42}},
			},
		}}},
	}

	plan := MeasureLayout(shell, Rect{W: 12, H: 6})
	var contentRegion HitRegion
	for _, region := range plan.HitRegions {
		if region.Kind == HitRegionHistoryRow {
			contentRegion = region
			break
		}
	}
	if contentRegion.Kind != HitRegionHistoryRow {
		t.Fatalf("expected content hit region, got %#v", plan.HitRegions)
	}
	contentRect := plan.Panels[0].ContentRect
	if contentRegion.Rect.X < contentRect.X || contentRegion.Rect.X+contentRegion.Rect.W > contentRect.X+contentRect.W {
		t.Fatalf("content hit region must stay inside content rect region=%#v content=%#v", contentRegion, contentRect)
	}
	if got, want := contentRegion.Rect.W, contentRect.W-2; got != want {
		t.Fatalf("wide content hit region should be clipped by content rect got=%d want=%d region=%#v content=%#v", got, want, contentRegion, contentRect)
	}
}

func TestMeasureLayoutAddsHeaderTabActionHitRegions(t *testing.T) {
	shell := ShellVM{
		Header: HeaderVM{
			Visible:   true,
			Workspace: "main",
			Tabs: []HeaderTabVM{
				{ID: "tab-main", Title: "main", Index: 1, Active: true, CloseActionID: ActionTabClose.String(), CloseTargetID: "tab-main"},
				{ID: "tab-logs", Title: "logs", Index: 2, CloseActionID: ActionTabClose.String(), CloseTargetID: "tab-logs"},
			},
			ActivePane: "pane-main",
		},
		Layout: LayoutVM{Panels: []PanelVM{{ID: "pane-main", Presentation: PanelPresentationCard, Active: true}}},
	}
	plan := MeasureLayout(shell, Rect{W: 80, H: 20})
	closeRegion := hitRegionByAction(t, plan.HitRegions, ActionTabClose.String())
	switchRegion := hitRegionByAction(t, plan.HitRegions, ActionTabSwitch.String())
	createRegion := hitRegionByAction(t, plan.HitRegions, ActionTabCreate.String())
	workspaceRegion := hitRegionByAction(t, plan.HitRegions, "menu.workbench_tree")
	if workspaceRegion.Kind != HitRegionContentAction || workspaceRegion.Rect.Y != plan.Header.Y || workspaceRegion.Rect.W != DisplayWidth(" WS main") {
		t.Fatalf("unexpected workspace navigator region %#v", workspaceRegion)
	}
	if closeRegion.Kind != HitRegionContentAction || closeRegion.Rect.Y != plan.Header.Y || closeRegion.Rect.W != DisplayWidth(DefaultPaneChromeGlyphs().Close) {
		t.Fatalf("unexpected tab close region %#v", closeRegion)
	}
	if closeRegion.PaneID != "tab-main" {
		t.Fatalf("tab close hit region should carry target tab id, got %#v", closeRegion)
	}
	if switchRegion.Kind != HitRegionContentAction || switchRegion.PaneID != "tab-main" || switchRegion.Rect.Y != plan.Header.Y || switchRegion.Rect.W == 0 {
		t.Fatalf("unexpected tab switch region %#v", switchRegion)
	}
	if createRegion.Kind != HitRegionContentAction || createRegion.Rect.Y != plan.Header.Y || createRegion.Rect.W != DisplayWidth(HeaderTabCreateText) {
		t.Fatalf("unexpected tab create region %#v", createRegion)
	}
	if closeRegion.Rect.X >= createRegion.Rect.X {
		t.Fatalf("tab close should appear before create, close=%#v create=%#v", closeRegion, createRegion)
	}
}

func TestMeasureLayoutPortableHeaderHitsStayAlignedAtNarrowUnicodeWidths(t *testing.T) {
	for _, viewportWidth := range []int{40, 80} {
		t.Run(strconv.Itoa(viewportWidth), func(t *testing.T) {
			shell := ShellVM{
				Header: HeaderVM{
					Visible:           true,
					Workspace:         "工作区🚀",
					WorkspaceTemplate: tuiconfig.DefaultWorkspaceTemplate,
					TabTemplate:       tuiconfig.DefaultTabTemplate,
					TabCreateIcon:     "+",
					Tabs: []HeaderTabVM{{
						ID: "tab-logs", Title: "日志🚀", Index: 1, Active: true,
						CloseActionID: ActionTabClose.String(), CloseTargetID: "tab-logs",
					}},
				},
				Layout: LayoutVM{Panels: []PanelVM{{ID: "pane-main", Presentation: PanelPresentationCard, Active: true}}},
			}
			plan := MeasureLayout(shell, Rect{W: viewportWidth, H: 12})
			seen := map[string]bool{}
			for _, region := range plan.HitRegions {
				if region.Rect.Y != plan.Header.Y {
					continue
				}
				if region.Rect.W <= 0 || region.Rect.X < 0 || region.Rect.X+region.Rect.W > viewportWidth {
					t.Fatalf("header hit region must stay within %d columns: %#v", viewportWidth, region)
				}
				seen[region.ActionID] = true
				if region.ActionID == ActionTabClose.String() && region.Rect.W != DisplayWidth(DefaultPaneChromeGlyphs().Close) {
					t.Fatalf("close hit width must follow the built-in glyph: %#v", region)
				}
			}
			for _, actionID := range []string{"menu.workbench_tree", ActionTabSwitch.String(), ActionTabClose.String(), ActionTabCreate.String()} {
				if !seen[actionID] {
					t.Fatalf("portable header action %q missing at width %d: %#v", actionID, viewportWidth, plan.HitRegions)
				}
			}
		})
	}
}

func TestMeasureLayoutAddsVisibleFooterActionHitRegions(t *testing.T) {
	shell := ShellVM{
		Footer: FooterVM{
			Visible: true,
			Mode:    "live",
			ActionTokens: []FooterActionVM{
				{Key: "^P", Label: "PANE", ActionID: "menu.panel", Invocation: actiondomain.Invocation{ID: "menu.panel"}, Click: ClickClickable},
				{Key: "w", Label: "CLOSE"},
				{Key: "^F", Label: "PICKER", ActionID: "menu.terminal_picker", Invocation: actiondomain.Invocation{ID: "menu.terminal_picker"}, Click: ClickClickable},
			},
		},
		Layout: LayoutVM{Panels: []PanelVM{{ID: "pane-main", Presentation: PanelPresentationCard, Active: true}}},
	}
	plan := MeasureLayout(shell, Rect{W: 160, H: 20})
	paneRegion := hitRegionByAction(t, plan.HitRegions, "menu.panel")
	pickerRegion := hitRegionByAction(t, plan.HitRegions, "menu.terminal_picker")
	if paneRegion.Kind != HitRegionContentAction || paneRegion.Rect.Y != plan.Footer.Y+plan.Footer.H-1 || paneRegion.Rect.W != DisplayWidth("[Ctrl+P] PANE") {
		t.Fatalf("unexpected footer pane action region %#v footer=%#v", paneRegion, plan.Footer)
	}
	if pickerRegion.Kind != HitRegionContentAction || pickerRegion.Rect.Y != paneRegion.Rect.Y || pickerRegion.Rect.X <= paneRegion.Rect.X {
		t.Fatalf("unexpected footer picker action region %#v pane=%#v", pickerRegion, paneRegion)
	}
	if _, ok := findHitRegionByAction(plan.HitRegions, "close"); ok {
		t.Fatalf("footer token without action id must not produce hit region: %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutFooterActionHitRegionsFollowNarrowSelection(t *testing.T) {
	shell := ShellVM{
		Footer: FooterVM{
			Visible: true,
			Mode:    "live",
			ActionTokens: []FooterActionVM{
				{Key: "^P", Label: "PANE", ActionID: "footer.pane", Invocation: actiondomain.Invocation{ID: "menu.panel"}, Click: ClickClickable},
				{Key: "^R", Label: "RESIZE", ActionID: "footer.resize", Invocation: actiondomain.Invocation{ID: "menu.resize"}, Click: ClickClickable},
				{Key: "^F", Label: "PICKER", ActionID: "footer.picker", Invocation: actiondomain.Invocation{ID: "terminal_picker.open"}, Click: ClickClickable},
				{Key: "^G", Label: "GLOBAL", ActionID: "footer.global", Invocation: actiondomain.Invocation{ID: "menu.system"}, Click: ClickClickable},
			},
		},
		Layout: LayoutVM{Panels: []PanelVM{{ID: "pane-main", Presentation: PanelPresentationCard, Active: true}}},
	}
	plan := MeasureLayout(shell, Rect{W: 30, H: 12})
	if _, ok := findHitRegionByAction(plan.HitRegions, "footer.picker"); ok {
		t.Fatalf("narrow footer should not expose hidden picker hit region: %#v", plan.HitRegions)
	}
	if global, ok := findHitRegionByAction(plan.HitRegions, "footer.global"); !ok || global.Kind != HitRegionContentAction {
		t.Fatalf("narrow footer should keep visible tail action region, got %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutAddsFloatingSummaryHitRegion(t *testing.T) {
	shell := ShellVM{
		Footer: FooterVM{
			Visible:             true,
			Mode:                "live",
			ActionTokens:        []FooterActionVM{{Key: "^G", Label: "GLOBAL", ActionID: "footer.global"}},
			GlobalSummary:       "ws:main float:1 collapsed:1 terminals:1",
			FloatingSummaryOpen: true,
		},
		Layout: LayoutVM{Panels: []PanelVM{{ID: "pane-main", Presentation: PanelPresentationCard, Active: true}}},
	}
	plan := MeasureLayout(shell, Rect{W: 100, H: 20})
	region := hitRegionByAction(t, plan.HitRegions, "menu.floating_overview")
	if region.Kind != HitRegionContentAction || region.Rect.Y != plan.Footer.Y || region.Rect.W != DisplayWidth(" float:1 collapsed:1") {
		t.Fatalf("floating summary should open overview, got %#v footer=%#v", region, plan.Footer)
	}
}

func TestMeasureLayoutAnchorsCursorWhenContentHasNoVisibleCursor(t *testing.T) {
	shell := ShellVM{
		Header: HeaderVM{Visible: true, Title: "main"},
		Footer: FooterVM{Visible: true, Mode: "live"},
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content:      ContentVM{Kind: ContentPlaceholder, Lines: []Line{NewLine("pending")}},
		}}},
	}

	plan := MeasureLayout(shell, Rect{W: 40, H: 10})
	want := plan.Panels[0].ContentRect
	if plan.Cursor.Visible || !plan.Cursor.Anchor || plan.Cursor.Shape != CursorShapeBar {
		t.Fatalf("missing IME cursor anchor, cursor=%#v rect=%#v", plan.Cursor, plan.CursorRect)
	}
	if plan.CursorRect != (Rect{X: want.X, Y: want.Y, W: 1, H: 1}) {
		t.Fatalf("cursor anchor should park at active content origin, content=%#v cursor=%#v", want, plan.CursorRect)
	}
}

func TestMeasureLayoutAddsPaneCommandHitRegionsBeforeContent(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content: ContentVM{
				HitRegions: []HitRegion{{Kind: HitRegionHistoryRow, Rect: Rect{W: 10, H: 1}, LineID: 7}},
			},
		}}},
	}

	plan := MeasureLayout(shell, Rect{W: 40, H: 10})
	if len(plan.HitRegions) < 5 {
		t.Fatalf("expected pane command and content hit regions, got %#v", plan.HitRegions)
	}
	if plan.HitRegions[0].Kind != HitRegionPaneAction || plan.HitRegions[0].PaneID != "pane-1" || plan.HitRegions[0].ActionID != ActionPaneZoom.String() {
		t.Fatalf("pane action region should be first, got %#v", plan.HitRegions)
	}
	if plan.HitRegions[1].Kind != HitRegionPaneAction || plan.HitRegions[1].ActionID != "pane.split-right" ||
		plan.HitRegions[2].Kind != HitRegionPaneAction || plan.HitRegions[2].ActionID != "pane.split-down" ||
		plan.HitRegions[3].Kind != HitRegionPaneAction || plan.HitRegions[3].ActionID != "pane.close" {
		t.Fatalf("pane action regions should expose visible zoom/split/close tokens, got %#v", plan.HitRegions[:4])
	}
	actionWidth := plan.HitRegions[0].Rect.W + plan.HitRegions[1].Rect.W + plan.HitRegions[2].Rect.W + plan.HitRegions[3].Rect.W + 3
	if got, want := actionWidth, DisplayWidth(paneChromeActionText(40)); got != want {
		t.Fatalf("pane action regions should cover visible action cluster got=%d want=%d regions=%#v", got, want, plan.HitRegions[:4])
	}
	if plan.HitRegions[4].Kind != HitRegionPaneChrome || plan.HitRegions[4].ActionID != "pane.focus" {
		t.Fatalf("pane chrome region should precede content, got %#v", plan.HitRegions)
	}
	historyIndex := hitRegionIndex(plan.HitRegions, HitRegionHistoryRow)
	paneContentIndex := hitRegionIndex(plan.HitRegions, HitRegionPaneContent)
	resizeIndex := hitRegionIndex(plan.HitRegions, HitRegionPaneResize)
	if resizeIndex >= 0 {
		t.Fatalf("single pane outer border is fixed and must not expose resize handle, got %#v", plan.HitRegions)
	}
	if historyIndex <= 4 {
		t.Fatalf("content hit region should remain after chrome regions, got %#v", plan.HitRegions)
	}
	if paneContentIndex <= historyIndex {
		t.Fatalf("broad pane content focus region must not cover specific content hits, got %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutZoomPaneChromeDropsSplitActionHitRegions(t *testing.T) {
	panel := PanelVM{
		ID:           "pane-zoom",
		Presentation: PanelPresentationCard,
		Active:       true,
		IsZoomMode:   true,
		Content:      ContentVM{Kind: ContentTerminalLive},
		Chrome: PanelChromeVM{
			Title:   ChromeSlotVM{Text: "zoomed"},
			Actions: defaultPaneChromeActionVMsForZoom(StyleAccent, true),
		},
	}
	plan := MeasureLayout(ShellVM{Layout: LayoutVM{Panels: []PanelVM{panel}}}, Rect{W: 40, H: 10})
	if hitRegionIndexByAction(plan.HitRegions, ActionPaneZoom.String()) < 0 || hitRegionIndexByAction(plan.HitRegions, ActionPaneClose.String()) < 0 {
		t.Fatalf("zoom pane should keep unzoom toggle and close hit regions, got %#v", plan.HitRegions)
	}
	if hitRegionIndexByAction(plan.HitRegions, ActionPaneSplitRight.String()) >= 0 || hitRegionIndexByAction(plan.HitRegions, ActionPaneSplitDown.String()) >= 0 {
		t.Fatalf("zoom pane must not expose split hit regions, got %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutPaneActionRegionsFollowStructuredVisibleSlots(t *testing.T) {
	panel := PanelVM{
		ID:           "pane-1",
		Presentation: PanelPresentationCard,
		Active:       true,
		Chrome: PanelChromeVM{Actions: []ChromeActionVM{
			{Text: "A", ActionID: ActionPaneZoom.String()},
			{Text: "B", ActionID: ActionPaneSplitRight.String()},
			{Text: paneChromeCloseActionText(), ActionID: ActionPaneClose.String()},
		}, State: ChromeSlotVM{Text: "● active"}, Meta: []ChromeSlotVM{{Text: "80x24"}}},
	}
	wide := MeasureLayout(ShellVM{Layout: LayoutVM{Panels: []PanelVM{panel}}}, Rect{W: 20, H: 8})
	if wide.HitRegions[0].Kind != HitRegionPaneAction || wide.HitRegions[0].ActionID != ActionPaneZoom.String() ||
		wide.HitRegions[1].ActionID != ActionPaneSplitRight.String() || wide.HitRegions[2].ActionID != ActionPaneClose.String() {
		t.Fatalf("wide pane should expose structured visible action regions, got %#v", wide.HitRegions[:3])
	}
	if got, want := wide.HitRegions[0].Rect.W+wide.HitRegions[1].Rect.W+wide.HitRegions[2].Rect.W+2, paneChromeActionItemsWidth(visiblePaneChromeActionItems(panel, 20)); got != want {
		t.Fatalf("wide pane action regions should match visible slots got=%d want=%d regions=%#v", got, want, wide.HitRegions[:3])
	}

	narrow := MeasureLayout(ShellVM{Layout: LayoutVM{Panels: []PanelVM{panel}}}, Rect{W: 8, H: 8})
	if narrow.HitRegions[0].Kind != HitRegionPaneAction || narrow.HitRegions[0].ActionID != ActionPaneClose.String() {
		t.Fatalf("narrow pane should degrade to close action region, got %#v", narrow.HitRegions)
	}
	if len(narrow.HitRegions) > 1 && narrow.HitRegions[1].Kind == HitRegionPaneAction {
		t.Fatalf("narrow pane should not expose hidden custom actions, got %#v", narrow.HitRegions)
	}
}

func TestMeasureLayoutPaneActionRegionsSkipConfiguredGroupCaps(t *testing.T) {
	ResetPaneChromeGlyphs()
	defer ResetPaneChromeGlyphs()
	SetPaneChromeGlyphs(PaneChromeGlyphs{
		ActionLeft:          "",
		ActionLeftSet:       true,
		ActionRight:         "",
		ActionRightSet:      true,
		ActionSeparator:     "",
		ActionSeparatorSet:  true,
		ActionGroupLeft:     "",
		ActionGroupLeftSet:  true,
		ActionGroupRight:    "",
		ActionGroupRightSet: true,
		Close:               "x",
	})

	panel := PanelVM{
		ID:           "pane-1",
		Presentation: PanelPresentationCard,
		Active:       true,
		Chrome: PanelChromeVM{Actions: []ChromeActionVM{
			{Text: "A", ActionID: ActionPaneZoom.String()},
			{Text: "B", ActionID: ActionPaneSplitRight.String()},
			{Text: paneChromeCloseActionText(), ActionID: ActionPaneClose.String()},
		}},
	}
	rect := Rect{W: 20, H: 8}
	plan := MeasureLayout(ShellVM{Layout: LayoutVM{Panels: []PanelVM{panel}}}, rect)
	if len(plan.HitRegions) < 3 {
		t.Fatalf("expected action regions, got %#v", plan.HitRegions)
	}
	actionRect := paneActionRect(panel, rect)
	if got, want := plan.HitRegions[0].Rect.X, actionRect.X+DisplayWidth(""); got != want {
		t.Fatalf("first action should start after group left cap got=%d want=%d actionRect=%#v regions=%#v", got, want, actionRect, plan.HitRegions[:3])
	}
	actionTokenWidth := plan.HitRegions[0].Rect.W + plan.HitRegions[1].Rect.W + plan.HitRegions[2].Rect.W
	if got, want := actionTokenWidth, paneChromeActionItemsWidth(visiblePaneChromeActionItems(panel, rect.W))-DisplayWidth("")-DisplayWidth(""); got != want {
		t.Fatalf("action regions should exclude group caps got=%d want=%d regions=%#v", got, want, plan.HitRegions[:3])
	}
}

func TestMeasureLayoutAddsSplitDividerResizeHitRegions(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{
			Panels: []PanelVM{
				{ID: "left", Presentation: PanelPresentationSplitLine},
				{ID: "right", Presentation: PanelPresentationSplitLine, Active: true},
			},
			Split: SplitVM{Direction: SplitVertical, Children: []SplitVM{{PaneID: "left"}, {PaneID: "right"}}},
		},
	}
	plan := MeasureLayout(shell, Rect{W: 40, H: 10})
	region := hitRegionByActionAndPane(t, plan.HitRegions, "pane.resize", "left")
	if region.Direction != "right" || region.SplitPath != "root" || region.Rect != (Rect{X: 20, Y: 0, W: 1, H: 10}) {
		t.Fatalf("expected vertical divider resize region, got %#v", region)
	}

	shell.Layout.Split = SplitVM{Direction: SplitHorizontal, Children: []SplitVM{{PaneID: "top"}, {PaneID: "bottom"}}}
	shell.Layout.Panels = []PanelVM{
		{ID: "top", Presentation: PanelPresentationSplitLine},
		{ID: "bottom", Presentation: PanelPresentationSplitLine, Active: true},
	}
	plan = MeasureLayout(shell, Rect{W: 40, H: 10})
	region = hitRegionByActionAndPane(t, plan.HitRegions, "pane.resize", "top")
	if region.Direction != "down" || region.SplitPath != "root" || region.Rect != (Rect{X: 0, Y: 5, W: 40, H: 1}) {
		t.Fatalf("expected horizontal divider resize region, got %#v", region)
	}
}

func TestMeasureLayoutAddsNestedSplitDividerPath(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{
			Panels: []PanelVM{
				{ID: "left", Presentation: PanelPresentationSplitLine},
				{ID: "middle", Presentation: PanelPresentationSplitLine},
				{ID: "right", Presentation: PanelPresentationSplitLine, Active: true},
			},
			Split: SplitVM{
				Direction: SplitVertical,
				Children: []SplitVM{
					{PaneID: "left"},
					{
						Direction: SplitVertical,
						Children:  []SplitVM{{PaneID: "middle"}, {PaneID: "right"}},
					},
				},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 60, H: 10})
	root := hitRegionByActionAndPane(t, plan.HitRegions, "pane.resize", "left")
	nested := hitRegionByActionAndPane(t, plan.HitRegions, "pane.resize", "middle")
	if root.SplitPath != "root" || root.Rect.X != 30 {
		t.Fatalf("expected root divider path, got %#v", root)
	}
	if nested.SplitPath != "root/1" || nested.Rect.X != 45 {
		t.Fatalf("expected nested divider path, got %#v", nested)
	}
}

func TestMeasureLayoutFourColumnResizeRegionsCarryAdjacentLeafGroup(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{
			Panels: []PanelVM{
				{ID: "pane-1", Presentation: PanelPresentationSplitLine},
				{ID: "pane-2", Presentation: PanelPresentationSplitLine},
				{ID: "pane-3", Presentation: PanelPresentationSplitLine},
				{ID: "pane-4", Presentation: PanelPresentationSplitLine, Active: true},
			},
			Split: SplitVM{
				Direction: SplitVertical,
				Children: []SplitVM{
					{PaneID: "pane-1"},
					{
						Direction: SplitVertical,
						Children: []SplitVM{
							{PaneID: "pane-2"},
							{
								Direction: SplitVertical,
								Children:  []SplitVM{{PaneID: "pane-3"}, {PaneID: "pane-4"}},
							},
						},
					},
				},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 10})
	leftOfSecond := hitRegionByActionAndPane(t, plan.HitRegions, "pane.resize", "pane-1")
	rightOfSecond := hitRegionByActionAndPane(t, plan.HitRegions, "pane.resize", "pane-2")
	if leftOfSecond.ResizeBeforePaneID != "pane-1" || leftOfSecond.ResizeAfterPaneID != "pane-2" {
		t.Fatalf("left divider should target adjacent pane-1/pane-2, got %#v", leftOfSecond)
	}
	if rightOfSecond.ResizeBeforePaneID != "pane-2" || rightOfSecond.ResizeAfterPaneID != "pane-3" {
		t.Fatalf("right divider should target adjacent pane-2/pane-3, got %#v", rightOfSecond)
	}
	if got := resizeGroupSignature(rightOfSecond.ResizeGroup); got != "pane-2:20,pane-3:10,pane-4:10" {
		t.Fatalf("right subtree group should preserve pane-4 independently, got %s from %#v", got, rightOfSecond.ResizeGroup)
	}
	if got := resizeGroupSignSignature(rightOfSecond.ResizeGroup); got != "pane-2:+1,pane-3:-1,pane-4:0" {
		t.Fatalf("right subtree group should mark only divider-adjacent leaves, got %s from %#v", got, rightOfSecond.ResizeGroup)
	}
}

func TestMeasureLayoutStackedRightColumnResizeGroupMarksSharedBoundary(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{
			Panels: []PanelVM{
				{ID: "left", Presentation: PanelPresentationSplitLine},
				{ID: "top", Presentation: PanelPresentationSplitLine},
				{ID: "middle-left", Presentation: PanelPresentationSplitLine, Active: true},
				{ID: "middle-right", Presentation: PanelPresentationSplitLine},
				{ID: "bottom", Presentation: PanelPresentationSplitLine},
			},
			Split: SplitVM{
				Direction: SplitVertical,
				Children: []SplitVM{
					{PaneID: "left"},
					{
						Direction: SplitHorizontal,
						Children: []SplitVM{
							{PaneID: "top"},
							{
								Direction: SplitHorizontal,
								Children: []SplitVM{
									{
										Direction: SplitVertical,
										Children:  []SplitVM{{PaneID: "middle-left"}, {PaneID: "middle-right"}},
									},
									{PaneID: "bottom"},
								},
							},
						},
					},
				},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 18})
	divider := hitRegionByActionAndPane(t, plan.HitRegions, "pane.resize", "left")
	if divider.ResizeBeforePaneID != "left" || divider.ResizeAfterPaneID != "top" {
		t.Fatalf("root divider should target visible left/right-column neighbors, got %#v", divider)
	}
	if got := resizeGroupSignature(divider.ResizeGroup); got != "left:40,top:40,middle-left:20,middle-right:20,bottom:40" {
		t.Fatalf("stacked right column group should carry current widths, got %s from %#v", got, divider.ResizeGroup)
	}
	if got := resizeGroupSignSignature(divider.ResizeGroup); got != "left:+1,top:-1,middle-left:-1,middle-right:0,bottom:-1" {
		t.Fatalf("only panes sharing the dragged boundary should change width, got %s from %#v", got, divider.ResizeGroup)
	}
}

func TestMeasureLayoutKeepsSplitActionsAboveDividerResize(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{
			Panels: []PanelVM{
				{ID: "top", Presentation: PanelPresentationSplitLine, Active: true},
				{ID: "bottom", Presentation: PanelPresentationSplitLine},
			},
			Split: SplitVM{
				Direction: SplitHorizontal,
				Children:  []SplitVM{{PaneID: "top"}, {PaneID: "bottom"}},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 24})
	bottomSplit := hitRegionIndexByPaneAndAction(plan.HitRegions, HitRegionPaneAction, "bottom", "pane.split-down")
	divider := hitRegionIndexByActionAndDirection(plan.HitRegions, "pane.resize", "down")
	if bottomSplit < 0 || divider < 0 {
		t.Fatalf("expected bottom split action and divider resize regions, got %#v", plan.HitRegions)
	}
	if bottomSplit > divider {
		t.Fatalf("visible bottom pane action must win over shared divider resize, action=%d divider=%d regions=%#v", bottomSplit, divider, plan.HitRegions)
	}
	if !rectsOverlap(plan.HitRegions[bottomSplit].Rect, plan.HitRegions[divider].Rect) {
		t.Fatalf("test precondition should cover action on divider row, action=%#v divider=%#v", plan.HitRegions[bottomSplit], plan.HitRegions[divider])
	}
}

func hitRegionIndex(regions []HitRegion, kind HitRegionKind) int {
	for i, region := range regions {
		if region.Kind == kind {
			return i
		}
	}
	return -1
}

func TestMeasureLayoutOpaqueOverlayOwnsHitRegionsAndCursorRect(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content: ContentVM{
				Kind:       ContentCopyHistory,
				HitRegions: []HitRegion{{Kind: HitRegionHistoryRow, Rect: Rect{W: 10, H: 1}, LineID: 7}},
			},
		}}},
		Overlay: OverlayVM{
			Kind:   OverlayPrompt,
			Opaque: true,
			Content: ContentVM{
				Kind:       ContentPrompt,
				Cursor:     Cursor{Visible: true, Row: 1, Col: 2, Shape: CursorShapeBar},
				HitRegions: []HitRegion{{Kind: HitRegionStatus, Rect: Rect{W: 4, H: 1}}},
			},
		},
		Toasts: []ToastVM{{ID: "toast-1", Title: "notice"}},
	}

	plan := MeasureLayout(shell, Rect{W: 40, H: 12})
	if !plan.Cursor.Visible || plan.Cursor.Shape != CursorShapeBar {
		t.Fatalf("expected overlay cursor ownership, got %#v", plan.Cursor)
	}
	if plan.CursorRect.X != plan.OverlayContentRect.X+2 || plan.CursorRect.Y != plan.OverlayContentRect.Y+1 {
		t.Fatalf("unexpected overlay cursor rect content=%#v cursor=%#v", plan.OverlayContentRect, plan.CursorRect)
	}
	for _, region := range plan.HitRegions {
		if region.LineID == 7 {
			t.Fatalf("opaque overlay must hide body hit regions, got %#v", plan.HitRegions)
		}
	}
	if len(plan.HitRegions) < 3 || plan.HitRegions[0].Kind != HitRegionToast || plan.HitRegions[1].Kind != HitRegionStatus || plan.HitRegions[2].Kind != HitRegionOverlay {
		t.Fatalf("expected toast, overlay content, overlay hit priority, got %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutTerminalPickerOwnsCursorAndActionHits(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content:      ContentVM{Kind: ContentTerminalLive, Cursor: Cursor{Visible: true, Row: 2, Col: 3}},
		}}},
		Overlay: OverlayVM{
			Kind: OverlayTerminalPicker,
			Content: ContentVM{
				Kind:   ContentTerminalPicker,
				Cursor: Cursor{Visible: true, Row: 0, Col: 7, Shape: CursorShapeBar},
				HitRegions: []HitRegion{{
					Kind:     HitRegionContentAction,
					Rect:     Rect{Y: 1, W: 20, H: 1},
					ActionID: "picker.attach",
					Invocation: actiondomain.Invocation{ID: "terminal_picker.attach",
						SourceActionID: "terminal_picker.attach"},
					TargetMode: HitTargetExplicit,
					PaneID:     "pane-1",
				}},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 24})
	if plan.Overlay.W > 80 || plan.Overlay.H > 12 {
		t.Fatalf("terminal picker should use compact overlay, overlay=%#v", plan.Overlay)
	}
	if plan.OverlayContentRect.X-plan.Overlay.X < 4 || plan.OverlayContentRect.Y-plan.Overlay.Y < 2 {
		t.Fatalf("terminal picker should keep adaptive modal padding, overlay=%#v content=%#v", plan.Overlay, plan.OverlayContentRect)
	}
	if !plan.Cursor.Visible || plan.Cursor.Shape != CursorShapeBar {
		t.Fatalf("terminal picker should own cursor, got %#v", plan.Cursor)
	}
	if got := plan.CursorRect; got.X != plan.OverlayContentRect.X+7 || got.Y != plan.OverlayContentRect.Y {
		t.Fatalf("unexpected picker cursor rect content=%#v cursor=%#v", plan.OverlayContentRect, got)
	}
	if len(plan.HitRegions) < 2 || plan.HitRegions[0].Kind != HitRegionContentAction || plan.HitRegions[0].ActionID != "picker.attach" || plan.HitRegions[1].Kind != HitRegionOverlay {
		t.Fatalf("picker content action should precede overlay background, got %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutTerminalPickerShrinksPaddingOnTinyViewport(t *testing.T) {
	shell := ShellVM{
		Overlay: OverlayVM{
			Kind: OverlayTerminalPicker,
			Content: ContentVM{
				Kind:  ContentTerminalPicker,
				Lines: []Line{NewLine("search:"), NewLine("▸ + new terminal")},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 24, H: 6})
	if plan.Overlay.W > 24 || plan.Overlay.H > 6 {
		t.Fatalf("tiny picker overlay should stay inside viewport, overlay=%#v", plan.Overlay)
	}
	if plan.OverlayContentRect.X-plan.Overlay.X > 1 || plan.OverlayContentRect.Y-plan.Overlay.Y > 1 {
		t.Fatalf("tiny picker overlay should shrink padding, overlay=%#v content=%#v", plan.Overlay, plan.OverlayContentRect)
	}
}

func TestMeasureLayoutTerminalPickerUsesAvailableDesktopHeight(t *testing.T) {
	lines := make([]Line, 32)
	for index := range lines {
		lines[index] = NewLine(fmt.Sprintf("terminal-%02d", index+1))
	}
	shell := ShellVM{Overlay: OverlayVM{Kind: OverlayTerminalPicker, Content: ContentVM{Kind: ContentTerminalPicker, Lines: lines}}}

	plan := MeasureLayout(shell, Rect{W: 120, H: 40})
	if plan.Overlay.H != 24 {
		t.Fatalf("large terminal picker should use its expanded height cap, overlay=%#v", plan.Overlay)
	}
	if plan.OverlayContentRect.H <= 8 {
		t.Fatalf("large terminal picker should expose substantially more rows, content=%#v", plan.OverlayContentRect)
	}
}

func TestMeasureLayoutPromptSuggestionPopupEscapesModalContentRect(t *testing.T) {
	shell := ShellVM{
		Overlay: OverlayVM{
			Kind:   OverlayPrompt,
			Opaque: true,
			Content: ContentVM{
				Kind:  ContentPrompt,
				Lines: []Line{NewLine("◆ Create Terminal"), NewLine("name*: shell"), NewLine("command: codex"), NewLine("workdir: /tmp/de")},
			},
			Popup: OverlayPopupVM{
				Kind:      OverlayPopupPromptSuggestion,
				AnchorRow: 4,
				AnchorCol: 9,
				Lines: []Line{
					NewLine("  path: /tmp"),
					NewLine("    demo/"),
					NewLine("    dev/"),
					NewLine("    delta/"),
				},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 40, H: 10})
	if plan.OverlayPopup.Rect.W <= 0 || plan.OverlayPopup.Rect.H != 4 {
		t.Fatalf("expected measured prompt suggestion popup, plan=%#v", plan)
	}
	if plan.OverlayPopup.Rect.Y < 0 || plan.OverlayPopup.Rect.Y+plan.OverlayPopup.Rect.H > plan.Viewport.H {
		t.Fatalf("popup should stay visible in viewport independent of modal content rect, content=%#v popup=%#v viewport=%#v", plan.OverlayContentRect, plan.OverlayPopup.Rect, plan.Viewport)
	}
	if plan.OverlayPopup.Rect.Y >= plan.OverlayContentRect.Y &&
		plan.OverlayPopup.Rect.Y+plan.OverlayPopup.Rect.H <= plan.OverlayContentRect.Y+plan.OverlayContentRect.H {
		t.Fatalf("popup should not be clipped to modal content rect, content=%#v popup=%#v", plan.OverlayContentRect, plan.OverlayPopup.Rect)
	}
	if plan.OverlayPopup.Rect.X != plan.OverlayContentRect.X+9 {
		t.Fatalf("popup should anchor to prompt field value column, content=%#v popup=%#v", plan.OverlayContentRect, plan.OverlayPopup.Rect)
	}
}

func TestMeasureLayoutTerminalPoolUsesPageSizedOverlay(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content:      ContentVM{Kind: ContentTerminalLive, Cursor: Cursor{Visible: true, Row: 2, Col: 3}},
		}}},
		Overlay: OverlayVM{
			Kind:   OverlayTerminalPool,
			Opaque: true,
			Content: ContentVM{
				Kind:   ContentTerminalPool,
				Cursor: Cursor{Visible: true, Row: 0, Col: 9, Shape: CursorShapeBar},
				HitRegions: []HitRegion{
					{Kind: HitRegionContentAction, Rect: Rect{Y: 3, W: 40, H: 1}, ActionID: "pool.select", Invocation: actiondomain.Invocation{ID: actiondomain.ActionTerminalPoolSelect, SourceActionID: actiondomain.ActionTerminalPoolSelect.String()}, TargetMode: HitTargetExplicit},
				},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 24})
	if plan.Overlay != (Rect{W: 80, H: 24}) || plan.OverlayContentRect != (Rect{X: 1, Y: 1, W: 78, H: 22}) {
		t.Fatalf("terminal pool page must own the full viewport, overlay=%#v content=%#v", plan.Overlay, plan.OverlayContentRect)
	}
	if got := plan.CursorRect; got.X != plan.OverlayContentRect.X+9 || got.Y != plan.OverlayContentRect.Y {
		t.Fatalf("unexpected pool cursor rect content=%#v cursor=%#v", plan.OverlayContentRect, got)
	}
	for _, action := range []string{"pool.select"} {
		if hitRegionIndexByAction(plan.HitRegions, action) < 0 {
			t.Fatalf("expected visible terminal pool action %s in hit regions %#v", action, plan.HitRegions)
		}
	}
	for _, action := range []string{"pool.attach", "pool.restart", "pool.delete"} {
		if hitRegionIndexByAction(plan.HitRegions, action) >= 0 {
			t.Fatalf("terminal pool management action %s should live in footer, got %#v", action, plan.HitRegions)
		}
	}
	if hitRegionIndex(plan.HitRegions, HitRegionPaneContent) >= 0 {
		t.Fatalf("opaque terminal pool page must hide body hit regions, got %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutWorkbenchTreeUsesPageSizedOverlay(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content:      ContentVM{Kind: ContentTerminalLive, Cursor: Cursor{Visible: true, Row: 2, Col: 3}},
		}}},
		Overlay: OverlayVM{
			Kind:   OverlayWorkbenchTree,
			Opaque: true,
			Content: ContentVM{
				Kind:   ContentWorkbenchTree,
				Cursor: Cursor{Visible: true, Row: 0, Col: 10, Shape: CursorShapeBar},
				HitRegions: []HitRegion{
					{Kind: HitRegionContentAction, Rect: Rect{Y: 2, W: 72, H: 1}, ActionID: "workbench.open", Invocation: actiondomain.Invocation{ID: "workbench_tree.open", SourceActionID: "workbench_tree.open"}, TargetMode: HitTargetExplicit},
					{Kind: HitRegionContentAction, Rect: Rect{Y: 9, W: 12, H: 1}, ActionID: "workbench.open", Invocation: actiondomain.Invocation{ID: "workbench_tree.open", SourceActionID: "workbench_tree.open"}, TargetMode: HitTargetExplicit},
				},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 24})
	if plan.Overlay != (Rect{W: 80, H: 24}) || plan.OverlayContentRect != (Rect{X: 1, Y: 1, W: 78, H: 22}) {
		t.Fatalf("workbench tree must own the full viewport, overlay=%#v content=%#v", plan.Overlay, plan.OverlayContentRect)
	}
	if got := plan.CursorRect; got.X != plan.OverlayContentRect.X+10 || got.Y != plan.OverlayContentRect.Y {
		t.Fatalf("unexpected tree cursor rect content=%#v cursor=%#v", plan.OverlayContentRect, got)
	}
	if hitRegionIndexByAction(plan.HitRegions, "workbench.open") < 0 {
		t.Fatalf("expected visible workbench tree open action in hit regions %#v", plan.HitRegions)
	}
	if hitRegionIndex(plan.HitRegions, HitRegionPaneContent) >= 0 {
		t.Fatalf("opaque workbench tree must hide body hit regions, got %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutHelpUsesPageSizedOverlay(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content:      ContentVM{Kind: ContentTerminalLive, Cursor: Cursor{Visible: true, Row: 2, Col: 3}},
		}}},
		Overlay: OverlayVM{
			Kind:   OverlayHelp,
			Opaque: true,
			Content: ContentVM{
				Kind: ContentHelp,
				HitRegions: []HitRegion{
					{Kind: HitRegionContentAction, Rect: Rect{Y: 9, W: 12, H: 1}, ActionID: "help.close", Invocation: actiondomain.Invocation{ID: "help.close", SourceActionID: "help.close"}, TargetMode: HitTargetExplicit},
				},
			},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 24})
	if plan.Overlay.W < 70 || plan.Overlay.H < 14 || plan.OverlayContentRect.H < 12 {
		t.Fatalf("help must use page-sized overlay, overlay=%#v content=%#v", plan.Overlay, plan.OverlayContentRect)
	}
	if hitRegionIndexByAction(plan.HitRegions, "help.close") < 0 {
		t.Fatalf("expected help close action hit region, got %#v", plan.HitRegions)
	}
	if hitRegionIndex(plan.HitRegions, HitRegionPaneContent) >= 0 {
		t.Fatalf("opaque help page must hide body hit regions, got %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutKeepsPaneChromeActionsForEmptyPane(t *testing.T) {
	panel := PanelVM{
		ID:           "pane-empty",
		Presentation: PanelPresentationCard,
		Active:       true,
		Title:        "unconnected",
		Content: ContentVM{
			Kind:       ContentEmptyPane,
			Empty:      true,
			HitRegions: []HitRegion{{Kind: HitRegionContentAction, Rect: Rect{X: 1, Y: 2, W: 12, H: 1}, PaneID: "pane-empty", ActionID: ActionEmptyAttach.String(), Invocation: actiondomain.Invocation{ID: actiondomain.ActionEmptyAttach, SourceActionID: actiondomain.ActionEmptyAttach.String()}, TargetMode: HitTargetExplicit}},
		},
		Chrome: PanelChromeVM{Title: ChromeSlotVM{Text: "unconnected"}, Actions: defaultPaneChromeActionVMs(StyleAccent)},
	}
	plan := MeasureLayout(ShellVM{Layout: LayoutVM{Panels: []PanelVM{panel}}}, Rect{W: 40, H: 10})
	if hitRegionIndexByAction(plan.HitRegions, ActionPaneZoom.String()) < 0 || hitRegionIndexByAction(plan.HitRegions, ActionPaneSplitRight.String()) < 0 || hitRegionIndexByAction(plan.HitRegions, ActionPaneClose.String()) < 0 {
		t.Fatalf("empty pane should keep still-available pane chrome actions, got %#v", plan.HitRegions)
	}
	if hitRegionIndexByAction(plan.HitRegions, ActionEmptyAttach.String()) < 0 {
		t.Fatalf("empty pane content CTA should remain clickable, got %#v", plan.HitRegions)
	}
}

func TestMeasureLayoutFloatingHitRegionsPrecedeTiledPane(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{
			Panels: []PanelVM{{
				ID:           "pane-1",
				Presentation: PanelPresentationCard,
				Active:       true,
				Content:      ContentVM{Kind: ContentTerminalLive},
			}},
			Floating: []FloatingVM{{
				ID:      "float-1",
				PaneID:  "float-pane-1",
				Title:   "float",
				Rect:    Rect{X: 10, Y: 4, W: 30, H: 8},
				Z:       2,
				Active:  true,
				Content: ContentVM{Kind: ContentEmptyPane},
			}},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 24})
	if len(plan.Floatings) != 1 || plan.Floatings[0].ContentRect != (Rect{X: 11, Y: 5, W: 28, H: 6}) {
		t.Fatalf("expected measured floating content rect, got %#v", plan.Floatings)
	}
	center := hitRegionIndexByAction(plan.HitRegions, "floating.center")
	collapse := hitRegionIndexByAction(plan.HitRegions, "floating.collapse")
	zoom := hitRegionIndexByAction(plan.HitRegions, "pane.zoom")
	close := hitRegionIndexByAction(plan.HitRegions, "floating.close")
	move := hitRegionIndexByAction(plan.HitRegions, "floating.move-drag")
	resize := hitRegionIndexByAction(plan.HitRegions, "floating.resize-drag")
	pane := hitRegionIndex(plan.HitRegions, HitRegionPaneContent)
	if center < 0 || collapse < 0 || zoom < 0 || close < 0 || move < 0 || resize < 0 || pane < 0 || center > pane || collapse > pane || zoom > pane || close > pane || move > pane || resize > pane {
		t.Fatalf("floating hit regions should precede tiled pane regions, got %#v", plan.HitRegions)
	}
	if plan.HitRegions[center].Rect.W != DisplayWidth(paneChromeBracketToken(DefaultPaneChromeGlyphs().CenterFloating)) ||
		plan.HitRegions[collapse].Rect.W != DisplayWidth(paneChromeBracketToken(DefaultPaneChromeGlyphs().CollapseFloating)) ||
		plan.HitRegions[zoom].Rect.W != DisplayWidth(paneChromeBracketToken(paneChromeZoomGlyph())) ||
		plan.HitRegions[close].Rect.W != DisplayWidth(paneChromeBracketToken(paneChromeCloseGlyph())) ||
		plan.HitRegions[center].Rect.X >= plan.HitRegions[collapse].Rect.X ||
		plan.HitRegions[collapse].Rect.X >= plan.HitRegions[zoom].Rect.X ||
		plan.HitRegions[zoom].Rect.X >= plan.HitRegions[close].Rect.X {
		t.Fatalf("floating action hit regions should match visible bracket slots, got center=%#v collapse=%#v zoom=%#v close=%#v", plan.HitRegions[center], plan.HitRegions[collapse], plan.HitRegions[zoom], plan.HitRegions[close])
	}
	if plan.HitRegions[move].Rect != paneChromeRect(plan.Floatings[0].Rect) {
		t.Fatalf("floating title drag should cover chrome title row, got %#v", plan.HitRegions[move])
	}
	if plan.HitRegions[resize].Rect != floatingResizeRect(plan.Floatings[0].Rect) {
		t.Fatalf("floating resize drag should cover resize handle, got %#v", plan.HitRegions[resize])
	}
	if plan.HitRegions[resize].Rect.W != 3 {
		t.Fatalf("floating resize drag should expose a wider bottom-right handle, got %#v", plan.HitRegions[resize])
	}
	for _, index := range []int{center, collapse, zoom, close, move, resize} {
		region := plan.HitRegions[index]
		if region.PaneID != "float-pane-1" || !region.Floating {
			t.Fatalf("floating hit region should use floating panel id and flag, got %#v", region)
		}
	}
}

func TestMeasureLayoutSkipsCollapsedFloatingPaneAndHitRegions(t *testing.T) {
	shell := ShellVM{
		Layout: LayoutVM{
			Panels: []PanelVM{{
				ID:           "pane-1",
				Presentation: PanelPresentationCard,
				Active:       true,
				Content:      ContentVM{Kind: ContentTerminalLive},
			}},
			Floating: []FloatingVM{{
				ID:        "float-1",
				PaneID:    "float-pane-1",
				Title:     "float",
				Rect:      Rect{X: 10, Y: 4, W: 30, H: 8},
				Z:         2,
				Active:    true,
				Collapsed: true,
				Content:   ContentVM{Kind: ContentTerminalLive},
			}},
		},
	}

	plan := MeasureLayout(shell, Rect{W: 80, H: 24})
	if len(plan.Floatings) != 0 {
		t.Fatalf("collapsed floating should not produce a layout plan, got %#v", plan.Floatings)
	}
	for _, action := range []string{
		ActionFloatingCenter.String(),
		ActionFloatingCollapse.String(),
		ActionPaneZoom.String(),
		ActionFloatingClose.String(),
		ActionFloatingMoveDrag.String(),
		ActionFloatingResizeDrag.String(),
		ActionFloatingRaise.String(),
	} {
		for _, region := range plan.HitRegions {
			if region.ActionID == action && (region.Floating || region.PaneID == "float-pane-1") {
				t.Fatalf("collapsed floating should not expose action %s, hit regions=%#v", action, plan.HitRegions)
			}
		}
	}
	for _, region := range plan.HitRegions {
		if region.Floating || region.PaneID == "float-pane-1" {
			t.Fatalf("collapsed floating should not expose floating hit regions, got %#v in %#v", region, plan.HitRegions)
		}
	}
}

func TestMeasureLayoutCentersEmptyPaneActionHitRegions(t *testing.T) {
	lines, regions, cursor := emptyPaneContentLayout("pane-1", 0)
	shell := ShellVM{Layout: LayoutVM{Panels: []PanelVM{{
		ID:           "pane-1",
		Presentation: PanelPresentationCard,
		Active:       true,
		Content:      ContentVM{Kind: ContentEmptyPane, Lines: lines, HitRegions: regions, Cursor: cursor},
	}}}}

	plan := MeasureLayout(shell, Rect{W: 40, H: 10})
	if len(plan.Panels) != 1 {
		t.Fatalf("expected one panel, got %#v", plan.Panels)
	}
	contentRect := plan.Panels[0].ContentRect
	attach := hitRegionByAction(t, plan.HitRegions, ActionEmptyAttach.String())
	create := hitRegionByAction(t, plan.HitRegions, ActionEmptyCreate.String())
	attachWidth := DisplayWidth("► Attach existing terminal ◄")
	createWidth := DisplayWidth("[ Create new terminal ]")
	if attach.Rect != (Rect{X: contentRect.X + (contentRect.W-attachWidth)/2, Y: contentRect.Y + 4, W: attachWidth, H: 1}) {
		t.Fatalf("attach hit region should match centered selected row content=%#v got=%#v", contentRect, attach)
	}
	if create.Rect != (Rect{X: contentRect.X + (contentRect.W-createWidth)/2, Y: contentRect.Y + 5, W: createWidth, H: 1}) {
		t.Fatalf("create hit region should match centered bracket row content=%#v got=%#v", contentRect, create)
	}
}

func TestMeasureLayoutBottomAlignsExitedPaneActionHitRegions(t *testing.T) {
	lines := []Line{
		NewLine("history A"),
		NewLine("history B"),
		NewLine("history C"),
		NewLine("history D"),
		NewLine(""),
		NewLine("terminal exited: term-1 code:23"),
		NewLine("exited at: 2026-06-17T12:30:00Z"),
		NewLine("command: bash -lc exit 23"),
		centeredStyledLine("► restart ◄", StyleWarning),
		centeredStyledLine("[ reconnect ]", StyleMuted),
	}
	regions := []HitRegion{
		{Kind: HitRegionContentAction, Rect: Rect{Y: 8, W: DisplayWidth("► restart ◄"), H: 1}, ActionID: ActionExitedRestart.String(), Invocation: actiondomain.Invocation{ID: actiondomain.ActionExitedRestart, SourceActionID: actiondomain.ActionExitedRestart.String()}, TargetMode: HitTargetExplicit},
		{Kind: HitRegionContentAction, Rect: Rect{Y: 9, W: DisplayWidth("[ reconnect ]"), H: 1}, ActionID: ActionExitedReconnect.String(), Invocation: actiondomain.Invocation{ID: actiondomain.ActionExitedReconnect, SourceActionID: actiondomain.ActionExitedReconnect.String()}, TargetMode: HitTargetExplicit},
	}
	shell := ShellVM{
		Header: HeaderVM{Visible: false},
		Footer: FooterVM{Visible: false},
		Layout: LayoutVM{Panels: []PanelVM{{
			ID:           "pane-1",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content:      ContentVM{Kind: ContentExitedPane, Lines: lines, HitRegions: regions},
		}}},
	}

	plan := MeasureLayout(shell, Rect{W: 82, H: 10})
	if len(plan.Panels) != 1 {
		t.Fatalf("expected one panel, got %#v", plan.Panels)
	}
	contentRect := plan.Panels[0].ContentRect
	restart := hitRegionByAction(t, plan.HitRegions, ActionExitedRestart.String())
	picker := hitRegionByAction(t, plan.HitRegions, ActionExitedReconnect.String())
	restartWidth := DisplayWidth("► restart ◄")
	pickerWidth := DisplayWidth("[ reconnect ]")
	if restart.Rect != (Rect{X: contentRect.X + (contentRect.W-restartWidth)/2, Y: contentRect.Y + 6, W: restartWidth, H: 1}) {
		t.Fatalf("restart hit region should match bottom-aligned visible action content=%#v got=%#v", contentRect, restart)
	}
	if picker.Rect != (Rect{X: contentRect.X + (contentRect.W-pickerWidth)/2, Y: contentRect.Y + 7, W: pickerWidth, H: 1}) {
		t.Fatalf("picker hit region should match bottom-aligned visible action content=%#v got=%#v", contentRect, picker)
	}
}

func hitRegionIndexByAction(regions []HitRegion, actionID string) int {
	for i, region := range regions {
		if region.ActionID == actionID {
			return i
		}
	}
	return -1
}

func hitRegionByAction(t *testing.T, regions []HitRegion, actionID string) HitRegion {
	t.Helper()
	for _, region := range regions {
		if region.ActionID == actionID {
			return region
		}
	}
	t.Fatalf("missing action %s in %#v", actionID, regions)
	return HitRegion{}
}

func findHitRegionByAction(regions []HitRegion, actionID string) (HitRegion, bool) {
	for _, region := range regions {
		if region.ActionID == actionID {
			return region, true
		}
	}
	return HitRegion{}, false
}

func hitRegionIndexByPaneAndAction(regions []HitRegion, kind HitRegionKind, paneID string, actionID string) int {
	for i, region := range regions {
		if region.Kind == kind && region.PaneID == paneID && region.ActionID == actionID {
			return i
		}
	}
	return -1
}

func hitRegionIndexByActionAndDirection(regions []HitRegion, actionID string, direction string) int {
	for i, region := range regions {
		if region.ActionID == actionID && region.Direction == direction {
			return i
		}
	}
	return -1
}

func rectsOverlap(a Rect, b Rect) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

func hitRegionByActionAndPane(t *testing.T, regions []HitRegion, actionID string, paneID string) HitRegion {
	t.Helper()
	for _, region := range regions {
		if region.ActionID == actionID && region.PaneID == paneID {
			return region
		}
	}
	t.Fatalf("missing action=%s pane=%s in %#v", actionID, paneID, regions)
	return HitRegion{}
}

func resizeGroupSignature(group []ResizeGroupItem) string {
	parts := make([]string, 0, len(group))
	for _, item := range group {
		parts = append(parts, item.PaneID+":"+strconv.Itoa(item.Cells))
	}
	return strings.Join(parts, ",")
}

func resizeGroupSignSignature(group []ResizeGroupItem) string {
	parts := make([]string, 0, len(group))
	for _, item := range group {
		sign := strconv.Itoa(item.DeltaSign)
		if item.DeltaSign > 0 {
			sign = "+" + sign
		}
		parts = append(parts, item.PaneID+":"+sign)
	}
	return strings.Join(parts, ",")
}
