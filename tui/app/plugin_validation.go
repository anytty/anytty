package app

import (
	"fmt"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/input"
	"math"
	"strings"
)

// validatePluginMount rejects unsupported content rather than accepting an
// invisible PTY or a component that cannot be operated by this host.
func validatePluginMount(update *apipb.PluginUiMountUpdate) error {
	if strings.ContainsAny(update.GetTitle(), "\x1b\r\n") {
		return fmt.Errorf("title must be one line without ANSI escapes")
	}
	nodes := map[string]bool{}
	count := 0
	var walk func(*apipb.PluginUiNode, int) error
	walk = func(n *apipb.PluginUiNode, depth int) error {
		if n == nil {
			return nil
		}
		count++
		if count > 4096 || depth >= 16 {
			return fmt.Errorf("UI tree exceeds 4096 nodes or 16 levels")
		}
		if n.GetId() == "" || nodes[n.GetId()] {
			return fmt.Errorf("UI node IDs must be nonempty and unique")
		}
		nodes[n.GetId()] = true
		switch n.GetKind() {
		case "text", "badge", "progress", "button", "list", "row", "card", "gap", "column", "table", "tree", "form", "input", "select", "option", "checkbox":
		default:
			return fmt.Errorf("unsupported UI component %q", n.GetKind())
		}
		if n.GetKind() == "progress" && (math.IsNaN(n.GetProgress()) || n.GetProgress() < 0 || n.GetProgress() > 1) {
			return fmt.Errorf("progress must be between zero and one")
		}
		if err := validatePluginNodeStyle(n.GetStyle()); err != nil {
			return err
		}
		if err := validatePluginNodeStyle(n.GetSelectedStyle()); err != nil {
			return err
		}
		if layout := n.GetLayout(); layout != nil {
			if layout.GetPaddingTop() > 4 || layout.GetPaddingRight() > 4 || layout.GetPaddingBottom() > 4 || layout.GetPaddingLeft() > 4 || layout.GetGapAfter() > 4 {
				return fmt.Errorf("plugin layout spacing exceeds four cells")
			}
		}
		if strings.ContainsAny(n.GetText()+n.GetStatus()+n.GetValue()+n.GetDescription()+n.GetPlaceholder(), "\x1b\r\n") {
			return fmt.Errorf("component text must be one line without ANSI escapes")
		}
		switch n.GetKind() {
		case "text", "badge", "progress", "button", "input", "option", "checkbox":
			if len(n.GetChildren()) > 0 {
				return fmt.Errorf("%s is a leaf component", n.GetKind())
			}
		case "select":
			values := map[string]bool{}
			for _, o := range n.GetChildren() {
				if o.GetKind() != "option" || values[o.GetValue()] {
					return fmt.Errorf("select children must be options with unique values")
				}
				values[o.GetValue()] = true
			}
		}
		if n.GetKind() == "checkbox" && n.GetValue() != "" && n.GetValue() != "true" && n.GetValue() != "false" {
			return fmt.Errorf("checkbox value must be true or false")
		}
		for _, c := range n.GetChildren() {
			if c.GetKind() == "option" && n.GetKind() != "select" {
				return fmt.Errorf("option requires select parent")
			}
			if err := walk(c, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(update.GetRoot(), 0); err != nil {
		return err
	}
	actions := map[string]bool{}
	for _, a := range update.GetActions() {
		if a.GetId() == "" || actions[a.GetId()] {
			return fmt.Errorf("action IDs must be nonempty and unique")
		}
		actions[a.GetId()] = true
		switch a.GetScope() {
		case "", "mount", "panel", "floating", "tab", "workspace", "global":
		default:
			return fmt.Errorf("unsupported action scope %q", a.GetScope())
		}
		if a.GetDefaultKey() != "" {
			if input.ShortcutKeyIsGlobalEscape(a.GetDefaultKey()) {
				return fmt.Errorf("Escape is reserved")
			}
			if _, ok := input.ShortcutBindingSignature("plugin", a.GetDefaultKey()); !ok {
				return fmt.Errorf("invalid plugin default key")
			}
		}
	}
	return nil
}

func validatePluginNodeStyle(style *apipb.PluginUiStyle) error {
	if style == nil {
		return nil
	}
	validForeground := map[string]bool{"": true, "primary": true, "muted": true, "success": true, "warning": true, "danger": true, "info": true}
	validBackground := map[string]bool{"": true, "surface": true, "elevated": true, "selected": true, "transparent": true}
	if !validForeground[style.GetForegroundRole()] {
		return fmt.Errorf("unsupported plugin foreground role %q", style.GetForegroundRole())
	}
	if !validBackground[style.GetBackgroundRole()] {
		return fmt.Errorf("unsupported plugin background role %q", style.GetBackgroundRole())
	}
	return nil
}
