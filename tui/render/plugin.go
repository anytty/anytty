package render

import (
	"fmt"
	"strings"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

type PluginMountVM struct {
	Mount   state.PluginMount
	Focused bool
	Error   string
	Hint    string
}
type PluginLayout struct {
	VM   PluginMountVM
	Rect Rect
}

func pluginMountVMs(root state.Root) []PluginMountVM {
	var out []PluginMountVM
	for _, m := range root.Plugins.Visible(root.Shell) {
		if m.Slot == "header" && (m.Owner.Kind == "panel" || m.Owner.Kind == "floating") {
			var b state.TerminalViewBinding
			if m.Owner.Kind == "floating" {
				b, _ = root.TerminalViews.FloatingBinding(m.Owner.FloatingID)
			} else {
				b, _ = root.TerminalViews.PaneBinding(m.Owner.PaneID)
			}
			var nodes []state.PluginNode
			for _, n := range m.Rows() {
				if n.TerminalID == "" || (n.TerminalID == b.TerminalID && n.EndpointID == b.EndpointID) {
					nodes = append(nodes, n)
				}
			}
			m.Nodes = nodes
		}
		out = append(out, PluginMountVM{Mount: m, Focused: root.Plugins.FocusedMountID == m.ID, Error: root.Plugins.Error, Hint: pluginKeyHint(root, m)})
	}
	return out
}
func measurePluginSpace(shell ShellVM, body Rect) (Rect, []PluginLayout) {
	var out []PluginLayout
	for _, vm := range shell.Plugins {
		m := vm.Mount
		if m.Slot == "sidebar" && body.W >= 30 && body.H > 0 {
			w := m.Width
			if w <= 0 {
				w = 30
			}
			w = minInt(w, body.W/2)
			r := Rect{X: body.X, Y: body.Y, W: w, H: body.H}
			out = append(out, PluginLayout{vm, r})
			body.X += w
			body.W -= w
		}
		if m.Slot == "sidebar" && body.W < 30 && vm.Focused && body.H > 0 {
			out = append(out, PluginLayout{vm, body})
		}
		if m.Slot == "statusbar" && body.H > 2 {
			body.H--
			out = append(out, PluginLayout{vm, Rect{X: body.X, Y: body.Y + body.H, W: body.W, H: 1}})
		}
	}
	return body, out
}
func measurePluginContainers(shell ShellVM, plan LayoutPlan) []PluginLayout {
	out := append([]PluginLayout(nil), plan.Plugins...)
	for _, vm := range shell.Plugins {
		m := vm.Mount
		var r Rect
		switch m.Owner.Kind {
		case "workspace", "tab":
			if m.Slot == "overlay" || m.Slot == "menu" {
				w := minInt(60, plan.Body.W)
				h := minInt(18, plan.Body.H)
				r = Rect{X: plan.Body.X + (plan.Body.W-w)/2, Y: plan.Body.Y + (plan.Body.H-h)/2, W: w, H: h}
			}
			if m.Slot == "content" {
				r = plan.Body
			}
			if m.Slot == "header" {
				r = plan.Header
			}
		case "panel":
			for _, p := range plan.Panels {
				if p.Panel.ID == m.Owner.PaneID {
					r = p.ContentRect
					if m.Slot == "header" {
						r = Rect{X: p.Rect.X + p.Rect.W/2, Y: p.Rect.Y, W: maxInt(8, p.Rect.W/4), H: 1}
					}
				}
			}
		case "floating":
			for _, p := range plan.Floatings {
				if p.Floating.ID == m.Owner.FloatingID {
					r = p.ContentRect
					if m.Slot == "header" {
						r = Rect{X: p.Rect.X + p.Rect.W/2, Y: p.Rect.Y, W: p.Rect.W / 2, H: 1}
					}
				}
			}
		}
		if r.W > 0 && r.H > 0 {
			out = append(out, PluginLayout{vm, r})
		}
	}
	return out
}
func pluginRowWindow(m state.PluginMount, height int) ([]state.PluginNode, int) {
	rows := m.Rows()
	start := 0
	for i, r := range rows {
		if r.ID == m.SelectedID && i >= height {
			start = i - height + 1
		}
	}
	if start > len(rows) {
		start = len(rows)
	}
	return rows[start:], start
}

type pluginCardItem struct {
	Node state.PluginNode
	Row  int
	Rect Rect
}

func pluginUsesCards(m state.PluginMount) bool {
	var visit func([]state.PluginNode) bool
	visit = func(nodes []state.PluginNode) bool {
		for _, node := range nodes {
			if node.Kind == "card" || visit(node.Children) {
				return true
			}
		}
		return false
	}
	return visit(m.Nodes)
}

func pluginCardDescription(node state.PluginNode) string {
	if node.Description != "" {
		return node.Description
	}
	if node.Detail != "" {
		return node.Detail
	}
	if node.Value != "" {
		return node.Value
	}
	return node.Status
}

func pluginCardDimensions(node state.PluginNode) (height, gap int) {
	layout := node.Layout
	contentLines := 1
	if pluginCardDescription(node) != "" {
		contentLines++
	}
	height = maxInt(1, layout.PaddingTop+contentLines+layout.PaddingBottom)
	gap = maxInt(0, layout.GapAfter)
	return height, gap
}

func pluginCardSpan(node state.PluginNode) int {
	height, gap := pluginCardDimensions(node)
	return height + gap
}

func pluginCardWindow(m state.PluginMount, height int) ([]state.PluginNode, int) {
	rows := m.Rows()
	if len(rows) == 0 {
		return nil, 0
	}
	height = maxInt(1, height)
	selected := -1
	for i, node := range rows {
		if node.ID == m.SelectedID {
			selected = i
			break
		}
	}
	start := 0
	if selected >= 0 {
		used := 0
		for i := selected; i >= 0; i-- {
			span := pluginCardSpan(rows[i])
			if i < selected && used+span > height {
				break
			}
			used += span
			start = i
		}
	}
	used := 0
	end := start
	for end < len(rows) {
		span := pluginCardSpan(rows[end])
		if used > 0 && used+span > height {
			break
		}
		used += span
		end++
	}
	if end == start {
		end = minInt(len(rows), start+1)
	}
	return rows[start:end], start
}

func pluginCardItems(m state.PluginMount, rect Rect) []pluginCardItem {
	if rect.W <= 0 || rect.H <= 2 {
		return nil
	}
	rows, start := pluginCardWindow(m, rect.H-2)
	items := make([]pluginCardItem, 0, len(rows))
	y := rect.Y + 2
	bottom := rect.Y + rect.H
	for i, node := range rows {
		height, gap := pluginCardDimensions(node)
		if y >= bottom {
			break
		}
		height = minInt(height, bottom-y)
		if height <= 0 {
			break
		}
		cardX := rect.X + 1
		cardW := maxInt(1, rect.W-2)
		items = append(items, pluginCardItem{Node: node, Row: start + i, Rect: Rect{X: cardX, Y: y, W: cardW, H: height}})
		y += height + gap
	}
	return items
}

func pluginStatusRole(status string) string {
	switch status {
	case "working":
		return "success"
	case "blocked":
		return "warning"
	case "error":
		return "danger"
	case "stale", "exited", "unknown", "idle":
		return "muted"
	default:
		return "primary"
	}
}

func pluginStyleToken(style state.PluginNodeStyle) StyleToken {
	background := style.BackgroundRole
	if background == "" {
		background = "surface"
	}
	foreground := style.ForegroundRole
	if foreground == "" {
		foreground = "primary"
	}
	return StyleToken(fmt.Sprintf("plugin/%s/%s/%t/%t", background, foreground, style.Bold, style.Dim))
}

func pluginCardNodeStyle(node state.PluginNode, selected bool) state.PluginNodeStyle {
	style := node.Style
	if selected {
		style = node.SelectedStyle
		if style == (state.PluginNodeStyle{}) {
			style = state.PluginNodeStyle{ForegroundRole: "primary", BackgroundRole: "selected", Bold: true}
			return style
		}
		if style.BackgroundRole == "" {
			style.BackgroundRole = "selected"
		}
		if style.ForegroundRole == "" {
			style.ForegroundRole = "primary"
		}
	} else {
		if style.BackgroundRole == "" {
			style.BackgroundRole = "surface"
		}
		if style.ForegroundRole == "" {
			style.ForegroundRole = pluginStatusRole(node.Status)
		}
	}
	return style
}

func pluginCardMetaStyle(node state.PluginNode, selected bool) StyleToken {
	style := pluginCardNodeStyle(node, selected)
	style.ForegroundRole = "muted"
	style.Dim = true
	return pluginStyleToken(style)
}

func pluginHitRegions(plan LayoutPlan) []HitRegion {
	var out []HitRegion
	for i := len(plan.Plugins) - 1; i >= 0; i-- {
		p := plan.Plugins[i]
		if !p.VM.Mount.Interactive {
			continue
		}
		r := p.Rect
		if pluginUsesCards(p.VM.Mount) {
			for _, item := range pluginCardItems(p.VM.Mount, r) {
				out = append(out, HitRegion{Kind: HitRegionPlugin, Rect: item.Rect, PluginMountID: p.VM.Mount.ID, PluginNodeID: item.Node.ID, Row: item.Row, HasRow: true})
			}
			out = append(out, HitRegion{Kind: HitRegionPlugin, Rect: r, PluginMountID: p.VM.Mount.ID})
			continue
		}
		offset := 0
		if r.H > 1 {
			offset = 2
		}
		rows, start := pluginRowWindow(p.VM.Mount, maxInt(1, r.H-offset))
		for j, n := range rows {
			if j+offset >= r.H {
				break
			}
			out = append(out, HitRegion{Kind: HitRegionPlugin, Rect: Rect{X: r.X, Y: r.Y + j + offset, W: r.W, H: 1}, PluginMountID: p.VM.Mount.ID, PluginNodeID: n.ID, Row: start + j, HasRow: true})
		}
		out = append(out, HitRegion{Kind: HitRegionPlugin, Rect: r, PluginMountID: p.VM.Mount.ID})
	}
	return out
}
func renderPlugins(c *canvas, plan LayoutPlan) {
	for _, p := range plan.Plugins {
		r := p.Rect
		m := p.VM.Mount
		owner := "plugin:" + m.ID
		if r.H == 1 {
			var texts []string
			for _, n := range m.Rows() {
				texts = append(texts, n.Text)
			}
			text := strings.Join(texts, " · ")
			if m.Stale {
				text = "offline · " + text
			}
			// Header mounts are additive badges. Native panel controls remain visible.
			c.overlayTextStyled(r.X, r.Y, r.W, text, StyleStatusAccent, owner, LayerPanel)
			continue
		}
		title := m.Title
		if m.Stale {
			title += " [offline]"
		}
		if pluginUsesCards(m) {
			baseStyle := pluginStyleToken(state.PluginNodeStyle{ForegroundRole: "primary", BackgroundRole: "surface"})
			c.fillStyledRect(r, baseStyle, owner, LayerPanel)
			titleStyle := pluginStyleToken(state.PluginNodeStyle{ForegroundRole: "primary", BackgroundRole: "surface", Bold: p.VM.Focused})
			c.overlayTextStyled(r.X+1, r.Y, maxInt(1, r.W-2), title, titleStyle, owner, LayerPanel)
			hint := "↑↓ select  Enter open"
			if p.VM.Hint != "" {
				hint = p.VM.Hint
			}
			if m.Searching || m.Search != "" {
				hint = "/ " + m.Search
			}
			if p.VM.Error != "" {
				hint = p.VM.Error
			}
			c.overlayTextStyled(r.X+1, r.Y+1, maxInt(1, r.W-2), hint, pluginStyleToken(state.PluginNodeStyle{ForegroundRole: "muted", BackgroundRole: "surface", Dim: true}), owner, LayerPanel)
			items := pluginCardItems(m, r)
			for _, item := range items {
				selected := item.Node.ID == m.SelectedID
				style := pluginCardNodeStyle(item.Node, selected)
				cardStyle := pluginStyleToken(style)
				c.fillStyledRect(item.Rect, cardStyle, owner, LayerPanel)
				left := item.Rect.X + maxInt(0, item.Node.Layout.PaddingLeft)
				width := maxInt(1, item.Rect.W-item.Node.Layout.PaddingLeft-item.Node.Layout.PaddingRight)
				titleY := item.Rect.Y + item.Node.Layout.PaddingTop
				prefix := ""
				if selected {
					prefix = "› "
				}
				c.overlayTextStyled(left, titleY, width, prefix+item.Node.Text, cardStyle, owner, LayerPanel)
				description := pluginCardDescription(item.Node)
				if description != "" && titleY+1 < item.Rect.Y+item.Rect.H {
					c.overlayTextStyled(left, titleY+1, width, description, pluginCardMetaStyle(item.Node, selected), owner, LayerPanel)
				}
			}
			if len(items) == 0 {
				c.overlayTextStyled(r.X+1, r.Y+2, maxInt(1, r.W-2), "No matching items", pluginStyleToken(state.PluginNodeStyle{ForegroundRole: "muted", BackgroundRole: "surface", Dim: true}), owner, LayerPanel)
			}
			continue
		}
		c.fillStyledRect(r, StyleForeground, owner, LayerPanel)
		style := StyleMuted
		if p.VM.Focused {
			style = StyleAccent
		}
		c.overlayTextStyled(r.X, r.Y, r.W, title, style, owner, LayerPanel)
		hint := "↑↓ select  Enter open"
		if p.VM.Hint != "" {
			hint = p.VM.Hint
		}
		if m.Searching || m.Search != "" {
			hint = "/ " + m.Search
		}
		if p.VM.Error != "" {
			hint = p.VM.Error
		}
		c.overlayTextStyled(r.X, r.Y+1, r.W, hint, StyleMuted, owner, LayerPanel)
		rows, _ := pluginRowWindow(m, maxInt(1, r.H-2))
		for i, n := range rows {
			if i+2 >= r.H {
				break
			}
			prefix := "  "
			style := pluginStatusStyle(n.Status)
			if n.ID == m.SelectedID {
				prefix = "› "
				style = StylePickerAccent
			}
			if n.Disabled {
				style = StyleMuted
			}
			text := prefix + strings.Repeat(" ", minInt(n.Depth, 16)) + pluginNodeText(m, n, r.W-2-n.Depth)
			if n.Kind == "progress" {
				text += fmt.Sprintf(" %.0f%%", n.Progress*100)
			}
			if n.Status != "" {
				text += " · " + n.Status
			}
			c.overlayTextStyled(r.X, r.Y+2+i, r.W, text, style, owner, LayerPanel)
		}
		if len(rows) == 0 {
			c.overlayTextStyled(r.X, r.Y+2, r.W, "No matching items", StyleMuted, owner, LayerPanel)
		}
	}
}

func pluginStatusStyle(status string) StyleToken {
	switch status {
	case "working":
		return StyleSuccess
	case "blocked", "error":
		return StyleWarning
	case "stale", "exited", "unknown":
		return StyleMuted
	default:
		return StyleForeground
	}
}

func renderPluginHeaderMounts(c *canvas, plan LayoutPlan) {
	headers := LayoutPlan{}
	for _, p := range plan.Plugins {
		if p.VM.Mount.Owner.Kind == "tab" && p.VM.Mount.Slot == "header" {
			headers.Plugins = append(headers.Plugins, p)
		}
	}
	renderPlugins(c, headers)
}

func pluginKeyHint(root state.Root, m state.PluginMount) string {
	var hints []string
	for _, a := range m.Actions {
		if !a.GetEnabled() {
			continue
		}
		key := a.GetDefaultKey()
		if override, ok := root.Config.PluginShortcuts[m.PluginID+"/"+a.GetId()]; ok {
			key = override
		}
		if key != "" {
			hints = append(hints, input.ShortcutKeyDisplay(key)+" "+a.GetLabel())
		}
	}
	return strings.Join(hints, " · ")
}

func pluginNodeText(m state.PluginMount, n state.PluginNode, width int) string {
	switch n.Kind {
	case "tree":
		mark := "▾ "
		if m.Collapsed[n.ID] {
			mark = "▸ "
		}
		return mark + n.Text
	case "input":
		value := n.Value
		if m.EditingID == n.ID {
			runes := []rune(value)
			cursor := m.EditCursor
			if cursor > len(runes) {
				cursor = len(runes)
			}
			if cursor < 0 {
				cursor = 0
			}
			limit := maxInt(1, width-DisplayWidth(n.Text)-4)
			start := maxInt(0, cursor-limit+1)
			value = string(runes[start:cursor]) + "│" + string(runes[cursor:])
			if start > 0 {
				value = "…" + value
			}
		} else if value == "" {
			value = n.Placeholder
		}
		return n.Text + ": " + value
	case "select":
		label := n.Value
		for _, option := range n.Children {
			if option.Value == n.Value {
				label = option.Text
				break
			}
		}
		return n.Text + ": ‹ " + label + " ›"
	case "checkbox":
		mark := "[ ] "
		if n.Value == "true" {
			mark = "[x] "
		}
		return mark + n.Text
	case "button":
		return "[ " + n.Text + " ]"
	}
	return n.Text
}
