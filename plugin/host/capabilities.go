package host

import (
	"fmt"
	"strings"

	"github.com/anytty/anytty/proto/apipb"
)

// Capabilities constrain this bridge's AnyTTY API. Plugins are trusted OS
// processes; this is not a filesystem, network, or same-user process sandbox.
func authorizeCommand(manifest Manifest, mode string, command *apipb.PluginCommand) error {
	capabilities := manifest.Capabilities.TUI
	if mode == "daemon" {
		capabilities = manifest.Capabilities.Daemon
	}
	has := func(required string) bool {
		for _, capability := range capabilities {
			if capability == required {
				return true
			}
		}
		return false
	}
	require := func(required string) error {
		if !has(required) {
			return fmt.Errorf("plugin %s lacks capability %s", manifest.ID, required)
		}
		return nil
	}
	if register := command.GetRegister(); register != nil && len(register.Topics) > 0 {
		if err := require("events.subscribe"); err != nil {
			return err
		}
	}
	if state := command.GetState(); state != nil {
		if state.GetPut() != nil {
			return require("state.write")
		}
		if err := require("state.read"); err != nil {
			return err
		}
		if state.GetWatch() != nil {
			return require("events.subscribe")
		}
	}
	if message := command.GetSend().GetMessage(); message != nil {
		switch {
		case message.GetReply() != nil:
			// The daemon separately validates the pending request and reply peer.
			return nil
		case message.GetInit() != nil:
			return require("ui.mounts")
		case message.GetUiQuery() != nil:
			return require("ui.read")
		case message.GetMountUpdate() != nil:
			if err := require("ui.mounts"); err != nil {
				return err
			}
			update := message.GetMountUpdate()
			slot := update.Slot
			if slot == "" && update.Placement != "" {
				slot = placementSlot[update.Placement]
			}
			for _, mount := range manifest.Mounts {
				idMatch := update.MountId == mount.ID || strings.HasPrefix(update.MountId, mount.ID+".")
				if update.Close && idMatch {
					return nil
				}
				slotMatch := slot != "" && (slot == mount.Slot || slot == placementSlot[mount.Placement])
				surfaceMatch := update.SurfaceId != "" && mount.SurfaceID != "" && update.SurfaceId == mount.SurfaceID
				scopeMatch := update.Scope == "" || mount.Scope == "" || update.Scope == mount.Scope
				placementMatch := update.Placement == "" || mount.Placement == "" || update.Placement == mount.Placement
				if idMatch && (slotMatch || surfaceMatch) && scopeMatch && placementMatch {
					declared := make(map[string]Action, len(manifest.Actions))
					for _, action := range manifest.Actions {
						declared[action.ID] = action
					}
					for _, action := range update.Actions {
						meta, ok := declared[action.GetId()]
						if !ok {
							return fmt.Errorf("undeclared plugin action %s", action.GetId())
						}
						if action.GetBehavior() != "" && action.GetBehavior() != meta.Behavior {
							return fmt.Errorf("plugin action %s changes manifest behavior", action.GetId())
						}
						if action.GetTargetPolicy() != "" && action.GetTargetPolicy() != meta.TargetPolicy {
							return fmt.Errorf("plugin action %s changes manifest target policy", action.GetId())
						}
					}
					return nil
				}
			}
			return fmt.Errorf("undeclared plugin mount %s at %s", update.MountId, update.Slot)
		case message.GetOperation() != nil:
			if message.GetOperation().GetBind() != nil {
				return require("ui.panes.bind")
			}
			return require("ui.notifications")
		default:
			return require("messages.send")
		}
	}
	return nil
}
