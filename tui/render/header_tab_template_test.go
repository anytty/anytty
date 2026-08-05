package render

import (
	"strings"
	"testing"

	tuiconfig "github.com/anytty/anytty/tui/config"
)

func TestHeaderTabDefaultSegmentsKeepStableWidth(t *testing.T) {
	active := headerTabSegmentParts(2, "logs", true, "tab-logs", ActionTabClose.String(), "tab-logs", "")
	inactive := headerTabSegmentParts(2, "logs", false, "tab-logs", ActionTabClose.String(), "tab-logs", "")
	if barSegmentsWidth(active) != barSegmentsWidth(inactive) {
		t.Fatalf("active/inactive tab width must stay stable, active=%d inactive=%d activeSegments=%#v inactiveSegments=%#v", barSegmentsWidth(active), barSegmentsWidth(inactive), active, inactive)
	}
	if got := (Line{Cells: cellsFromBarSegments(active)}).PlainString(); !strings.Contains(got, "▎ 2 logs "+paneChromeCloseGlyph()) {
		t.Fatalf("active tab should keep marker/index/title/close slots, got %q", got)
	}
	if got := (Line{Cells: cellsFromBarSegments(inactive)}).PlainString(); strings.Contains(got, "▎") || !strings.Contains(got, "  2 logs "+paneChromeCloseGlyph()) {
		t.Fatalf("inactive tab should keep marker footprint without active marker, got %q", got)
	}
}

func TestPortableDefaultHeaderTemplatesKeepUnicodeTabWidthsStable(t *testing.T) {
	workspace := headerWorkspaceTemplateSegments(tuiconfig.DefaultWorkspaceTemplate, "工作区🚀")
	if got := (Line{Cells: cellsFromBarSegments(workspace)}).PlainString(); got != " WS 工作区🚀  " {
		t.Fatalf("portable workspace template got %q", got)
	}
	ctx := headerTabTemplateContext{
		Index:        2,
		Title:        "日志🚀",
		TabID:        "tab-logs",
		SwitchAction: ActionTabSwitch.String(),
		CloseAction:  ActionTabClose.String(),
		CloseTarget:  "tab-logs",
		CloseIcon:    DefaultPaneChromeGlyphs().Close,
	}
	ctx.Active = true
	active := headerTabTemplateSegments(tuiconfig.DefaultTabTemplate, ctx)
	ctx.Active = false
	inactive := headerTabTemplateSegments(tuiconfig.DefaultTabTemplate, ctx)
	if activeWidth, inactiveWidth := barSegmentsWidth(active), barSegmentsWidth(inactive); activeWidth != inactiveWidth || activeWidth <= 0 {
		t.Fatalf("default unicode tab widths must remain stable, active=%d inactive=%d", activeWidth, inactiveWidth)
	}
	for _, segments := range [][]barSegment{active, inactive} {
		plain := (Line{Cells: cellsFromBarSegments(segments)}).PlainString()
		if !strings.Contains(plain, "日志🚀") || !strings.Contains(plain, DefaultPaneChromeGlyphs().Close) {
			t.Fatalf("default tab must keep unicode title and visible close action, got %q", plain)
		}
	}
}

func TestHeaderTabTemplateSupportsVariablesStyleAndActions(t *testing.T) {
	template := "{{tab_id}}[fg:#ffffff;bg:#332244;font:bold,italic] {{if active}}▎{{else}}|{{end}} {{title | truncate 4}} [action:tab.close]{{close_icon}}[/action][/style]"
	segments := headerTabTemplateSegments(template, headerTabTemplateContext{
		Index:        3,
		Title:        "build-log",
		TabID:        "tab-build",
		Active:       true,
		SwitchAction: ActionTabSwitch.String(),
		CloseAction:  ActionTabClose.String(),
		CloseTarget:  "tab-build",
		CloseIcon:    paneChromeCloseGlyph(),
	})
	if len(segments) == 0 {
		t.Fatalf("expected template segments")
	}
	line := Line{Cells: cellsFromBarSegments(segments)}
	plain := line.PlainString()
	if !strings.Contains(plain, "tab-build") || !strings.Contains(plain, "▎ buil") || !strings.Contains(plain, paneChromeCloseGlyph()) || !strings.HasSuffix(plain, "") {
		t.Fatalf("template did not render expected tab text, got %q", plain)
	}
	foundANSI := false
	foundClose := false
	for _, segment := range segments {
		if segment.ansi.FG == "#ffffff" && segment.ansi.BG == "#332244" && segment.ansi.Bold && segment.ansi.Italic {
			foundANSI = true
		}
		if segment.actionID == ActionTabClose.String() && segment.targetID == "tab-build" && strings.Contains(segment.text, paneChromeCloseGlyph()) {
			foundClose = true
		}
	}
	if !foundANSI || !foundClose {
		t.Fatalf("template should emit ANSI style and close action target, ansi=%v close=%v segments=%#v", foundANSI, foundClose, segments)
	}
}

func TestHeaderWorkspaceTemplateUsesNavigatorActionAndEdgeStyle(t *testing.T) {
	segments := headerWorkspaceTemplateSegments("[style:header-workspace-edge][/style][style:header-workspace] {{workspace | truncate 4}} [/style]", "build-prod")
	if len(segments) == 0 {
		t.Fatalf("expected workspace template segments")
	}
	line := Line{Cells: cellsFromBarSegments(segments)}
	if got := line.PlainString(); got != " buil " {
		t.Fatalf("workspace template should render truncated workspace pill, got %q", got)
	}
	foundEdge := false
	for _, segment := range segments {
		if segment.actionID != "menu.workbench_tree" || segment.targetID != "build-prod" {
			t.Fatalf("workspace template must keep navigator action target, segments=%#v", segments)
		}
		if segment.style == StyleHeaderWorkspaceEdge {
			foundEdge = true
		}
	}
	if !foundEdge {
		t.Fatalf("workspace template should allow workspace edge style, segments=%#v", segments)
	}
}

func TestHeaderSegmentsUseConfiguredWorkspaceTemplate(t *testing.T) {
	segments := headerLeftSegments(HeaderVM{
		Workspace:         "ops",
		WorkspaceTemplate: "[style:header-workspace-edge][/style][style:header-workspace] {{workspace}} [/style][style:header-workspace-edge][/style]",
	})
	line := Line{Cells: cellsFromBarSegments(segments)}
	if got := line.PlainString(); !strings.HasPrefix(got, " ops ") {
		t.Fatalf("header should render configured workspace template before tabs, got %q", got)
	}
}

func TestHeaderCreateTemplateUsesCreateActionAndIcon(t *testing.T) {
	segments := headerCreateTemplateSegments("[fg:#040a0d;bg:#8ffcff;font:bold] {{create_icon}} [fg:#8ffcff;bg:#040a0d]", "󰐕")
	if len(segments) == 0 {
		t.Fatalf("expected create template segments")
	}
	line := Line{Cells: cellsFromBarSegments(segments)}
	if got := line.PlainString(); got != " 󰐕 " {
		t.Fatalf("create template should render configured icon, got %q", got)
	}
	foundANSI := false
	for _, segment := range segments {
		if segment.actionID != ActionTabCreate.String() || segment.targetID != "" {
			t.Fatalf("create template must keep tab.create action without tab target, segments=%#v", segments)
		}
		if segment.ansi.FG == "#040a0d" && segment.ansi.BG == "#8ffcff" && segment.ansi.Bold {
			foundANSI = true
		}
	}
	if !foundANSI {
		t.Fatalf("create template should preserve ANSI styling, segments=%#v", segments)
	}
}

func TestHeaderSegmentsUseConfiguredCreateTemplate(t *testing.T) {
	segments := headerLeftSegments(HeaderVM{
		Workspace:         "ops",
		TabCreateIcon:     "",
		TabCreateTemplate: "[fg:#040a0d;bg:#ff6bff] {{create_icon}} [fg:#ff6bff;bg:#040a0d]",
		WorkspaceTemplate: "[action:none]",
		TabTemplate:       "",
		Tab:               "",
		Tabs:              nil,
		TerminalSummary:   "",
		FloatingSummary:   "",
		Notice:            "",
		ActivePane:        "",
	})
	line := Line{Cells: cellsFromBarSegments(segments)}
	if got := line.PlainString(); !strings.Contains(got, "  ") {
		t.Fatalf("header should render configured create template, got %q", got)
	}
}

func TestHeaderTabTemplateEscapesDynamicTextBeforeSpanParse(t *testing.T) {
	segments := headerTabTemplateSegments("{{title}}", headerTabTemplateContext{
		Index:        1,
		Title:        "[action:tab.close]raw[/action]",
		TabID:        "tab-main",
		Active:       true,
		SwitchAction: ActionTabSwitch.String(),
		CloseAction:  ActionTabClose.String(),
		CloseTarget:  "tab-main",
		CloseIcon:    paneChromeCloseGlyph(),
	})
	line := Line{Cells: cellsFromBarSegments(segments)}
	if got := line.PlainString(); got != "[action:tab.close]raw[/action]" {
		t.Fatalf("dynamic title should render as literal text, got %q", got)
	}
	for _, segment := range segments {
		if segment.actionID != ActionTabSwitch.String() || segment.targetID != "tab-main" {
			t.Fatalf("dynamic title must not inject template action, segments=%#v", segments)
		}
	}
}
