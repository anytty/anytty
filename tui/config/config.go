package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anytty/anytty/shared/userdirs"
	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/shortcut"
	"github.com/anytty/anytty/tui/state"
)

const DefaultFileName = "tui-v3.yaml"

const maxHistorySizeMB = 8 * 1024 * 1024

const (
	minResourceSampleIntervalMS = 100
	maxResourceSampleIntervalMS = 60_000
	minResourceHistorySamples   = 512
	maxResourceHistorySamples   = 65_536
)

const (
	DefaultWorkspaceTemplate = "[style:header-workspace] WS {{workspace | truncate 18}} [/style][style:header-spacer] [/style]"
	DefaultTabTemplate       = "{{if active}}[style:header-active-edge] [/style][style:header-active-marker]{{marker}}[/style][style:header-active-index] {{index}}[/style][style:header-active-title] {{title | truncate 14}} [/style][action:tab.close][style:header-active-close]{{close_icon}}[/style][/action][style:header-active-edge] [/style]{{else}}[style:header-spacer] [/style][style:header-spacer]{{marker}}[/style][style:header-inactive-index] {{index}}[/style][style:header-inactive-title] {{title | truncate 14}} [/style][action:tab.close][style:header-inactive-close]{{close_icon}}[/style][/action][style:header-spacer] [/style]{{end}}"
)

func DefaultPath() string {
	return filepath.Join(userdirs.ConfigHome(), "anytty", DefaultFileName)
}

func Default() state.TUIConfigStore {
	return state.TUIConfigStore{
		Version: 1,
		Daemon: state.DaemonConfig{
			OutputBuffer: state.DaemonOutputBufferConfig{
				CapacityBytes:       32 << 20,
				Overflow:            "block",
				ResidentBudgetBytes: 512 << 20,
			},
			History: state.DaemonHistoryConfig{
				MaxSizeMB:        512,
				MaxAgeDays:       0,
				Compression:      "zstd",
				CompressionLevel: "fast",
			},
			ResourceSampling: state.DaemonResourceSamplingConfig{
				IntervalMS: 500,
				MaxSamples: 512,
			},
		},
		Profile: "default",
		Theme: state.TUIThemeConfig{
			Mode:    "dark",
			Palette: "host",
		},
		Chrome: state.TUIChromeConfig{
			Header:            true,
			Footer:            true,
			PanelPresentation: "split-line",
			Picker: state.TUIPickerChromeConfig{
				Presentation:   "card",
				Width:          "adaptive",
				Density:        "compact",
				EndpointTabs:   "underline",
				EndpointStatus: state.DefaultTUIPickerEndpointStatusConfig(),
			},
			TabCreateIcon:     "+",
			TabCreateTemplate: "",
			WorkspaceTemplate: DefaultWorkspaceTemplate,
			TabTemplate:       DefaultTabTemplate,
			PaneTitleTemplate: "",
		},
		Footer: state.TUIFooterConfig{
			Templates: state.TUIFooterTemplatesConfig{
				ModeBadge:                "{{mode_icon}} {{mode_label}}",
				Key:                      "{{key}}",
				Action:                   "{{key}} {{icon}} {{label}}",
				Separator:                " │ ",
				WorkspaceSummary:         "ws:{{workspace}}",
				FloatingSummary:          "float:{{count}}",
				FloatingCollapsedSummary: "collapsed:{{count}}",
				TerminalsSummary:         "terminals:{{count}}",
				TabsSummary:              "tabs:{{count}}",
				PanesSummary:             "panes:{{count}}",
				KeylockOn:                "LOCK",
			},
		},
		Interaction: state.TUIInteractionConfig{
			Mouse:                         true,
			StickyPrefixTimeoutMS:         3000,
			ShortcutPassthroughIntervalMS: 1000,
			ConfirmDestructive:            true,
			ClipboardHistory: state.TUIClipboardHistoryConfig{
				MaxItems:          200,
				NameWidth:         34,
				PreviewWidthRatio: 0.68,
			},
			Picker: state.TUIPickerConfig{
				FuzzyMatch:       "subsequence",
				HighlightMatches: true,
			},
		},
		Terminal: state.TUITerminalConfig{
			AutoTakeOwner: true,
		},
	}
}

func Load(path string, lookupEnv func(string) string) (state.TUIConfigStore, error) {
	cfg := Default()
	explicit := strings.TrimSpace(path) != ""
	if !explicit {
		path = DefaultPath()
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) || explicit {
				return state.TUIConfigStore{}, err
			}
		} else {
			parsed, err := Parse(data)
			if err != nil {
				return state.TUIConfigStore{}, fmt.Errorf("parse tui-v3 config %q: %w", path, err)
			}
			cfg = parsed
		}
	}
	if lookupEnv != nil {
		if err := applyEnv(&cfg, lookupEnv); err != nil {
			return state.TUIConfigStore{}, err
		}
	}
	if err := Validate(cfg); err != nil {
		return state.TUIConfigStore{}, err
	}
	return cfg, nil
}

func Parse(data []byte) (state.TUIConfigStore, error) {
	cfg := Default()
	if strings.TrimSpace(string(data)) == "" {
		return cfg, nil
	}
	stack := map[int]string{}
	sectionPaths := map[int]string{}
	sectionLines := map[int]int{}
	sectionHasChild := map[int]bool{}
	for lineNo, raw := range strings.Split(string(data), "\n") {
		raw = strings.TrimRight(raw, "\r")
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if indent%2 != 0 {
			return state.TUIConfigStore{}, fmt.Errorf("line %d: indentation must use two-space levels", lineNo+1)
		}
		level := indent / 2
		line = stripInlineComment(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "-") {
			return state.TUIConfigStore{}, fmt.Errorf("line %d: list values are not supported in tui-v3 config", lineNo+1)
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return state.TUIConfigStore{}, fmt.Errorf("line %d: expected key: value", lineNo+1)
		}
		key = strings.TrimSpace(key)
		parsedKey, err := parseConfigKey(key)
		if err != nil {
			return state.TUIConfigStore{}, fmt.Errorf("line %d: %w", lineNo+1, err)
		}
		key = parsedKey
		value = strings.TrimSpace(value)
		if key == "" {
			return state.TUIConfigStore{}, fmt.Errorf("line %d: empty key", lineNo+1)
		}
		path := joinPath(stack, level, key)
		for parentLevel := 0; parentLevel < level; parentLevel++ {
			if sectionPaths[parentLevel] != "" {
				sectionHasChild[parentLevel] = true
			}
		}
		for existing := range sectionPaths {
			if existing >= level {
				if err := validateClosedConfigSection(sectionPaths[existing], sectionLines[existing], sectionHasChild[existing]); err != nil {
					return state.TUIConfigStore{}, err
				}
				delete(sectionPaths, existing)
				delete(sectionLines, existing)
				delete(sectionHasChild, existing)
			}
		}
		if value == "" {
			if !knownSection(path) {
				return state.TUIConfigStore{}, fmt.Errorf("line %d: unknown section %q", lineNo+1, path)
			}
			if path == "tui.shortcuts" || strings.HasPrefix(path, "tui.shortcuts.") {
				cfg.Shortcuts.Configured = true
			}
			sectionPaths[level] = path
			sectionLines[level] = lineNo + 1
			stack[level] = key
			for existing := range stack {
				if existing > level {
					delete(stack, existing)
				}
			}
			continue
		}
		if value == "{}" {
			handled, err := applyEmptyMap(&cfg, path)
			if err != nil {
				return state.TUIConfigStore{}, fmt.Errorf("line %d: %s: %w", lineNo+1, path, err)
			}
			if !handled {
				return state.TUIConfigStore{}, fmt.Errorf("line %d: unknown field %q", lineNo+1, path)
			}
			continue
		}
		setter, ok := scalarSetters[path]
		if !ok {
			parsedValue, err := parseScalar(value)
			if err != nil {
				return state.TUIConfigStore{}, fmt.Errorf("line %d: %w", lineNo+1, err)
			}
			handled, err := setDynamicScalar(&cfg, path, parsedValue)
			if err != nil {
				return state.TUIConfigStore{}, fmt.Errorf("line %d: %s: %w", lineNo+1, path, err)
			}
			if !handled {
				return state.TUIConfigStore{}, fmt.Errorf("line %d: unknown field %q", lineNo+1, path)
			}
			continue
		}
		parsedValue, err := parseScalar(value)
		if err != nil {
			return state.TUIConfigStore{}, fmt.Errorf("line %d: %w", lineNo+1, err)
		}
		if err := setter(&cfg, parsedValue); err != nil {
			return state.TUIConfigStore{}, fmt.Errorf("line %d: %s: %w", lineNo+1, path, err)
		}
	}
	for level, path := range sectionPaths {
		if err := validateClosedConfigSection(path, sectionLines[level], sectionHasChild[level]); err != nil {
			return state.TUIConfigStore{}, err
		}
	}
	if err := Validate(cfg); err != nil {
		return state.TUIConfigStore{}, err
	}
	return cfg, nil
}

func validateClosedConfigSection(path string, line int, hasChild bool) error {
	if hasChild || (path != "tui.shortcuts" && !strings.HasPrefix(path, "tui.shortcuts.")) {
		return nil
	}
	return fmt.Errorf("line %d: %s must be a map; use %s: {} for an empty map", line, path, path[strings.LastIndex(path, ".")+1:])
}

func applyEmptyMap(cfg *state.TUIConfigStore, path string) (bool, error) {
	switch path {
	case "tui.shortcuts":
		return true, nil
	case "tui.shortcuts.actions":
		cfg.Shortcuts.Configured = true
		if cfg.Shortcuts.Actions == nil {
			cfg.Shortcuts.Actions = map[string]state.TUIShortcutActionConfig{}
		}
		return true, nil
	}
	const prefix = "tui.shortcuts."
	if !strings.HasPrefix(path, prefix) {
		return false, nil
	}
	sceneName := strings.TrimPrefix(path, prefix)
	if !builtinShortcutScene(sceneName) {
		return false, nil
	}
	cfg.Shortcuts.Configured = true
	if cfg.Shortcuts.Scenes == nil {
		cfg.Shortcuts.Scenes = map[string]state.TUIShortcutSceneConfig{}
	}
	cfg.Shortcuts.Scenes[sceneName] = state.TUIShortcutSceneConfig{Bindings: map[string]state.TUIShortcutBindingConfig{}}
	return true, nil
}

func joinPath(stack map[int]string, level int, key string) string {
	parts := make([]string, 0, level+1)
	for i := 0; i < level; i++ {
		if value := stack[i]; value != "" {
			parts = append(parts, value)
		}
	}
	parts = append(parts, key)
	return strings.Join(parts, ".")
}

func stripInlineComment(value string) string {
	var quote rune
	escaped := false
	for index, r := range value {
		if escaped {
			escaped = false
			continue
		}
		if quote == '"' && r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			continue
		}
		if r == '#' {
			return strings.TrimSpace(value[:index])
		}
	}
	return strings.TrimSpace(value)
}

func parseScalar(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "\"") {
		out, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("invalid quoted string %q", value)
		}
		return out, nil
	}
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'"), nil
	}
	return strings.TrimSpace(value), nil
}

func parseConfigKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "\"") || strings.HasPrefix(value, "'") {
		parsed, err := parseScalar(value)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(parsed), nil
	}
	return value, nil
}

func knownSection(path string) bool {
	switch path {
	case "tui",
		"daemon",
		"daemon.output_buffer",
		"daemon.history",
		"daemon.resource_sampling",
		"tui.theme",
		"tui.theme.border",
		"tui.theme.surface",
		"tui.chrome",
		"tui.chrome.picker",
		"tui.chrome.picker.endpoint_status",
		"tui.chrome.picker.endpoint_status.unknown",
		"tui.chrome.picker.endpoint_status.connected",
		"tui.chrome.picker.endpoint_status.connecting",
		"tui.chrome.picker.endpoint_status.idle",
		"tui.chrome.picker.endpoint_status.disabled",
		"tui.chrome.picker.endpoint_status.offline",
		"tui.chrome.picker.endpoint_status.reconnect_required",
		"tui.chrome.picker.endpoint_status.unregistered",
		"tui.chrome.pane_glyphs",
		"tui.footer",
		"tui.footer.templates",
		"tui.footer.modes",
		"tui.interaction",
		"tui.interaction.clipboard_history",
		"tui.interaction.picker",
		"tui.terminal",
		"tui.shortcuts",
		"tui.shortcuts.actions":
		return true
	default:
		return knownFooterDynamicSection(path) || knownShortcutDynamicSection(path)
	}
}

func knownFooterDynamicSection(path string) bool {
	const prefix = "tui.footer.modes."
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" || strings.Contains(rest, ".") {
		return false
	}
	return validFooterConfigName(rest)
}

func knownShortcutDynamicSection(path string) bool {
	if strings.HasPrefix(path, "tui.shortcuts.actions.") {
		rest := strings.TrimPrefix(path, "tui.shortcuts.actions.")
		return validShortcutActionID(rest)
	}
	if !strings.HasPrefix(path, "tui.shortcuts.") {
		return false
	}
	rest := strings.TrimPrefix(path, "tui.shortcuts.")
	scene, key, ok := strings.Cut(rest, ".")
	if !ok {
		return builtinShortcutScene(scene)
	}
	if scene == "actions" || !builtinShortcutScene(scene) {
		return false
	}
	for _, suffix := range []string{".action", ".label", ".show"} {
		if base, ok := strings.CutSuffix(key, suffix); ok {
			return validShortcutKeyExpression(base)
		}
	}
	return validShortcutKeyExpression(key)
}

type scalarSetter func(*state.TUIConfigStore, string) error

var scalarSetters = map[string]scalarSetter{
	"version": setInt(func(cfg *state.TUIConfigStore, value int) { cfg.Version = value }),
	"daemon.output_buffer.capacity_bytes": setInt64(func(cfg *state.TUIConfigStore, value int64) {
		cfg.Daemon.OutputBuffer.CapacityBytes = value
	}),
	"daemon.output_buffer.overflow": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Daemon.OutputBuffer.Overflow = value
	}),
	"daemon.output_buffer.resident_budget_bytes": setInt64(func(cfg *state.TUIConfigStore, value int64) {
		cfg.Daemon.OutputBuffer.ResidentBudgetBytes = value
	}),
	"daemon.history.max_size_mb":       setInt(func(cfg *state.TUIConfigStore, value int) { cfg.Daemon.History.MaxSizeMB = value }),
	"daemon.history.max_age_days":      setInt(func(cfg *state.TUIConfigStore, value int) { cfg.Daemon.History.MaxAgeDays = value }),
	"daemon.history.compression":       setString(func(cfg *state.TUIConfigStore, value string) { cfg.Daemon.History.Compression = value }),
	"daemon.history.compression_level": setString(func(cfg *state.TUIConfigStore, value string) { cfg.Daemon.History.CompressionLevel = value }),
	"daemon.resource_sampling.interval_ms": setInt(func(cfg *state.TUIConfigStore, value int) {
		cfg.Daemon.ResourceSampling.IntervalMS = value
	}),
	"daemon.resource_sampling.max_samples": setInt(func(cfg *state.TUIConfigStore, value int) {
		cfg.Daemon.ResourceSampling.MaxSamples = value
	}),
	"tui.profile":                     setString(func(cfg *state.TUIConfigStore, value string) { cfg.Profile = value }),
	"tui.theme.mode":                  setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Mode = value }),
	"tui.theme.palette":               setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Palette = value }),
	"tui.theme.primary":               setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Primary = value }),
	"tui.theme.secondary":             setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Secondary = value }),
	"tui.theme.foreground":            setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Foreground = value }),
	"tui.theme.background":            setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Background = value }),
	"tui.theme.muted":                 setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Muted = value }),
	"tui.theme.success":               setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Success = value }),
	"tui.theme.warning":               setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Warning = value }),
	"tui.theme.danger":                setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Danger = value }),
	"tui.theme.info":                  setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Info = value }),
	"tui.theme.border.panel":          setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Border.Panel = value }),
	"tui.theme.border.active":         setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Border.Active = value }),
	"tui.theme.border.inactive":       setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Border.Inactive = value }),
	"tui.theme.border.muted":          setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Border.Muted = value }),
	"tui.theme.surface.chrome_bg":     setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Surface.ChromeBG = value }),
	"tui.theme.surface.status_bg":     setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Surface.StatusBG = value }),
	"tui.theme.surface.overlay_bg":    setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Surface.OverlayBG = value }),
	"tui.theme.surface.toast_bg":      setString(func(cfg *state.TUIConfigStore, value string) { cfg.Theme.Surface.ToastBG = value }),
	"tui.chrome.header":               setBool(func(cfg *state.TUIConfigStore, value bool) { cfg.Chrome.Header = value }),
	"tui.chrome.footer":               setBool(func(cfg *state.TUIConfigStore, value bool) { cfg.Chrome.Footer = value }),
	"tui.chrome.panel_presentation":   setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PanelPresentation = value }),
	"tui.chrome.picker.presentation":  setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.Picker.Presentation = value }),
	"tui.chrome.picker.width":         setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.Picker.Width = value }),
	"tui.chrome.picker.density":       setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.Picker.Density = value }),
	"tui.chrome.picker.endpoint_tabs": setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.Picker.EndpointTabs = value }),
	"tui.chrome.picker.endpoint_status.unknown.glyph": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Unknown.Glyph = value
	}),
	"tui.chrome.picker.endpoint_status.unknown.style": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Unknown.Style = value
	}),
	"tui.chrome.picker.endpoint_status.connected.glyph": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Connected.Glyph = value
	}),
	"tui.chrome.picker.endpoint_status.connected.style": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Connected.Style = value
	}),
	"tui.chrome.picker.endpoint_status.connecting.glyph": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Connecting.Glyph = value
	}),
	"tui.chrome.picker.endpoint_status.connecting.style": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Connecting.Style = value
	}),
	"tui.chrome.picker.endpoint_status.idle.glyph": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Idle.Glyph = value
	}),
	"tui.chrome.picker.endpoint_status.idle.style": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Idle.Style = value
	}),
	"tui.chrome.picker.endpoint_status.disabled.glyph": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Disabled.Glyph = value
	}),
	"tui.chrome.picker.endpoint_status.disabled.style": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Disabled.Style = value
	}),
	"tui.chrome.picker.endpoint_status.offline.glyph": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Offline.Glyph = value
	}),
	"tui.chrome.picker.endpoint_status.offline.style": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Offline.Style = value
	}),
	"tui.chrome.picker.endpoint_status.reconnect_required.glyph": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.ReconnectRequired.Glyph = value
	}),
	"tui.chrome.picker.endpoint_status.reconnect_required.style": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.ReconnectRequired.Style = value
	}),
	"tui.chrome.picker.endpoint_status.unregistered.glyph": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Unregistered.Glyph = value
	}),
	"tui.chrome.picker.endpoint_status.unregistered.style": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.Picker.EndpointStatus.Unregistered.Style = value
	}),
	"tui.chrome.tab_create_icon": setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.TabCreateIcon = value }),
	"tui.chrome.tab_create_template": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.TabCreateTemplate = value
	}),
	"tui.chrome.workspace_template": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.WorkspaceTemplate = value
	}),
	"tui.chrome.tab_template": setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.TabTemplate = value }),
	"tui.chrome.pane_title_template": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneTitleTemplate = value
	}),
	"tui.chrome.pane_glyphs.action_left": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.ActionLeft = value
		cfg.Chrome.PaneGlyphs.ActionLeftSet = true
	}),
	"tui.chrome.pane_glyphs.action_right": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.ActionRight = value
		cfg.Chrome.PaneGlyphs.ActionRightSet = true
	}),
	"tui.chrome.pane_glyphs.action_separator": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.ActionSeparator = value
		cfg.Chrome.PaneGlyphs.ActionSeparatorSet = true
	}),
	"tui.chrome.pane_glyphs.action_group_left": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.ActionGroupLeft = value
		cfg.Chrome.PaneGlyphs.ActionGroupLeftSet = true
	}),
	"tui.chrome.pane_glyphs.action_group_right": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.ActionGroupRight = value
		cfg.Chrome.PaneGlyphs.ActionGroupRightSet = true
	}),
	"tui.chrome.pane_glyphs.owner_left": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.OwnerLeft = value
		cfg.Chrome.PaneGlyphs.OwnerLeftSet = true
	}),
	"tui.chrome.pane_glyphs.owner_right": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.OwnerRight = value
		cfg.Chrome.PaneGlyphs.OwnerRightSet = true
	}),
	"tui.chrome.pane_glyphs.owner": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.Owner = value
		cfg.Chrome.PaneGlyphs.OwnerSet = true
	}),
	"tui.chrome.pane_glyphs.owner_pending": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.OwnerPending = value
		cfg.Chrome.PaneGlyphs.OwnerPendingSet = true
	}),
	"tui.chrome.pane_glyphs.take_owner": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.TakeOwner = value
		cfg.Chrome.PaneGlyphs.TakeOwnerSet = true
	}),
	"tui.chrome.pane_glyphs.zoom":              setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.Zoom = value }),
	"tui.chrome.pane_glyphs.unzoom":            setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.Unzoom = value }),
	"tui.chrome.pane_glyphs.split_vertical":    setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.SplitVertical = value }),
	"tui.chrome.pane_glyphs.split_horizontal":  setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.SplitHorizontal = value }),
	"tui.chrome.pane_glyphs.close":             setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.Close = value }),
	"tui.chrome.pane_glyphs.size_lock":         setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.SizeLock = value }),
	"tui.chrome.pane_glyphs.size_unlock":       setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.SizeUnlock = value }),
	"tui.chrome.pane_glyphs.center_floating":   setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.CenterFloating = value }),
	"tui.chrome.pane_glyphs.collapse_floating": setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.CollapseFloating = value }),
	"tui.chrome.pane_glyphs.running":           setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.Running = value }),
	"tui.chrome.pane_glyphs.waiting":           setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.Waiting = value }),
	"tui.chrome.pane_glyphs.exited":            setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.Exited = value }),
	"tui.chrome.pane_glyphs.killed":            setString(func(cfg *state.TUIConfigStore, value string) { cfg.Chrome.PaneGlyphs.Killed = value }),
	"tui.chrome.pane_glyphs.overflow_left": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.OverflowLeft = value
		cfg.Chrome.PaneGlyphs.OverflowLeftSet = true
	}),
	"tui.chrome.pane_glyphs.overflow_right": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.OverflowRight = value
		cfg.Chrome.PaneGlyphs.OverflowRightSet = true
	}),
	"tui.chrome.pane_glyphs.overflow_top": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.OverflowTop = value
		cfg.Chrome.PaneGlyphs.OverflowTopSet = true
	}),
	"tui.chrome.pane_glyphs.overflow_bottom": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.OverflowBottom = value
		cfg.Chrome.PaneGlyphs.OverflowBottomSet = true
	}),
	"tui.chrome.pane_glyphs.overflow_style": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.OverflowStyle = value
		cfg.Chrome.PaneGlyphs.OverflowStyleSet = true
	}),
	"tui.chrome.pane_glyphs.extent_placeholder": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.ExtentPlaceholder = value
		cfg.Chrome.PaneGlyphs.ExtentPlaceholderSet = true
	}),
	"tui.chrome.pane_glyphs.extent_placeholder_style": setString(func(cfg *state.TUIConfigStore, value string) {
		cfg.Chrome.PaneGlyphs.ExtentPlaceholderStyle = value
		cfg.Chrome.PaneGlyphs.ExtentPlaceholderStyleSet = true
	}),
	"tui.footer.templates.mode_badge":                  setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.ModeBadge = value }),
	"tui.footer.templates.key":                         setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.Key = value }),
	"tui.footer.templates.action":                      setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.Action = value }),
	"tui.footer.templates.separator":                   setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.Separator = value }),
	"tui.footer.templates.workspace_summary":           setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.WorkspaceSummary = value }),
	"tui.footer.templates.floating_summary":            setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.FloatingSummary = value }),
	"tui.footer.templates.floating_collapsed_summary":  setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.FloatingCollapsedSummary = value }),
	"tui.footer.templates.terminals_summary":           setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.TerminalsSummary = value }),
	"tui.footer.templates.tabs_summary":                setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.TabsSummary = value }),
	"tui.footer.templates.panes_summary":               setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.PanesSummary = value }),
	"tui.footer.templates.keylock_on":                  setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.KeylockOn = value }),
	"tui.footer.templates.keylock_off":                 setString(func(cfg *state.TUIConfigStore, value string) { cfg.Footer.Templates.KeylockOff = value }),
	"tui.interaction.mouse":                            setBool(func(cfg *state.TUIConfigStore, value bool) { cfg.Interaction.Mouse = value }),
	"tui.interaction.sticky_prefix_timeout_ms":         setInt(func(cfg *state.TUIConfigStore, value int) { cfg.Interaction.StickyPrefixTimeoutMS = value }),
	"tui.interaction.shortcut_passthrough_interval_ms": setInt(func(cfg *state.TUIConfigStore, value int) { cfg.Interaction.ShortcutPassthroughIntervalMS = value }),
	"tui.interaction.confirm_destructive":              setBool(func(cfg *state.TUIConfigStore, value bool) { cfg.Interaction.ConfirmDestructive = value }),
	"tui.interaction.clipboard_history.max_items":      setInt(func(cfg *state.TUIConfigStore, value int) { cfg.Interaction.ClipboardHistory.MaxItems = value }),
	"tui.interaction.clipboard_history.name_width": setInt(func(cfg *state.TUIConfigStore, value int) {
		cfg.Interaction.ClipboardHistory.NameWidth = value
	}),
	"tui.interaction.clipboard_history.preview_width_ratio": setFloat(func(cfg *state.TUIConfigStore, value float64) {
		cfg.Interaction.ClipboardHistory.PreviewWidthRatio = value
	}),
	"tui.interaction.picker.fuzzy_match":       setString(func(cfg *state.TUIConfigStore, value string) { cfg.Interaction.Picker.FuzzyMatch = value }),
	"tui.interaction.picker.highlight_matches": setBool(func(cfg *state.TUIConfigStore, value bool) { cfg.Interaction.Picker.HighlightMatches = value }),
	"tui.terminal.auto_take_owner":             setBool(func(cfg *state.TUIConfigStore, value bool) { cfg.Terminal.AutoTakeOwner = value }),
}

func setDynamicScalar(cfg *state.TUIConfigStore, path string, value string) (bool, error) {
	if handled, err := setFooterDynamicScalar(cfg, path, value); handled || err != nil {
		return handled, err
	}
	return setShortcutsDynamicScalar(cfg, path, value)
}

func setFooterDynamicScalar(cfg *state.TUIConfigStore, path string, value string) (bool, error) {
	if strings.HasPrefix(path, "tui.footer.modes.") {
		rest := strings.TrimPrefix(path, "tui.footer.modes.")
		name, field, ok := strings.Cut(rest, ".")
		if !ok || !validFooterConfigName(name) {
			return false, nil
		}
		mode := cfg.Footer.Modes[name]
		switch field {
		case "icon":
			mode.Icon = value
		case "label":
			mode.Label = value
		case "style":
			mode.Style = value
		default:
			return false, nil
		}
		if cfg.Footer.Modes == nil {
			cfg.Footer.Modes = map[string]state.TUIFooterModeConfig{}
		}
		cfg.Footer.Modes[name] = mode
		return true, nil
	}
	return false, nil
}

func setShortcutsDynamicScalar(cfg *state.TUIConfigStore, path string, value string) (bool, error) {
	if strings.HasPrefix(path, "tui.shortcuts.actions.") {
		cfg.Shortcuts.Configured = true
		rest := strings.TrimPrefix(path, "tui.shortcuts.actions.")
		actionID := ""
		field := ""
		if base, ok := strings.CutSuffix(rest, ".label"); ok {
			actionID, field = base, "label"
		} else if base, ok := strings.CutSuffix(rest, ".style"); ok {
			actionID, field = base, "style"
		}
		if actionID == "" || !validShortcutActionID(actionID) {
			return false, nil
		}
		_, invocation, _, ok := shortcut.PolicyForSource(actionID)
		if !ok {
			return false, nil
		}
		canonicalID := string(invocation.ID)
		if cfg.Shortcuts.Actions == nil {
			cfg.Shortcuts.Actions = map[string]state.TUIShortcutActionConfig{}
		}
		action := cfg.Shortcuts.Actions[canonicalID]
		switch field {
		case "label":
			if action.Label != "" {
				return true, fmt.Errorf("duplicate shortcut action label for canonical action %q", canonicalID)
			}
			action.Label = value
		case "style":
			if action.Style != "" {
				return true, fmt.Errorf("duplicate shortcut action style for canonical action %q", canonicalID)
			}
			action.Style = value
		}
		cfg.Shortcuts.Actions[canonicalID] = action
		return true, nil
	}
	if !strings.HasPrefix(path, "tui.shortcuts.") {
		return false, nil
	}
	cfg.Shortcuts.Configured = true
	rest := strings.TrimPrefix(path, "tui.shortcuts.")
	sceneName, tail, ok := strings.Cut(rest, ".")
	if !ok || sceneName == "actions" || !builtinShortcutScene(sceneName) {
		return false, nil
	}
	key := tail
	field := "action"
	if base, ok := strings.CutSuffix(tail, ".action"); ok {
		key = base
		field = "action"
	} else if base, ok := strings.CutSuffix(tail, ".label"); ok {
		key = base
		field = "label"
	} else if base, ok := strings.CutSuffix(tail, ".show"); ok {
		key = base
		field = "show"
	}
	expandedKeys, ranged, ok := expandShortcutKeyExpression(key)
	if !ok {
		if strings.Contains(key, "...") {
			return false, fmt.Errorf("invalid shortcut key range %q; expected an ascending single-digit range such as [1...9]", key)
		}
		return false, nil
	}
	if cfg.Shortcuts.Scenes == nil {
		cfg.Shortcuts.Scenes = map[string]state.TUIShortcutSceneConfig{}
	}
	scene := cfg.Shortcuts.Scenes[sceneName]
	if scene.Bindings == nil {
		scene.Bindings = map[string]state.TUIShortcutBindingConfig{}
	}
	scene.KeyOrder = appendShortcutKeyOrder(scene.KeyOrder, expandedKeys)
	for _, expanded := range expandedKeys {
		binding := scene.Bindings[expanded]
		switch field {
		case "action":
			if strings.TrimSpace(binding.Action) != "" {
				return false, fmt.Errorf("duplicate shortcut key %q in scene %q", expanded, sceneName)
			}
			if !ranged && strings.Contains(value, "{key}") {
				return false, fmt.Errorf("action placeholder {key} requires a shortcut key range")
			}
			binding.Action = strings.ReplaceAll(value, "{key}", shortcutRangeValue(expanded))
		case "label":
			binding.Label = value
		case "show":
			show, err := parseShortcutShow(value)
			if err != nil {
				return false, err
			}
			binding.Show = &show
		default:
			return false, nil
		}
		scene.Bindings[expanded] = binding
	}
	cfg.Shortcuts.Scenes[sceneName] = scene
	return true, nil
}

func parseShortcutShow(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("show expects boolean true or false, got %q", value)
	}
}

func appendShortcutKeyOrder(order []string, keys []string) []string {
	seen := make(map[string]bool, len(order)+len(keys))
	for _, existing := range order {
		seen[existing] = true
	}
	for _, key := range keys {
		if !seen[key] {
			order = append(order, key)
			seen[key] = true
		}
	}
	return order
}

func validShortcutKeyExpression(value string) bool {
	_, _, ok := expandShortcutKeyExpression(value)
	return ok
}

// expandShortcutKeyExpression 只在配置边界展开升序单数字范围；运行时 catalog 继续只持有具体 key。
func expandShortcutKeyExpression(value string) ([]string, bool, bool) {
	value = strings.TrimSpace(value)
	if validShortcutKey(value) {
		return []string{value}, false, true
	}
	if !strings.Contains(value, "...") {
		return nil, false, false
	}
	open := strings.Index(value, "[")
	if open < 0 || strings.Count(value, "[") != 1 || strings.Count(value, "]") != 1 || !strings.HasSuffix(value, "]") {
		return nil, false, false
	}
	inside := value[open+1 : len(value)-1]
	parts := strings.Split(inside, "...")
	if len(parts) != 2 || len(parts[0]) != 1 || len(parts[1]) != 1 || parts[0][0] < '0' || parts[0][0] > '9' || parts[1][0] < '0' || parts[1][0] > '9' || parts[0][0] > parts[1][0] {
		return nil, false, false
	}
	prefix := value[:open]
	if strings.HasSuffix(prefix, "+") {
		prefix = strings.TrimSuffix(prefix, "+") + "-"
	}
	keys := make([]string, 0, int(parts[1][0]-parts[0][0])+1)
	for digit := parts[0][0]; digit <= parts[1][0]; digit++ {
		key := prefix + string(digit)
		if !validShortcutKey(key) {
			return nil, false, false
		}
		keys = append(keys, key)
	}
	return keys, true, true
}

func shortcutRangeValue(expandedKey string) string {
	if expandedKey == "" {
		return ""
	}
	return expandedKey[len(expandedKey)-1:]
}

func validFooterConfigName(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func setString(assign func(*state.TUIConfigStore, string)) scalarSetter {
	return func(cfg *state.TUIConfigStore, value string) error {
		assign(cfg, value)
		return nil
	}
}

func setBool(assign func(*state.TUIConfigStore, bool)) scalarSetter {
	return func(cfg *state.TUIConfigStore, value string) error {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true":
			assign(cfg, true)
			return nil
		case "false":
			assign(cfg, false)
			return nil
		default:
			return fmt.Errorf("expected boolean true or false, got %q", value)
		}
	}
}

func setInt(assign func(*state.TUIConfigStore, int)) scalarSetter {
	return func(cfg *state.TUIConfigStore, value string) error {
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return fmt.Errorf("expected integer, got %q", value)
		}
		assign(cfg, parsed)
		return nil
	}
}

func setInt64(assign func(*state.TUIConfigStore, int64)) scalarSetter {
	return func(cfg *state.TUIConfigStore, value string) error {
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return fmt.Errorf("expected integer, got %q", value)
		}
		assign(cfg, parsed)
		return nil
	}
}

func setFloat(assign func(*state.TUIConfigStore, float64)) scalarSetter {
	return func(cfg *state.TUIConfigStore, value string) error {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return fmt.Errorf("expected number, got %q", value)
		}
		assign(cfg, parsed)
		return nil
	}
}

func applyEnv(cfg *state.TUIConfigStore, lookup func(string) string) error {
	for env, path := range envScalarPaths {
		value := strings.TrimSpace(lookup(env))
		if value == "" {
			continue
		}
		setter, ok := scalarSetters[path]
		if !ok {
			return fmt.Errorf("unknown env binding %s -> %s", env, path)
		}
		if err := setter(cfg, value); err != nil {
			return fmt.Errorf("%s: %w", env, err)
		}
	}
	return nil
}

var envScalarPaths = map[string]string{
	"ANYTTY_OUTPUT_BUFFER_CAPACITY_BYTES":         "daemon.output_buffer.capacity_bytes",
	"ANYTTY_OUTPUT_BUFFER_OVERFLOW":               "daemon.output_buffer.overflow",
	"ANYTTY_OUTPUT_RESIDENT_BUDGET_BYTES":         "daemon.output_buffer.resident_budget_bytes",
	"ANYTTY_HISTORY_MAX_SIZE_MB":                  "daemon.history.max_size_mb",
	"ANYTTY_HISTORY_MAX_AGE_DAYS":                 "daemon.history.max_age_days",
	"ANYTTY_HISTORY_COMPRESSION":                  "daemon.history.compression",
	"ANYTTY_HISTORY_COMPRESSION_LEVEL":            "daemon.history.compression_level",
	"ANYTTY_RESOURCE_SAMPLING_INTERVAL_MS":        "daemon.resource_sampling.interval_ms",
	"ANYTTY_RESOURCE_SAMPLING_MAX_SAMPLES":        "daemon.resource_sampling.max_samples",
	"ANYTTY_TUI_THEME_MODE":                       "tui.theme.mode",
	"ANYTTY_TUI_THEME_PALETTE":                    "tui.theme.palette",
	"ANYTTY_TUI_THEME_PRIMARY":                    "tui.theme.primary",
	"ANYTTY_TUI_THEME_SECONDARY":                  "tui.theme.secondary",
	"ANYTTY_TUI_THEME_FOREGROUND":                 "tui.theme.foreground",
	"ANYTTY_TUI_THEME_BACKGROUND":                 "tui.theme.background",
	"ANYTTY_TUI_THEME_MUTED":                      "tui.theme.muted",
	"ANYTTY_TUI_THEME_SUCCESS":                    "tui.theme.success",
	"ANYTTY_TUI_THEME_WARNING":                    "tui.theme.warning",
	"ANYTTY_TUI_THEME_DANGER":                     "tui.theme.danger",
	"ANYTTY_TUI_THEME_INFO":                       "tui.theme.info",
	"ANYTTY_TUI_CHROME_HEADER":                    "tui.chrome.header",
	"ANYTTY_TUI_CHROME_FOOTER":                    "tui.chrome.footer",
	"ANYTTY_TUI_CHROME_PANEL_PRESENTATION":        "tui.chrome.panel_presentation",
	"ANYTTY_TUI_CHROME_PICKER_PRESENTATION":       "tui.chrome.picker.presentation",
	"ANYTTY_TUI_CHROME_PICKER_WIDTH":              "tui.chrome.picker.width",
	"ANYTTY_TUI_CHROME_PICKER_DENSITY":            "tui.chrome.picker.density",
	"ANYTTY_TUI_CHROME_PICKER_ENDPOINT_TABS":      "tui.chrome.picker.endpoint_tabs",
	"ANYTTY_TUI_CHROME_TAB_CREATE_TEMPLATE":       "tui.chrome.tab_create_template",
	"ANYTTY_TUI_CHROME_PANE_TITLE_TEMPLATE":       "tui.chrome.pane_title_template",
	"ANYTTY_TUI_INTERACTION_MOUSE":                "tui.interaction.mouse",
	"ANYTTY_TUI_STICKY_PREFIX_TIMEOUT_MS":         "tui.interaction.sticky_prefix_timeout_ms",
	"ANYTTY_TUI_SHORTCUT_PASSTHROUGH_INTERVAL_MS": "tui.interaction.shortcut_passthrough_interval_ms",
	"ANYTTY_TUI_CONFIRM_DESTRUCTIVE":              "tui.interaction.confirm_destructive",
	"ANYTTY_TUI_CLIPBOARD_HISTORY_MAX_ITEMS":      "tui.interaction.clipboard_history.max_items",
	"ANYTTY_TUI_CLIPBOARD_HISTORY_NAME_WIDTH":     "tui.interaction.clipboard_history.name_width",
	"ANYTTY_TUI_CLIPBOARD_HISTORY_PREVIEW_RATIO":  "tui.interaction.clipboard_history.preview_width_ratio",
	"ANYTTY_TUI_TERMINAL_AUTO_TAKE_OWNER":         "tui.terminal.auto_take_owner",
}

func Validate(cfg state.TUIConfigStore) error {
	if cfg.Version != 1 {
		return fmt.Errorf("version must be 1, got %d", cfg.Version)
	}
	if cfg.Daemon.OutputBuffer.CapacityBytes < 64<<10 || cfg.Daemon.OutputBuffer.CapacityBytes > 256<<20 {
		return fmt.Errorf("daemon.output_buffer.capacity_bytes must be between 65536 and 268435456")
	}
	if !oneOf(cfg.Daemon.OutputBuffer.Overflow, "drop", "block") {
		return fmt.Errorf("daemon.output_buffer.overflow must be drop or block, got %q", cfg.Daemon.OutputBuffer.Overflow)
	}
	if cfg.Daemon.OutputBuffer.ResidentBudgetBytes < 64<<10 || cfg.Daemon.OutputBuffer.ResidentBudgetBytes > 2<<30 {
		return fmt.Errorf("daemon.output_buffer.resident_budget_bytes must be between 65536 and 2147483648")
	}
	if cfg.Daemon.ResourceSampling.IntervalMS < minResourceSampleIntervalMS || cfg.Daemon.ResourceSampling.IntervalMS > maxResourceSampleIntervalMS {
		return fmt.Errorf("daemon.resource_sampling.interval_ms must be between %d and %d", minResourceSampleIntervalMS, maxResourceSampleIntervalMS)
	}
	if cfg.Daemon.ResourceSampling.MaxSamples < minResourceHistorySamples || cfg.Daemon.ResourceSampling.MaxSamples > maxResourceHistorySamples {
		return fmt.Errorf("daemon.resource_sampling.max_samples must be between %d and %d", minResourceHistorySamples, maxResourceHistorySamples)
	}
	if cfg.Daemon.History.MaxSizeMB < 0 || cfg.Daemon.History.MaxSizeMB > maxHistorySizeMB {
		return fmt.Errorf("daemon.history.max_size_mb must be between 0 and %d", maxHistorySizeMB)
	}
	if cfg.Daemon.History.MaxAgeDays < 0 || cfg.Daemon.History.MaxAgeDays > 36500 {
		return fmt.Errorf("daemon.history.max_age_days must be between 0 and 36500")
	}
	if !oneOf(cfg.Daemon.History.Compression, "zstd", "s2", "none") {
		return fmt.Errorf("daemon.history.compression must be zstd, s2 or none, got %q", cfg.Daemon.History.Compression)
	}
	if !oneOf(cfg.Daemon.History.CompressionLevel, "fast", "balanced", "best") {
		return fmt.Errorf("daemon.history.compression_level must be fast, balanced or best, got %q", cfg.Daemon.History.CompressionLevel)
	}
	if strings.TrimSpace(cfg.Profile) == "" {
		return fmt.Errorf("tui.profile must not be empty")
	}
	if !oneOf(cfg.Theme.Mode, "dark", "light", "system") {
		return fmt.Errorf("tui.theme.mode must be dark, light or system, got %q", cfg.Theme.Mode)
	}
	if !oneOf(cfg.Theme.Palette, "host", "builtin") {
		return fmt.Errorf("tui.theme.palette must be host or builtin, got %q", cfg.Theme.Palette)
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{"tui.theme.primary", cfg.Theme.Primary},
		{"tui.theme.secondary", cfg.Theme.Secondary},
		{"tui.theme.foreground", cfg.Theme.Foreground},
		{"tui.theme.background", cfg.Theme.Background},
		{"tui.theme.muted", cfg.Theme.Muted},
		{"tui.theme.success", cfg.Theme.Success},
		{"tui.theme.warning", cfg.Theme.Warning},
		{"tui.theme.danger", cfg.Theme.Danger},
		{"tui.theme.info", cfg.Theme.Info},
		{"tui.theme.border.panel", cfg.Theme.Border.Panel},
		{"tui.theme.border.active", cfg.Theme.Border.Active},
		{"tui.theme.border.inactive", cfg.Theme.Border.Inactive},
		{"tui.theme.border.muted", cfg.Theme.Border.Muted},
		{"tui.theme.surface.chrome_bg", cfg.Theme.Surface.ChromeBG},
		{"tui.theme.surface.status_bg", cfg.Theme.Surface.StatusBG},
		{"tui.theme.surface.overlay_bg", cfg.Theme.Surface.OverlayBG},
		{"tui.theme.surface.toast_bg", cfg.Theme.Surface.ToastBG},
	} {
		if !validOptionalHexColor(item.value) {
			return fmt.Errorf("%s must be empty or #RRGGBB, got %q", item.name, item.value)
		}
	}
	if !oneOf(cfg.Chrome.PanelPresentation, "split-line", "card") {
		return fmt.Errorf("tui.chrome.panel_presentation must be split-line or card, got %q", cfg.Chrome.PanelPresentation)
	}
	if !oneOf(cfg.Chrome.Picker.Presentation, "card", "flat") {
		return fmt.Errorf("tui.chrome.picker.presentation must be card or flat, got %q", cfg.Chrome.Picker.Presentation)
	}
	if !oneOf(cfg.Chrome.Picker.Width, "adaptive", "wide") {
		return fmt.Errorf("tui.chrome.picker.width must be adaptive or wide, got %q", cfg.Chrome.Picker.Width)
	}
	if !oneOf(cfg.Chrome.Picker.Density, "compact", "comfortable") {
		return fmt.Errorf("tui.chrome.picker.density must be compact or comfortable, got %q", cfg.Chrome.Picker.Density)
	}
	if !oneOf(cfg.Chrome.Picker.EndpointTabs, "underline", "plain") {
		return fmt.Errorf("tui.chrome.picker.endpoint_tabs must be underline or plain, got %q", cfg.Chrome.Picker.EndpointTabs)
	}
	if err := validatePickerEndpointStatus(cfg.Chrome.Picker.EndpointStatus); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Chrome.TabCreateIcon) == "" {
		return fmt.Errorf("tui.chrome.tab_create_icon must not be empty")
	}
	if strings.ContainsAny(cfg.Chrome.TabCreateTemplate, "\r\n") {
		return fmt.Errorf("tui.chrome.tab_create_template must be a single-line template")
	}
	if strings.ContainsAny(cfg.Chrome.WorkspaceTemplate, "\r\n") {
		return fmt.Errorf("tui.chrome.workspace_template must be a single-line template")
	}
	if strings.ContainsAny(cfg.Chrome.TabTemplate, "\r\n") {
		return fmt.Errorf("tui.chrome.tab_template must be a single-line template")
	}
	if strings.ContainsAny(cfg.Chrome.PaneTitleTemplate, "\r\n") {
		return fmt.Errorf("tui.chrome.pane_title_template must be a single-line template")
	}
	if err := validatePaneChromeGlyphs(cfg.Chrome.PaneGlyphs); err != nil {
		return err
	}
	if err := validateFooterConfig(cfg.Footer); err != nil {
		return err
	}
	if cfg.Interaction.StickyPrefixTimeoutMS < 0 {
		return fmt.Errorf("tui.interaction.sticky_prefix_timeout_ms must be >= 0")
	}
	if cfg.Interaction.ShortcutPassthroughIntervalMS <= 0 {
		return fmt.Errorf("tui.interaction.shortcut_passthrough_interval_ms must be > 0")
	}
	if cfg.Interaction.ClipboardHistory.MaxItems <= 0 {
		return fmt.Errorf("tui.interaction.clipboard_history.max_items must be > 0")
	}
	if cfg.Interaction.ClipboardHistory.NameWidth <= 0 {
		return fmt.Errorf("tui.interaction.clipboard_history.name_width must be > 0")
	}
	if cfg.Interaction.ClipboardHistory.PreviewWidthRatio <= 0 || cfg.Interaction.ClipboardHistory.PreviewWidthRatio >= 1 {
		return fmt.Errorf("tui.interaction.clipboard_history.preview_width_ratio must be > 0 and < 1")
	}
	if !oneOf(cfg.Interaction.Picker.FuzzyMatch, "subsequence") {
		return fmt.Errorf("tui.interaction.picker.fuzzy_match must be subsequence, got %q", cfg.Interaction.Picker.FuzzyMatch)
	}
	if err := validateShortcutsConfig(cfg.Shortcuts); err != nil {
		return err
	}
	return nil
}

func validatePaneChromeGlyphs(glyphs state.TUIPaneChromeGlyphsConfig) error {
	for _, item := range []struct {
		name  string
		value string
	}{
		{"tui.chrome.pane_glyphs.action_left", glyphs.ActionLeft},
		{"tui.chrome.pane_glyphs.action_right", glyphs.ActionRight},
		{"tui.chrome.pane_glyphs.action_separator", glyphs.ActionSeparator},
		{"tui.chrome.pane_glyphs.action_group_left", glyphs.ActionGroupLeft},
		{"tui.chrome.pane_glyphs.action_group_right", glyphs.ActionGroupRight},
		{"tui.chrome.pane_glyphs.owner_left", glyphs.OwnerLeft},
		{"tui.chrome.pane_glyphs.owner_right", glyphs.OwnerRight},
		{"tui.chrome.pane_glyphs.owner", glyphs.Owner},
		{"tui.chrome.pane_glyphs.owner_pending", glyphs.OwnerPending},
		{"tui.chrome.pane_glyphs.take_owner", glyphs.TakeOwner},
		{"tui.chrome.pane_glyphs.zoom", glyphs.Zoom},
		{"tui.chrome.pane_glyphs.unzoom", glyphs.Unzoom},
		{"tui.chrome.pane_glyphs.split_vertical", glyphs.SplitVertical},
		{"tui.chrome.pane_glyphs.split_horizontal", glyphs.SplitHorizontal},
		{"tui.chrome.pane_glyphs.close", glyphs.Close},
		{"tui.chrome.pane_glyphs.size_lock", glyphs.SizeLock},
		{"tui.chrome.pane_glyphs.size_unlock", glyphs.SizeUnlock},
		{"tui.chrome.pane_glyphs.center_floating", glyphs.CenterFloating},
		{"tui.chrome.pane_glyphs.collapse_floating", glyphs.CollapseFloating},
		{"tui.chrome.pane_glyphs.running", glyphs.Running},
		{"tui.chrome.pane_glyphs.waiting", glyphs.Waiting},
		{"tui.chrome.pane_glyphs.exited", glyphs.Exited},
		{"tui.chrome.pane_glyphs.killed", glyphs.Killed},
	} {
		if strings.ContainsAny(item.value, "\r\n") {
			return fmt.Errorf("%s must be a single-line glyph", item.name)
		}
	}
	return nil
}

func validatePickerEndpointStatus(status state.TUIPickerEndpointStatusConfig) error {
	for _, item := range []struct {
		name       string
		appearance state.TUIEndpointStatusAppearanceConfig
	}{
		{"unknown", status.Unknown},
		{"connected", status.Connected},
		{"connecting", status.Connecting},
		{"idle", status.Idle},
		{"disabled", status.Disabled},
		{"offline", status.Offline},
		{"reconnect_required", status.ReconnectRequired},
		{"unregistered", status.Unregistered},
	} {
		path := "tui.chrome.picker.endpoint_status." + item.name
		if err := validateSingleLine(path+".glyph", item.appearance.Glyph); err != nil {
			return err
		}
		if err := validateSingleLine(path+".style", item.appearance.Style); err != nil {
			return err
		}
		if !validEndpointStatusStyleToken(item.appearance.Style) {
			return fmt.Errorf("%s.style has unknown style token %q", path, item.appearance.Style)
		}
	}
	return nil
}

func validEndpointStatusStyleToken(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if strings.HasPrefix(value, "#") {
		return validOptionalHexColor(value)
	}
	return oneOf(value,
		"accent",
		"foreground",
		"strong-foreground",
		"muted",
		"info",
		"success",
		"warning",
		"danger",
		"danger-strong",
		"picker",
		"picker-muted",
		"picker-accent",
		"picker-info",
		"picker-success",
		"picker-match",
	)
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func validOptionalHexColor(value string) bool {
	if value == "" {
		return true
	}
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

func validateFooterConfig(footer state.TUIFooterConfig) error {
	for _, item := range []struct {
		path  string
		value string
	}{
		{"tui.footer.templates.mode_badge", footer.Templates.ModeBadge},
		{"tui.footer.templates.key", footer.Templates.Key},
		{"tui.footer.templates.action", footer.Templates.Action},
		{"tui.footer.templates.separator", footer.Templates.Separator},
		{"tui.footer.templates.workspace_summary", footer.Templates.WorkspaceSummary},
		{"tui.footer.templates.floating_summary", footer.Templates.FloatingSummary},
		{"tui.footer.templates.floating_collapsed_summary", footer.Templates.FloatingCollapsedSummary},
		{"tui.footer.templates.terminals_summary", footer.Templates.TerminalsSummary},
		{"tui.footer.templates.tabs_summary", footer.Templates.TabsSummary},
		{"tui.footer.templates.panes_summary", footer.Templates.PanesSummary},
		{"tui.footer.templates.keylock_on", footer.Templates.KeylockOn},
		{"tui.footer.templates.keylock_off", footer.Templates.KeylockOff},
	} {
		if err := validateSingleLine(item.path, item.value); err != nil {
			return err
		}
	}
	for name, mode := range footer.Modes {
		if !validFooterConfigName(name) {
			return fmt.Errorf("tui.footer.modes.%s has invalid name", name)
		}
		for _, item := range []struct {
			path  string
			value string
		}{
			{"icon", mode.Icon},
			{"label", mode.Label},
			{"style", mode.Style},
		} {
			if err := validateSingleLine("tui.footer.modes."+name+"."+item.path, item.value); err != nil {
				return err
			}
		}
		if !validFooterStyleToken(mode.Style) {
			return fmt.Errorf("tui.footer.modes.%s.style has unknown style token %q", name, mode.Style)
		}
	}
	return nil
}

func validateSingleLine(path string, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s must be a single-line value", path)
	}
	return nil
}

func validFooterStyleToken(value string) bool {
	return oneOf(strings.TrimSpace(value),
		"",
		"accent",
		"foreground",
		"strong-foreground",
		"muted",
		"status",
		"status-accent",
		"status-muted",
		"status-warning",
		"footer-chrome",
		"footer-muted",
		"footer-accent",
		"footer-key-pane",
		"footer-key-resize",
		"footer-key-tab",
		"footer-key-workspace",
		"footer-key-float",
		"footer-key-copy",
		"footer-key-picker",
		"footer-key-global",
		"info",
		"success",
		"warning",
		"danger",
		"danger-strong",
	)
}

func validateShortcutsConfig(shortcuts state.TUIShortcutConfig) error {
	canonicalActions := map[string]string{}
	for actionID, action := range shortcuts.Actions {
		if !validShortcutActionID(actionID) {
			return fmt.Errorf("tui.shortcuts.actions has invalid action id %q", actionID)
		}
		_, invocation, _, ok := shortcut.PolicyForSource(actionID)
		if !ok {
			return fmt.Errorf("tui.shortcuts.actions.%s references unknown shortcut action %q", actionID, actionID)
		}
		if previous, ok := canonicalActions[string(invocation.ID)]; ok {
			return fmt.Errorf("tui.shortcuts.actions.%s duplicates canonical shortcut action configured by %s", actionID, previous)
		}
		canonicalActions[string(invocation.ID)] = actionID
		if err := validateSingleLine("tui.shortcuts.actions."+actionID+".label", action.Label); err != nil {
			return err
		}
		if err := validateSingleLine("tui.shortcuts.actions."+actionID+".style", action.Style); err != nil {
			return err
		}
		if !validFooterStyleToken(action.Style) {
			return fmt.Errorf("tui.shortcuts.actions.%s.style has unknown style token %q", actionID, action.Style)
		}
	}
	effectiveKeys := map[string]string{}
	for sceneName, scene := range shortcuts.Scenes {
		if !builtinShortcutScene(sceneName) {
			return fmt.Errorf("tui.shortcuts has invalid scene %q", sceneName)
		}
		for key, binding := range scene.Bindings {
			path := "tui.shortcuts." + sceneName + "." + key
			if input.ShortcutKeyIsGlobalEscape(key) {
				return fmt.Errorf("%s uses reserved global back key; Esc is not configurable", path)
			}
			signature, ok := input.ShortcutBindingSignature(sceneName, key)
			if !ok {
				return fmt.Errorf("%s has invalid key", path)
			}
			if previous, ok := effectiveKeys[signature]; ok {
				return fmt.Errorf("%s conflicts with %s at runtime shortcut key", path, previous)
			}
			effectiveKeys[signature] = path
			if strings.TrimSpace(binding.Action) == "" {
				return fmt.Errorf("%s.action must not be empty", path)
			}
			if !validShortcutActionID(binding.Action) {
				return fmt.Errorf("%s.action has invalid action id %q", path, binding.Action)
			}
			if err := validateSingleLine(path+".label", binding.Label); err != nil {
				return err
			}
			if strings.HasPrefix(binding.Action, "menu.") {
				target := strings.TrimPrefix(binding.Action, "menu.")
				if !validShortcutSceneName(target) {
					return fmt.Errorf("%s.action has invalid menu target %q", path, target)
				}
				if _, ok := shortcuts.Scenes[target]; !ok && !builtinShortcutScene(target) {
					return fmt.Errorf("%s.action references unknown shortcut scene %q", path, target)
				}
			}
			invocation, _, err := actiondomain.ParseInvocation(binding.Action)
			if err != nil {
				return fmt.Errorf("%s.action references unknown shortcut action %q: %w", path, binding.Action, err)
			}
			policy, _, _, ok := shortcut.PolicyForSource(binding.Action)
			if !ok {
				return fmt.Errorf("%s.action %q is not bindable", path, binding.Action)
			}
			if !shortcut.AllowsScene(invocation.ID, sceneName) {
				return fmt.Errorf("%s.action %q is not allowed in scene %q", path, binding.Action, sceneName)
			}
			if routedShortcutScene(sceneName) && !policy.Routable {
				return fmt.Errorf("%s.action references non-routable shortcut action %q", path, binding.Action)
			}
		}
	}
	return nil
}

func validShortcutActionID(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func validShortcutSceneName(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func validShortcutKey(value string) bool {
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n\t ") {
		return false
	}
	// 手写 YAML parser 以点拼接配置 path；单字符 "." 仍是合法 shortcut key，
	// 其他含点 token 会与 long-form 的 .action/.label 字段产生歧义，且输入 parser 本身也不接受。
	return value == "." || !strings.Contains(value, ".")
}

func routedShortcutScene(value string) bool {
	scene, ok := shortcut.SceneByName(value)
	return ok && scene.Routable
}

func builtinShortcutScene(value string) bool {
	_, ok := shortcut.SceneByName(value)
	return ok
}
