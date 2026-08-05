package history

// HistoryTheme 是查看 history projection 时解析 terminal default fg/bg 的输入。
// domain owner 是 viewer/projection 层；它不能写回 LogicalLine payload，也不能把
// 当前主题 RGB 烘焙进 authoritative history truth。
type HistoryTheme struct {
	DefaultFG string
	DefaultBG string
}

// ResolvedCellStyle 是 view-time style 结果。它服务 renderer/copy projection，
// 不是 logical-line payload；调用方不得把它作为 history truth 持久化。
type ResolvedCellStyle struct {
	FG            string
	BG            string
	Bold          bool
	Italic        bool
	Underline     bool
	Blink         bool
	Reverse       bool
	Strikethrough bool
}

// ResolveCellStyleForTheme 在查看时解析 default fg/bg。空 FG/BG 表示 terminal
// default 语义，只有这里才替换成 theme 默认色；ansi:N、idx:N 和 #rrggbb 都是
// 内容 token，必须原样保留。
func ResolveCellStyleForTheme(style CellStyle, theme HistoryTheme) ResolvedCellStyle {
	fg := style.FG
	if fg == "" {
		fg = theme.DefaultFG
	}
	bg := style.BG
	if bg == "" {
		bg = theme.DefaultBG
	}
	return ResolvedCellStyle{
		FG:            fg,
		BG:            bg,
		Bold:          style.Bold,
		Italic:        style.Italic,
		Underline:     style.Underline,
		Blink:         style.Blink,
		Reverse:       style.Reverse,
		Strikethrough: style.Strikethrough,
	}
}
