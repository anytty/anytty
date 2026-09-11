package app

import (
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
	"strings"
	"unicode"
)

func pluginField(node state.PluginNode) bool {
	return node.Kind == "input" || node.Kind == "select" || node.Kind == "checkbox"
}
func pluginFormInput(root state.Root, m state.PluginMount, event input.InputEvent, deps PluginDeps) (state.Root, []Effect, bool) {
	if event.Kind == input.EventKindKey && (event.Ctrl || event.Alt) {
		if _, ok := input.ShortcutEntryForEvent(root.Config.Shortcuts, "global", event); ok {
			return root, nil, false
		}
	}
	node, ok := m.Selected()
	if !ok || node.Disabled || m.Stale {
		return root, nil, false
	}
	finish := func(kind string) (state.Root, []Effect, bool) {
		root.Plugins = root.Plugins.Set(m)
		n, _ := m.Selected()
		next, effects := pluginEmitInteraction(root, m, n, kind, deps, event)
		return next, effects, true
	}
	if node.Kind == "tree" && (event.Kind == input.EventKindMouse && event.Mouse == input.MouseLeft || event.Kind == input.EventKindKey && (event.Key == input.KeyLeft || event.Key == input.KeyRight || event.Key == input.KeyEnter)) {
		collapsed := !m.Collapsed[node.ID]
		if event.Key == input.KeyLeft {
			collapsed = true
		}
		if event.Key == input.KeyRight {
			collapsed = false
		}
		m = m.SetCollapsed(node.ID, collapsed)
		kind := "expand"
		if collapsed {
			kind = "collapse"
		}
		return finish(kind)
	}
	if event.Kind == input.EventKindMouse && event.Mouse == input.MouseLeft {
		switch node.Kind {
		case "button":
			next, effects := pluginActivateAction(root, m, "", deps, event)
			return next, effects, true
		case "input":
			m.EditingID = node.ID
			m.EditOriginal = node.Value
			m.EditCursor = len([]rune(node.Value))
			return finish("focus")
		case "checkbox":
			m = m.WithValue(node.ID, pluginToggle(node.Value))
			return finish("change")
		case "select":
			m = pluginSelectNext(m, node, 1)
			return finish("change")
		}
	}
	if event.Kind != input.EventKindKey && event.Kind != input.EventKindPaste {
		return root, nil, false
	}
	if m.EditingID == node.ID {
		if event.Key == input.KeyEsc {
			m = m.WithValue(node.ID, m.EditOriginal)
			m.EditingID = ""
			return finish("cancel")
		}
		if event.Key == input.KeyTab || event.Key == input.KeyShiftTab {
			m.EditingID = ""
			delta := 1
			if event.Key == input.KeyShiftTab {
				delta = -1
			}
			m = m.MoveFocus(delta)
			return finish("focus")
		}
		if event.Key == input.KeyEnter {
			m.EditingID = ""
			if action := m.FormAction(); action != "" {
				root.Plugins = root.Plugins.Set(m)
				next, effects := pluginActivateAction(root, m, action, deps, event)
				return next, effects, true
			}
			return finish("change")
		}
		if node.Kind == "select" {
			switch event.Key {
			case input.KeyLeft, input.KeyUp:
				m = pluginSelectNext(m, node, -1)
				return finish("change")
			case input.KeyRight, input.KeyDown:
				m = pluginSelectNext(m, node, 1)
				return finish("change")
			}
			return root, []Effect{handledEffect{}}, true
		}
		if node.Kind == "input" {
			value := []rune(node.Value)
			cursor := m.EditCursor
			if cursor > len(value) {
				cursor = len(value)
			}
			if cursor < 0 {
				cursor = 0
			}
			text := ""
			changed := false
			switch {
			case event.Kind == input.EventKindPaste:
				text = event.Paste
			case event.Key == input.KeyChar && !event.Ctrl && !event.Alt:
				text = event.Char
				if event.Text != "" {
					text = event.Text
				} else if event.ShiftedChar != "" {
					text = event.ShiftedChar
				}
			case event.Key == input.KeyBackspace:
				if cursor > 0 {
					value = append(value[:cursor-1], value[cursor:]...)
					cursor--
					changed = true
				}
			case event.Key == input.KeyDelete:
				if cursor < len(value) {
					value = append(value[:cursor], value[cursor+1:]...)
					changed = true
				}
			case event.Key == input.KeyLeft:
				if cursor > 0 {
					cursor--
				}
			case event.Key == input.KeyRight:
				if cursor < len(value) {
					cursor++
				}
			case event.Key == input.KeyHome:
				cursor = 0
			case event.Key == input.KeyEnd:
				cursor = len(value)
			}
			text = strings.Map(func(r rune) rune {
				if unicode.IsControl(r) {
					return -1
				}
				return r
			}, text)
			if text != "" {
				insert := []rune(text)
				if len(value)+len(insert) > 4096 {
					insert = insert[:max(0, 4096-len(value))]
				}
				updated := append([]rune(nil), value[:cursor]...)
				updated = append(updated, insert...)
				updated = append(updated, value[cursor:]...)
				value = updated
				cursor += len(insert)
				changed = true
			}
			m.EditCursor = cursor
			m = m.WithValue(node.ID, string(value))
			if changed {
				return finish("change")
			}
			root.Plugins = root.Plugins.Set(m)
			return root.Advance(), []Effect{handledEffect{}}, true
		}
	}
	if pluginField(node) && (event.Key == input.KeyEnter || (event.Key == input.KeyChar && event.Char == " ")) {
		if node.Kind == "checkbox" {
			m = m.WithValue(node.ID, pluginToggle(node.Value))
			return finish("change")
		}
		m.EditingID = node.ID
		m.EditOriginal = node.Value
		m.EditCursor = len([]rune(node.Value))
		return finish("focus")
	}
	return root, nil, false
}
func pluginToggle(value string) string {
	if value == "true" {
		return "false"
	}
	return "true"
}
func pluginSelectNext(m state.PluginMount, node state.PluginNode, delta int) state.PluginMount {
	if len(node.Children) == 0 {
		return m
	}
	selected := -1
	for i, o := range node.Children {
		if o.Value == node.Value {
			selected = i
			break
		}
	}
	for attempts := 0; attempts < len(node.Children); attempts++ {
		selected = (selected + delta + len(node.Children)) % len(node.Children)
		if !node.Children[selected].Disabled {
			return m.WithValue(node.ID, node.Children[selected].Value)
		}
	}
	return m
}
