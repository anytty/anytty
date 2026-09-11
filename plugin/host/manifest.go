// Package host manages installed trusted plugins and their child processes.
package host

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pelletier/go-toml/v2"
)

const ManifestFile = "anytty-plugin.toml"

var validID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*(\.[a-zA-Z0-9][a-zA-Z0-9_-]*)+$`)
var placementSlot = map[string]string{"sidebar": "sidebar", "statusbar": "statusbar", "floating": "overlay", "overlay": "overlay", "menu": "menu", "header": "header", "content": "content"}
var validPlacement = map[string]bool{"sidebar": true, "statusbar": true, "floating": true, "overlay": true, "menu": true, "header": true, "content": true}
var validScope = map[string]bool{"workspace": true, "active_tab": true, "active_panel": true, "global": true}
var validBehavior = map[string]bool{"activate": true, "hide": true, "close": true, "show": true, "toggle": true}
var validTargetPolicy = map[string]bool{"none": true, "active_panel": true, "focused_panel": true, "source_panel": true}

type Component struct {
	Command []string `toml:"command"`
	Start   string   `toml:"start"`
	Restart string   `toml:"restart"`
}
type Mount struct {
	ID            string `toml:"id"`
	Slot          string `toml:"slot"`
	SurfaceID     string `toml:"surface_id"`
	Placement     string `toml:"placement"`
	Scope         string `toml:"scope"`
	Hideable      bool   `toml:"hideable"`
	Closeable     bool   `toml:"closeable"`
	Renderer      string `toml:"renderer"`
	AutoMount     bool   `toml:"auto_mount"`
	OwnerSelector string `toml:"owner_selector"`
}
type Action struct {
	ID           string   `toml:"id"`
	Label        string   `toml:"label"`
	Contexts     []string `toml:"contexts"`
	DefaultKey   string   `toml:"default_key"`
	Behavior     string   `toml:"behavior"`
	TargetPolicy string   `toml:"target_policy"`
}
type Capabilities struct {
	Daemon []string `toml:"daemon"`
	TUI    []string `toml:"tui"`
}
type Manifest struct {
	ID           string       `toml:"id"`
	Version      string       `toml:"version"`
	API          string       `toml:"api"`
	Daemon       Component    `toml:"daemon"`
	TUI          Component    `toml:"tui"`
	Capabilities Capabilities `toml:"capabilities"`
	Mounts       []Mount      `toml:"mounts"`
	Actions      []Action     `toml:"actions"`
}

func ReadManifest(directory string) (Manifest, error) {
	var result Manifest
	data, err := os.ReadFile(filepath.Join(directory, ManifestFile))
	if err != nil {
		return result, err
	}
	if len(data) > 256<<10 {
		return result, errors.New("plugin manifest exceeds 256 KiB")
	}
	if err = toml.Unmarshal(data, &result); err != nil {
		return result, err
	}
	if err = result.Validate(); err != nil {
		return Manifest{}, err
	}
	return result, nil
}
func (m Manifest) Validate() error {
	if !validID.MatchString(m.ID) {
		return fmt.Errorf("invalid plugin ID %q", m.ID)
	}
	if m.API != "anytty.plugin/1" {
		return fmt.Errorf("unsupported plugin API %q", m.API)
	}
	if m.Version == "" {
		return errors.New("plugin version required")
	}
	if len(m.Daemon.Command) == 0 && len(m.TUI.Command) == 0 {
		return errors.New("plugin requires daemon or TUI component")
	}
	for name, c := range map[string]Component{"daemon": m.Daemon, "tui": m.TUI} {
		if len(c.Command) > 0 && c.Command[0] == "" {
			return fmt.Errorf("%s executable is empty", name)
		}
		if c.Start != "" && c.Start != name {
			return fmt.Errorf("invalid %s startup policy", name)
		}
		if c.Restart != "" && c.Restart != "never" && c.Restart != "on-failure" {
			return fmt.Errorf("invalid %s restart policy", name)
		}
	}
	seen := make(map[string]bool)
	for _, mount := range m.Mounts {
		if mount.ID == "" || seen[mount.ID] {
			return fmt.Errorf("missing or duplicate mount ID %q", mount.ID)
		}
		seen[mount.ID] = true
		if mount.Renderer != "declarative" && mount.Renderer != "pty" {
			return fmt.Errorf("invalid mount renderer %q", mount.Renderer)
		}
		if mount.Slot == "" && mount.Placement == "" {
			return fmt.Errorf("mount %s needs slot or placement", mount.ID)
		}
		if mount.Placement != "" && !validPlacement[mount.Placement] {
			return fmt.Errorf("invalid mount placement %q", mount.Placement)
		}
		if mount.Scope != "" && !validScope[mount.Scope] {
			return fmt.Errorf("invalid mount scope %q", mount.Scope)
		}
		if mount.Placement != "" && mount.Slot != "" && placementSlot[mount.Placement] != mount.Slot {
			return fmt.Errorf("mount %s placement %q conflicts with slot %q", mount.ID, mount.Placement, mount.Slot)
		}
		if mount.AutoMount && mount.OwnerSelector == "" {
			return fmt.Errorf("auto mount %s needs owner_selector", mount.ID)
		}
	}
	seen = make(map[string]bool)
	for _, action := range m.Actions {
		if action.ID == "" || seen[action.ID] {
			return fmt.Errorf("missing or duplicate action ID %q", action.ID)
		}
		seen[action.ID] = true
		if action.Behavior != "" && !validBehavior[action.Behavior] {
			return fmt.Errorf("invalid action behavior %q", action.Behavior)
		}
		if action.TargetPolicy != "" && !validTargetPolicy[action.TargetPolicy] {
			return fmt.Errorf("invalid action target policy %q", action.TargetPolicy)
		}
	}
	return nil
}
