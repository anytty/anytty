package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

// ParsePaneMiniCommand 是 CLI mini command / command palette 到 PaneCommand 的稳定 adapter。
// 它只解析命令语义，不读取或修改 reducer-owned state。
func ParsePaneMiniCommand(text string) (state.PaneCommand, error) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return state.PaneCommand{}, fmt.Errorf("empty pane command")
	}
	if fields[0] == "pane" {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return state.PaneCommand{}, fmt.Errorf("missing pane command action")
	}
	command := state.PaneCommand{Source: state.PaneCommandSourceCLIMini}
	action := fields[0]
	args := fields[1:]
	switch action {
	case "split-right", "split-vertical":
		command.Action = state.PaneCommandSplit
		command.SplitDirection = state.SplitDirectionVertical
	case "split-down", "split-horizontal":
		command.Action = state.PaneCommandSplit
		command.SplitDirection = state.SplitDirectionHorizontal
	case "close":
		command.Action = state.PaneCommandClose
	case "kill":
		command.Action = state.PaneCommandKill
	case "close-kill":
		command.Action = state.PaneCommandCloseAndKill
	case "focus":
		command.Action = state.PaneCommandFocus
	case "focus-next":
		command.Action = state.PaneCommandFocusNext
	case "focus-prev", "focus-previous":
		command.Action = state.PaneCommandFocusPrevious
	case "zoom":
		command.Action = state.PaneCommandZoom
	case "unzoom":
		command.Action = state.PaneCommandUnzoom
	case "toggle-zoom":
		command.Action = state.PaneCommandToggleZoom
	case "resize":
		command.Action = state.PaneCommandResize
	case "set-size":
		command.Action = state.PaneCommandSetSize
	case "balance", "equalize":
		command.Action = state.PaneCommandBalance
	case "presentation":
		command.Action = state.PaneCommandSetPresentation
	default:
		return state.PaneCommand{}, fmt.Errorf("unknown pane command action %q", action)
	}
	if err := applyPaneMiniArgs(&command, args); err != nil {
		return state.PaneCommand{}, err
	}
	return command, nil
}

func PaneCommandFromHitRegion(region render.HitRegion) (state.PaneCommand, bool) {
	if region.PaneID == "" {
		return state.PaneCommand{}, false
	}
	command := state.PaneCommand{
		Target: state.PaneCommandTarget{PaneID: region.PaneID},
		Source: state.PaneCommandSourceMouse,
	}
	switch region.Kind {
	case render.HitRegionPaneResize:
		command.Action = state.PaneCommandResize
		command.ResizeDirection = state.PaneResizeRight
		command.Delta = 1
		command.ResizeSplitPath = region.SplitPath
		command.ResizeGroupCells = paneResizeGroupFromHitRegion(region)
	case render.HitRegionPaneContent:
		command.Action = state.PaneCommandFocus
	default:
		return state.PaneCommand{}, false
	}
	return command, true
}

func PaneCommandFromIntent(intent input.Intent) (state.PaneCommand, bool) {
	if intent.Kind != input.IntentPaneCommand {
		return state.PaneCommand{}, false
	}
	command, err := ParsePaneMiniCommand(intent.Command)
	if err != nil {
		return state.PaneCommand{}, false
	}
	command.Source = state.PaneCommandSourceKeyboard
	return command, true
}

func PaneModeCommand(action state.PaneCommandAction, paneID string) (state.PaneCommand, bool) {
	switch action {
	case state.PaneCommandClose,
		state.PaneCommandFocus,
		state.PaneCommandZoom,
		state.PaneCommandUnzoom,
		state.PaneCommandToggleZoom,
		state.PaneCommandBalance,
		state.PaneCommandTogglePresentation:
		return state.PaneCommand{
			Action: action,
			Target: state.PaneCommandTarget{PaneID: paneID},
			Source: state.PaneCommandSourceKeyboard,
		}, true
	default:
		return state.PaneCommand{}, false
	}
}

func ResizeModeCommand(paneID string, direction state.PaneResizeDirection, delta int) (state.PaneCommand, bool) {
	if direction != state.PaneResizeLeft && direction != state.PaneResizeRight && direction != state.PaneResizeUp && direction != state.PaneResizeDown {
		return state.PaneCommand{}, false
	}
	if delta == 0 {
		delta = 1
	}
	return state.PaneCommand{
		Action:          state.PaneCommandResize,
		Target:          state.PaneCommandTarget{PaneID: paneID},
		ResizeDirection: direction,
		Delta:           delta,
		Source:          state.PaneCommandSourceKeyboard,
	}, true
}

func applyPaneMiniArgs(command *state.PaneCommand, args []string) error {
	for _, arg := range args {
		key, value, ok := strings.Cut(arg, "=")
		if !ok {
			if err := applyPaneMiniPositional(command, arg); err != nil {
				return err
			}
			continue
		}
		switch key {
		case "pane", "pane-id":
			command.Target.PaneID = value
		case "tab":
			command.Target.TabID = value
		case "workspace":
			command.Target.WorkspaceID = value
		case "new-pane":
			command.NewPane.ID = value
		case "direction":
			command.ResizeDirection = state.PaneResizeDirection(value)
		case "delta":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid delta %q", value)
			}
			command.Delta = parsed
		case "ratio":
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("invalid ratio %q", value)
			}
			command.SizeMode = state.PaneSizeRatio
			command.Ratio = parsed
		case "cols":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid cols %q", value)
			}
			command.SizeMode = state.PaneSizeCells
			command.Cols = parsed
		case "rows":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid rows %q", value)
			}
			command.SizeMode = state.PaneSizeCells
			command.Rows = parsed
		case "presentation":
			command.Presentation = state.PanelPresentation(value)
		case "confirm":
			if value == "accepted" || value == "true" || value == "yes" {
				command.Confirm = state.PaneConfirmAccepted
			}
		default:
			return fmt.Errorf("unknown pane command argument %q", key)
		}
	}
	return nil
}

func applyPaneMiniPositional(command *state.PaneCommand, arg string) error {
	switch command.Action {
	case state.PaneCommandResize:
		switch arg {
		case "left", "right", "up", "down":
			command.ResizeDirection = state.PaneResizeDirection(arg)
			if command.Delta == 0 {
				command.Delta = 1
			}
			return nil
		}
	case state.PaneCommandSetPresentation:
		if arg == string(state.PanelPresentationCard) || arg == string(state.PanelPresentationSplitLine) {
			command.Presentation = state.PanelPresentation(arg)
			return nil
		}
	}
	if command.Target.PaneID == "" {
		command.Target.PaneID = arg
		return nil
	}
	return fmt.Errorf("unexpected pane command argument %q", arg)
}
