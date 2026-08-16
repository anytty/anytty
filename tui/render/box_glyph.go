package render

const (
	boxConnUp = 1 << iota
	boxConnDown
	boxConnLeft
	boxConnRight
)

type boxStyle struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
}

var squareBoxStyle = boxStyle{
	TopLeft:     "┌",
	TopRight:    "┐",
	BottomLeft:  "└",
	BottomRight: "┘",
	Horizontal:  "─",
	Vertical:    "│",
}

var boxGlyphConnections = map[string]uint8{
	"│": boxConnUp | boxConnDown,
	"─": boxConnLeft | boxConnRight,
	"┌": boxConnDown | boxConnRight,
	"┐": boxConnDown | boxConnLeft,
	"└": boxConnUp | boxConnRight,
	"┘": boxConnUp | boxConnLeft,
	"├": boxConnUp | boxConnDown | boxConnRight,
	"┤": boxConnUp | boxConnDown | boxConnLeft,
	"┬": boxConnDown | boxConnLeft | boxConnRight,
	"┴": boxConnUp | boxConnLeft | boxConnRight,
	"┼": boxConnUp | boxConnDown | boxConnLeft | boxConnRight,
}

var boxConnectionGlyph = map[uint8]string{
	boxConnUp:                                            "│",
	boxConnDown:                                          "│",
	boxConnLeft:                                          "─",
	boxConnRight:                                         "─",
	boxConnUp | boxConnDown:                              "│",
	boxConnLeft | boxConnRight:                           "─",
	boxConnDown | boxConnRight:                           "┌",
	boxConnDown | boxConnLeft:                            "┐",
	boxConnUp | boxConnRight:                             "└",
	boxConnUp | boxConnLeft:                              "┘",
	boxConnUp | boxConnDown | boxConnRight:               "├",
	boxConnUp | boxConnDown | boxConnLeft:                "┤",
	boxConnDown | boxConnLeft | boxConnRight:             "┬",
	boxConnUp | boxConnLeft | boxConnRight:               "┴",
	boxConnUp | boxConnDown | boxConnLeft | boxConnRight: "┼",
}

func boxConnectionsForGlyph(glyph string) (uint8, bool) {
	connections, ok := boxGlyphConnections[glyph]
	return connections, ok
}

func boxGlyphForConnections(connections uint8) (string, bool) {
	glyph, ok := boxConnectionGlyph[connections]
	return glyph, ok
}

func mergeBoxCellConnections(existing uint8, incoming uint8, existingStyle StyleToken, incomingStyle StyleToken) uint8 {
	// 中文说明：active split 边框是当前焦点 owner；它覆盖共享 junction，避免视觉上伸到相邻 pane。
	if existingStyle == StyleAccent && incomingStyle != StyleAccent {
		return existing
	}
	if incomingStyle == StyleAccent && existingStyle != StyleAccent {
		return incoming
	}
	return existing | incoming
}

func mergeBoxCellStyle(existing StyleToken, incoming StyleToken) StyleToken {
	// 中文说明：split-line 的 shared divider 会被多个 pane 先后 merge；active 边框是焦点真值，不能被后绘制的 muted pane 降级。
	if existing == StyleAccent || incoming == StyleAccent {
		return StyleAccent
	}
	if incoming != "" {
		return incoming
	}
	return existing
}
