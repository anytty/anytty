package render

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/anytty/anytty/tui/state"
)

type Theme struct {
	HostFG string
	HostBG string

	ChromeFG string
	ChromeBG string

	Accent string
	Muted  string

	Success string
	Warning string
	Danger  string
	Info    string

	PanelBorder      string
	MutedBorder      string
	ActivePaneBorder string
	InactivePane     string

	ToastBG   string
	OverlayBG string

	StatusFG string
	StatusBG string
}

func DefaultTheme() Theme {
	return Theme{
		HostFG: "#dedbe6",
		HostBG: "#050507",

		ChromeFG: "#dedbe6",
		ChromeBG: "#111016",

		Accent: "#a970ff",
		Muted:  "#b8b1c4",

		Success: "#8adf7a",
		Warning: "#f0c45c",
		Danger:  "#ff6b6b",
		Info:    "#7ab8ff",

		PanelBorder:      "#9b93a6",
		MutedBorder:      "#6f6878",
		ActivePaneBorder: "#a970ff",
		InactivePane:     "#b8b1c4",

		ToastBG:   "#15131d",
		OverlayBG: "#14121b",

		StatusFG: "#e7e2ef",
		StatusBG: "#08080d",
	}
}

func ThemeFromHostTheme(host state.HostThemeStore) Theme {
	return ThemeFromHostThemeConfig(host, state.TUIConfigStore{})
}

func ThemeFromHostThemeConfig(host state.HostThemeStore, cfg state.TUIConfigStore) Theme {
	theme := DefaultTheme()
	paletteDrivenBorder := false
	if cfg.Theme.Palette != "builtin" {
		if host.DefaultFG != "" {
			theme.HostFG = host.DefaultFG
			theme.ChromeFG = host.DefaultFG
			theme.StatusFG = host.DefaultFG
		}
		if host.DefaultBG != "" {
			theme.HostBG = host.DefaultBG
			theme.StatusBG = host.DefaultBG
			theme.ChromeBG = mixHostColor(host.DefaultBG, "#ffffff", 0.06)
			theme.ToastBG = mixHostColor(host.DefaultBG, "#ffffff", 0.08)
			theme.OverlayBG = mixHostColor(host.DefaultBG, "#ffffff", 0.10)
		}
		if color, ok := host.PaletteColor(5); ok {
			theme.Accent = color
			theme.ActivePaneBorder = color
			paletteDrivenBorder = true
		}
		if color, ok := host.PaletteColor(8); ok {
			theme.Muted = color
			theme.MutedBorder = color
			theme.InactivePane = color
			paletteDrivenBorder = true
		}
		if color, ok := host.PaletteColor(2); ok {
			theme.Success = color
		}
		if color, ok := host.PaletteColor(3); ok {
			theme.Warning = color
		}
		if color, ok := host.PaletteColor(1); ok {
			theme.Danger = color
		}
		if color, ok := host.PaletteColor(4); ok {
			theme.Info = color
		}
	}
	if paletteDrivenBorder && theme.PanelBorder == DefaultTheme().PanelBorder && theme.Muted != "" {
		theme.PanelBorder = mixHostColor(theme.Muted, theme.Accent, 0.45)
	}
	theme = applyThemeConfig(theme, cfg.Theme)
	return theme.WithFallback()
}

func applyThemeConfig(theme Theme, cfg state.TUIThemeConfig) Theme {
	if cfg.Foreground != "" {
		theme.HostFG = cfg.Foreground
		theme.ChromeFG = cfg.Foreground
		theme.StatusFG = cfg.Foreground
	}
	if cfg.Background != "" {
		theme.HostBG = cfg.Background
		theme.StatusBG = cfg.Background
		theme.ChromeBG = mixHostColor(cfg.Background, "#ffffff", 0.06)
		theme.ToastBG = mixHostColor(cfg.Background, "#ffffff", 0.08)
		theme.OverlayBG = mixHostColor(cfg.Background, "#ffffff", 0.10)
	}
	if cfg.Primary != "" {
		theme.Accent = cfg.Primary
		theme.ActivePaneBorder = cfg.Primary
	}
	if cfg.Secondary != "" {
		theme.Info = cfg.Secondary
	}
	if cfg.Muted != "" {
		theme.Muted = cfg.Muted
		theme.MutedBorder = cfg.Muted
		theme.InactivePane = cfg.Muted
	}
	if cfg.Success != "" {
		theme.Success = cfg.Success
	}
	if cfg.Warning != "" {
		theme.Warning = cfg.Warning
	}
	if cfg.Danger != "" {
		theme.Danger = cfg.Danger
	}
	if cfg.Info != "" {
		theme.Info = cfg.Info
	}
	if cfg.Border.Panel != "" {
		theme.PanelBorder = cfg.Border.Panel
	}
	if cfg.Border.Active != "" {
		theme.ActivePaneBorder = cfg.Border.Active
	}
	if cfg.Border.Inactive != "" {
		theme.InactivePane = cfg.Border.Inactive
	}
	if cfg.Border.Muted != "" {
		theme.MutedBorder = cfg.Border.Muted
	}
	if cfg.Surface.ChromeBG != "" {
		theme.ChromeBG = cfg.Surface.ChromeBG
	}
	if cfg.Surface.StatusBG != "" {
		theme.StatusBG = cfg.Surface.StatusBG
	}
	if cfg.Surface.OverlayBG != "" {
		theme.OverlayBG = cfg.Surface.OverlayBG
	}
	if cfg.Surface.ToastBG != "" {
		theme.ToastBG = cfg.Surface.ToastBG
	}
	return theme
}

func (theme Theme) WithFallback() Theme {
	defaults := DefaultTheme()
	if theme.HostFG == "" {
		theme.HostFG = defaults.HostFG
	}
	if theme.HostBG == "" {
		theme.HostBG = defaults.HostBG
	}
	if theme.ChromeFG == "" {
		theme.ChromeFG = defaults.ChromeFG
	}
	if theme.ChromeBG == "" {
		theme.ChromeBG = defaults.ChromeBG
	}
	if theme.Accent == "" {
		theme.Accent = defaults.Accent
	}
	if theme.Muted == "" {
		theme.Muted = defaults.Muted
	}
	if theme.Success == "" {
		theme.Success = defaults.Success
	}
	if theme.Warning == "" {
		theme.Warning = defaults.Warning
	}
	if theme.Danger == "" {
		theme.Danger = defaults.Danger
	}
	if theme.Info == "" {
		theme.Info = defaults.Info
	}
	if theme.PanelBorder == "" {
		theme.PanelBorder = defaults.PanelBorder
	}
	if theme.MutedBorder == "" {
		theme.MutedBorder = defaults.MutedBorder
	}
	if theme.ActivePaneBorder == "" {
		theme.ActivePaneBorder = defaults.ActivePaneBorder
	}
	if theme.InactivePane == "" {
		theme.InactivePane = defaults.InactivePane
	}
	if theme.ToastBG == "" {
		theme.ToastBG = defaults.ToastBG
	}
	if theme.OverlayBG == "" {
		theme.OverlayBG = defaults.OverlayBG
	}
	if theme.StatusFG == "" {
		theme.StatusFG = defaults.StatusFG
	}
	if theme.StatusBG == "" {
		theme.StatusBG = defaults.StatusBG
	}
	return theme
}

func StatusStyle(theme Theme) lipgloss.Style {
	theme = theme.WithFallback()
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.StatusFG)).
		Background(lipgloss.Color(theme.StatusBG))
}

func mixHostColor(base string, overlay string, ratio float64) string {
	baseR, baseG, baseB, okBase := parseHexColorBytes(base)
	overlayR, overlayG, overlayB, okOverlay := parseHexColorBytes(overlay)
	if !okBase || !okOverlay {
		return base
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	mix := func(a int, b int) int {
		return int(float64(a)*(1-ratio) + float64(b)*ratio + 0.5)
	}
	return formatHexColor(mix(baseR, overlayR), mix(baseG, overlayG), mix(baseB, overlayB))
}

func parseHexColorBytes(value string) (int, int, int, bool) {
	if len(value) != 7 || value[0] != '#' {
		return 0, 0, 0, false
	}
	parse := func(part string) (int, bool) {
		var out int
		for _, r := range part {
			out <<= 4
			switch {
			case r >= '0' && r <= '9':
				out += int(r - '0')
			case r >= 'a' && r <= 'f':
				out += int(r-'a') + 10
			case r >= 'A' && r <= 'F':
				out += int(r-'A') + 10
			default:
				return 0, false
			}
		}
		return out, true
	}
	r, okR := parse(value[1:3])
	g, okG := parse(value[3:5])
	b, okB := parse(value[5:7])
	return r, g, b, okR && okG && okB
}

func formatHexColor(r int, g int, b int) string {
	const hex = "0123456789abcdef"
	if r < 0 {
		r = 0
	}
	if r > 255 {
		r = 255
	}
	if g < 0 {
		g = 0
	}
	if g > 255 {
		g = 255
	}
	if b < 0 {
		b = 0
	}
	if b > 255 {
		b = 255
	}
	return string([]byte{'#', hex[r>>4], hex[r&15], hex[g>>4], hex[g&15], hex[b>>4], hex[b&15]})
}

func SafeLine(value string) string {
	return strings.ReplaceAll(value, "\n", " ")
}

func Width(value string) int {
	return DisplayWidth(value)
}

func Truncate(value string, width int) string {
	return TruncateCells(value, width)
}
