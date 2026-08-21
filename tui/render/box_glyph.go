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
	// 中文说明：history 边框和 active split 边框是当前 owner；它们覆盖共享 junction，避免视觉上伸到相邻 pane。
	if boxBorderStylePriority(existingStyle) > boxBorderStylePriority(incomingStyle) {
		return existing
	}
	if boxBorderStylePriority(incomingStyle) > boxBorderStylePriority(existingStyle) {
		return incoming
	}
	return existing | incoming
}

func mergeBoxCellStyle(existing StyleToken, incoming StyleToken) StyleToken {
	// 中文说明：shared divider 优先保留 history，其次保留 active，不能被后绘制的 muted pane 降级。
	if boxBorderStylePriority(existing) > boxBorderStylePriority(incoming) {
		return existing
	}
	if boxBorderStylePriority(incoming) > boxBorderStylePriority(existing) {
		return incoming
	}
	if incoming != "" {
		return incoming
	}
	return existing
}

func boxBorderStylePriority(style StyleToken) int {
	switch style {
	case StyleHistoryBorder:
		return 2
	case StyleAccent:
		return 1
	default:
		return 0
	}
}
