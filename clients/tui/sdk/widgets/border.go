package widgets

import (
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
)

// BorderSet is a box-drawing glyph set. Every field is one glyph except Top,
// Bottom, Left and Right, which are the repeated edge pieces.
type BorderSet struct {
	TopLeft     string
	Top         string
	TopRight    string
	Left        string
	Right       string
	BottomLeft  string
	Bottom      string
	BottomRight string
}

// BorderNormal is the default single-line box set (┌─┐│└┘).
func BorderNormal() BorderSet {
	return BorderSet{
		TopLeft: "┌", Top: "─", TopRight: "┐",
		Left: "│", Right: "│",
		BottomLeft: "└", Bottom: "─", BottomRight: "┘",
	}
}

// BorderRounded is the rounded-corner set (╭─╮│╰╯).
func BorderRounded() BorderSet {
	return BorderSet{
		TopLeft: "╭", Top: "─", TopRight: "╮",
		Left: "│", Right: "│",
		BottomLeft: "╰", Bottom: "─", BottomRight: "╯",
	}
}

// BorderThick is the heavy set (┏━┓┃┗┛).
func BorderThick() BorderSet {
	return BorderSet{
		TopLeft: "┏", Top: "━", TopRight: "┓",
		Left: "┃", Right: "┃",
		BottomLeft: "┗", Bottom: "━", BottomRight: "┛",
	}
}

// BorderDouble is the double-line set (╔═╗║╚╝).
func BorderDouble() BorderSet {
	return BorderSet{
		TopLeft: "╔", Top: "═", TopRight: "╗",
		Left: "║", Right: "║",
		BottomLeft: "╚", Bottom: "═", BottomRight: "╝",
	}
}

// BorderSetOrDefault returns the given set when non-zero, else BorderNormal.
func BorderSetOrDefault(set BorderSet) BorderSet {
	if set == (BorderSet{}) {
		return BorderNormal()
	}
	return set
}

// BorderBox draws a bordered box around an optional child builder. The border
// glyphs and the title are drawn program-side as styled text rows, independent
// of Frame. The box declares Width x Height; a zero geometry falls back to the
// child's measured content or a sensible minimum. Title is embedded in the top
// edge and truncated to fit the available inner width (CJK-safe). Boxes too
// small for chrome degrade to the bare child.
type BorderBox struct {
	ID         string
	Title      string
	Width      int
	Height     int
	Border     BorderSet
	Style      string
	TitleStyle string
	Child      *sdk.Builder
}

// Build returns the bordered box as a box tree.
func (b BorderBox) Build() *sdk.Builder {
	set := BorderSetOrDefault(b.Border)
	style := b.Style
	if style == "" {
		style = StyleBorder
	}
	titleStyle := b.TitleStyle
	if titleStyle == "" {
		titleStyle = StyleStrongForeground
	}

	width, height := b.Width, b.Height
	if width <= 0 {
		width = boxChildWidth(b.Child) + 2
		if width < 2 {
			width = 2
		}
	}
	if height <= 0 {
		height = 2
		if b.Child != nil {
			height++
		}
	}

	col := sdk.Box().Flow("col")
	if b.ID != "" {
		col.ID(b.ID)
	}
	col.Width(width).Height(height)

	// Too small for a full frame: degrade to the child alone (or an empty box).
	if width < 2 || height < 2 {
		if b.Child != nil {
			col.Child(b.Child)
		}
		return col
	}

	inner := width - 2
	col.Child(b.topEdge(set, titleStyle, inner))
	body := height - 2
	for i := 0; i < body; i++ {
		line := sdk.Row().Height(1)
		line.Child(sdk.Text(set.Left).Style(style).Width(1))
		if b.Child != nil && i == 0 {
			line.Child(b.Child)
		} else {
			line.Child(sdk.Text(strings.Repeat(" ", inner)).Width(inner).Flex(1))
		}
		line.Child(sdk.Text(set.Right).Style(style).Width(1))
		col.Child(line)
	}
	col.Child(sdk.Text(b.bottomEdge(set, inner)).Style(style).Width(width).Height(1))
	return col
}

// topEdge draws the top edge: a left corner, a title (when present) in its own
// styled segment, the remaining horizontal run and a right corner. The edge is
// exactly inner+2 cells wide.
func (b BorderBox) topEdge(set BorderSet, titleStyle string, inner int) *sdk.Builder {
	line := sdk.Row().Height(1)
	line.Child(sdk.Text(set.TopLeft).Style(b.borderStyle()).Width(1))
	if b.Title == "" {
		line.Child(sdk.Text(strings.Repeat(set.Top, inner)).Style(b.borderStyle()).Width(inner).Flex(1))
		line.Child(sdk.Text(set.TopRight).Style(b.borderStyle()).Width(1))
		return line
	}
	fill := inner - 1
	if fill < 0 {
		fill = 0
	}
	label := sdk.Truncate(" "+b.Title+" ", fill+1)
	labelWidth := sdk.DisplayWidth(label)
	rest := inner - labelWidth
	if rest < 0 {
		rest = 0
	}
	line.Child(sdk.Text(label).Style(titleStyle))
	if rest > 0 {
		line.Child(sdk.Text(strings.Repeat(set.Top, rest)).Style(b.borderStyle()).Width(rest).Flex(1))
	}
	line.Child(sdk.Text(set.TopRight).Style(b.borderStyle()).Width(1))
	return line
}

// bottomEdge draws the bottom edge as one styled run.
func (b BorderBox) bottomEdge(set BorderSet, inner int) string {
	return set.BottomLeft + strings.Repeat(set.Bottom, inner) + set.BottomRight
}

func (b BorderBox) borderStyle() string {
	if b.Style != "" {
		return b.Style
	}
	return StyleBorder
}

// boxChildWidth measures a child's declared width, falling back to its content
// width when the child leaves its size unset.
func boxChildWidth(child *sdk.Builder) int {
	if child == nil {
		return 0
	}
	node := child.Build()
	if width := int(node.GetSize().GetWidth()); width > 0 {
		return width
	}
	if content := node.GetContent(); content != nil {
		width := 0
		if content.GetText() != "" {
			width = sdk.DisplayWidth(content.GetText())
		}
		for _, line := range content.GetLines() {
			if w := sdk.DisplayWidth(line); w > width {
				width = w
			}
		}
		return width
	}
	return 0
}
