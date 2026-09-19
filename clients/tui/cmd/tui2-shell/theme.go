package main

import "strings"

// style is the program-side explicit style ("fg:#RRGGBB;bg:#RRGGBB;bold;dim").
// The wire protocol carries the string itself (PROTOCOL §2), so the program
// needs no host style type; this is a local, minimal writer.
type style struct {
	FG, BG                                string
	Bold, Dim, Italic, Underline, Reverse bool
}

// String renders the explicit style string the host translates verbatim.
func (s style) String() string {
	var parts []string
	if s.FG != "" {
		parts = append(parts, "fg:"+s.FG)
	}
	if s.BG != "" {
		parts = append(parts, "bg:"+s.BG)
	}
	for _, attr := range []struct {
		name string
		set  bool
	}{
		{"bold", s.Bold},
		{"dim", s.Dim},
		{"italic", s.Italic},
		{"underline", s.Underline},
		{"reverse", s.Reverse},
	} {
		if attr.set {
			parts = append(parts, attr.name)
		}
	}
	return strings.Join(parts, ";")
}

// theme is the program-side palette. Every slot resolves to an explicit,
// theme-free style string ("fg:#RRGGBB;bg:#RRGGBB;bold;dim") that the host
// translates verbatim to SGR: the host holds no palette and no theme config.
// `tui2.json`'s "theme" picks one of the built-in tables; unknown slots
// degrade to the host default style.
type theme struct {
	name  string
	slots map[string]style
}

// themeByName selects the built-in palette (the legacy "coralline-candy"
// recommended profile by default; dark/light stay available).
func themeByName(name string) theme {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "light":
		return theme{name: "light", slots: themeLight()}
	case "dark":
		return theme{name: "dark", slots: themeDark()}
	default:
		return theme{name: "recommended", slots: themeRecommended()}
	}
}

// style resolves one slot to its explicit wire style. An unknown slot
// returns "" (host default), never an error.
func (t theme) style(slot string) string {
	if s, ok := t.slots[slot]; ok {
		return s.String()
	}
	return ""
}

// DefaultThemeName is the palette used when the config does not say: the
// legacy coralline-candy recommended profile.
const DefaultThemeName = "recommended"

// ThemeNames lists the built-in palettes accepted by tui2.json.
var ThemeNames = []string{"recommended", "dark", "light"}

// themeRecommended is the explicit-style translation of the legacy
// "coralline-candy" recommended profile (tui/ui-v3.recommended.yaml):
// primary #f0abfc, secondary #3b2f63, fg #f8f4ff, bg #070611, surfaces
// #17132a/#261b44/#3b2f63 and the semantic success/warning/danger/info set.
func themeRecommended() map[string]style {
	return map[string]style{
		"default":        {FG: "#f8f4ff"},
		"bg":             {BG: "#070611"},
		"fg":             {FG: "#f8f4ff", BG: "#070611"},
		"muted":          {FG: "#9ca3c9"},
		"accent":         {FG: "#f0abfc", Bold: true},
		"accent_dim":     {FG: "#c4b5fd"},
		"warning":        {FG: "#fde68a"},
		"danger":         {FG: "#fb7185"},
		"ok":             {FG: "#86efac"},
		"info":           {FG: "#7dd3fc"},
		"chrome":         {FG: "#f8f4ff", BG: "#070611"},
		"chrome_focus":   {FG: "#1b1230", BG: "#f0abfc", Bold: true},
		"tab_active":     {FG: "#1b1230", BG: "#f0abfc", Bold: true},
		"tab_inactive":   {FG: "#c4b5fd", BG: "#261b44"},
		"button":         {FG: "#c4b5fd"},
		"button_hover":   {FG: "#1b1230", BG: "#f0abfc", Bold: true},
		"button_pressed": {FG: "#1b1230", BG: "#fde68a", Bold: true},
		"status":         {FG: "#f8f4ff", BG: "#17132a"},
		"selection":      {FG: "#f8f4ff", BG: "#3b2f63"},
		"overlay":        {FG: "#f8f4ff", BG: "#261b44"},
		"border":         {FG: "#3b2f63"},
		"border_focus":   {FG: "#f0abfc", Bold: true},
		"border_dead":    {FG: "#fb7185"},
		// Footer slots (old recommended footer-key-* tokens). The legacy
		// token->SGR table lived in the removed old program, so the v2 shell
		// maps each slot to the recommended palette by semantics
		// (RECOMMENDED_CONFIG §2 "footer 规格表" records the mapping).
		"footer":               {FG: "#9ca3c9"},
		"footer-accent":        {FG: "#f0abfc", Bold: true},
		"footer-key-pane":      {FG: "#f0abfc"},
		"footer-key-resize":    {FG: "#fde68a"},
		"footer-key-tab":       {FG: "#c4b5fd"},
		"footer-key-workspace": {FG: "#c4b5fd"},
		"footer-key-float":     {FG: "#7dd3fc"},
		"footer-key-copy":      {FG: "#86efac"},
		"footer-key-picker":    {FG: "#f0abfc"},
		"footer-key-global":    {FG: "#fde68a"},
		// Icon slots: the recommended profile tints glyphs with the primary
		// color; endpoint rows keep the info accent.
		"icon":     {FG: "#f0abfc"},
		"icon_dim": {FG: "#c4b5fd"},
		"endpoint": {FG: "#7dd3fc"},
	}
}

func themeDark() map[string]style {
	return map[string]style{
		"default":              {FG: "#e6e2ec"},
		"bg":                   {BG: "#0f1117"},
		"fg":                   {FG: "#e6e2ec"},
		"muted":                {FG: "#9a94a8", Dim: true},
		"accent":               {FG: "#a78bfa", Bold: true},
		"accent_dim":           {FG: "#6b5fa8"},
		"warning":              {FG: "#f0c05a"},
		"danger":               {FG: "#ef6f6f"},
		"ok":                   {FG: "#5fd08a"},
		"info":                 {FG: "#7fb2f0"},
		"chrome":               {FG: "#d9d4e4", BG: "#161823", Bold: true},
		"chrome_focus":         {FG: "#f2effa", BG: "#23263a", Bold: true},
		"tab_active":           {FG: "#14121c", BG: "#a78bfa", Bold: true},
		"tab_inactive":         {FG: "#9a94a8", Dim: true},
		"button":               {FG: "#b9b3c9"},
		"button_hover":         {FG: "#f2effa", BG: "#23263a", Bold: true},
		"button_pressed":       {FG: "#14121c", BG: "#a78bfa", Bold: true},
		"status":               {FG: "#d5d0e0", BG: "#12141c"},
		"selection":            {FG: "#f4f1fa", BG: "#3a3357"},
		"overlay":              {FG: "#d9d4e4", BG: "#191b26"},
		"border":               {FG: "#464a5a"},
		"border_focus":         {FG: "#a78bfa", Bold: true},
		"border_dead":          {FG: "#a35d5d"},
		"footer":               {FG: "#9a94a8"},
		"footer-accent":        {FG: "#a78bfa", Bold: true},
		"footer-key-pane":      {FG: "#a78bfa"},
		"footer-key-resize":    {FG: "#f0c05a"},
		"footer-key-tab":       {FG: "#b9b3c9"},
		"footer-key-workspace": {FG: "#b9b3c9"},
		"footer-key-float":     {FG: "#7fb2f0"},
		"footer-key-copy":      {FG: "#5fd08a"},
		"footer-key-picker":    {FG: "#a78bfa"},
		"footer-key-global":    {FG: "#f0c05a"},
	}
}

func themeLight() map[string]style {
	return map[string]style{
		"default":              {FG: "#2c2936"},
		"bg":                   {BG: "#f5f3f8"},
		"fg":                   {FG: "#2c2936"},
		"muted":                {FG: "#6f6a7c", Dim: true},
		"accent":               {FG: "#6d3fe0", Bold: true},
		"accent_dim":           {FG: "#9a7fe0"},
		"warning":              {FG: "#9a6a00"},
		"danger":               {FG: "#c0392b"},
		"ok":                   {FG: "#2f7d4f"},
		"info":                 {FG: "#2860b8"},
		"chrome":               {FG: "#39354a", BG: "#e7e3ef", Bold: true},
		"chrome_focus":         {FG: "#241f33", BG: "#d9d3e8", Bold: true},
		"tab_active":           {FG: "#fbfaff", BG: "#6d3fe0", Bold: true},
		"tab_inactive":         {FG: "#6f6a7c", Dim: true},
		"button":               {FG: "#6f6a7c"},
		"button_hover":         {FG: "#241f33", BG: "#d9d3e8", Bold: true},
		"button_pressed":       {FG: "#fbfaff", BG: "#6d3fe0", Bold: true},
		"status":               {FG: "#3a3648", BG: "#e2deeb"},
		"selection":            {FG: "#241f33", BG: "#cfc3f2"},
		"overlay":              {FG: "#39354a", BG: "#efeaf7"},
		"border":               {FG: "#b3adc0"},
		"border_focus":         {FG: "#6d3fe0", Bold: true},
		"border_dead":          {FG: "#a05a5a"},
		"footer":               {FG: "#6f6a7c"},
		"footer-accent":        {FG: "#6d3fe0", Bold: true},
		"footer-key-pane":      {FG: "#6d3fe0"},
		"footer-key-resize":    {FG: "#9a6a00"},
		"footer-key-tab":       {FG: "#7a6a9a"},
		"footer-key-workspace": {FG: "#7a6a9a"},
		"footer-key-float":     {FG: "#2860b8"},
		"footer-key-copy":      {FG: "#2f7d4f"},
		"footer-key-picker":    {FG: "#6d3fe0"},
		"footer-key-global":    {FG: "#9a6a00"},
	}
}

// terminalChrome is the explicit chrome style set the shell pushes to every
// bound terminal slot as content.props. Colors therefore follow the program
// theme (tui2.json "theme"), not a host palette; a slot without props would
// keep the component built-in defaults, and unknown keys are ignored. The
// prop keys are protocol literals (PROTOCOL §5), so the program does not
// import the host component package.
func (t theme) terminalChrome() map[string]string {
	return map[string]string{
		"chrome.border":       t.style("border"),
		"chrome.title":        t.style("muted"),
		"chrome.border_focus": t.style("border_focus"),
		"chrome.border_dead":  t.style("border_dead"),
		"chrome.badge":        t.style("warning"),
	}
}

// style resolves a slot through the active palette (model helper).
func (m *model) style(slot string) string { return m.theme.style(slot) }

// terminalProps is the chrome prop set for bound terminal slots (model
// helper); it is rebuilt per view so a theme switch repaints immediately.
func (m *model) terminalProps() map[string]string { return m.theme.terminalChrome() }
