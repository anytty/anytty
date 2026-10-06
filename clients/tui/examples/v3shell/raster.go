package main

import (
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// rasterize is the offline rasterizer behind -selftest and the golden tests:
// it mirrors the reference screen() painter (declaration order, per-box
// clear, wide-rune continuation cells) so the Go program can be compared cell
// by cell with the Python replica goldens.
func (m *model) rasterize(root *pb.Box) ([]string, [][]string) {
	grid := make([][]string, m.rows)
	styles := make([][]string, m.rows)
	for y := 0; y < m.rows; y++ {
		grid[y] = make([]string, m.cols)
		styles[y] = make([]string, m.cols)
		for x := 0; x < m.cols; x++ {
			grid[y][x] = " "
		}
	}
	m.cursor = nil
	for _, box := range root.GetChildren() {
		if box.Visible != nil && !box.GetVisible() {
			continue
		}
		px, py := 0, 0
		if box.GetPos() != nil {
			px, py = int(box.GetPos().GetX()), int(box.GetPos().GetY())
		}
		width := m.cols
		if box.GetSize() != nil && box.GetSize().GetWidth() > 0 {
			width = int(box.GetSize().GetWidth())
		}
		height := 1
		if box.GetSize() != nil && box.GetSize().GetHeight() > 0 {
			height = int(box.GetSize().GetHeight())
		}
		clearRect(grid, styles, rect{px, py, width, height}, m.cols, m.rows)
		m.drawBox(box, rect{px, py, width, height}, grid, styles)
	}
	lines := make([]string, m.rows)
	for y := 0; y < m.rows; y++ {
		lines[y] = strings.Join(grid[y], "")
	}
	return lines, styles
}

func (m *model) drawBox(box *pb.Box, r rect, grid [][]string, styles [][]string) {
	if r.w <= 0 || r.h <= 0 {
		return
	}
	if content := box.GetContent(); content != nil {
		if content.GetSelf() != "" {
			lines := m.selfLines(box)
			for index := 0; index < r.h; index++ {
				text := ""
				if index < len(lines) {
					text = lines[index]
				}
				putText(grid, styles, r.x, r.y+index, r.w, sdk.Truncate(text, r.w), stContent, m)
			}
		} else {
			lines := content.GetLines()
			if len(lines) == 0 && content.GetText() != "" {
				lines = strings.Split(content.GetText(), "\n")
			}
			textStyle := box.GetStyle()
			for index, line := range lines {
				if index >= r.h {
					break
				}
				putText(grid, styles, r.x, r.y+index, r.w, sdk.Truncate(line, r.w), textStyle, m)
			}
		}
	}
	if cursor := box.GetCursor(); cursor != nil {
		m.cursor = &cursorPos{x: r.x + int(cursor.GetCol()), y: r.y + int(cursor.GetRow())}
	}
	for _, child := range box.GetChildren() {
		if child.Visible != nil && !child.GetVisible() {
			continue
		}
		if child.GetPos() == nil {
			continue
		}
		px, py := int(child.GetPos().GetX()), int(child.GetPos().GetY())
		cw, ch := r.w, r.h
		if child.GetSize() != nil && child.GetSize().GetWidth() > 0 {
			cw = int(child.GetSize().GetWidth())
		}
		if child.GetSize() != nil && child.GetSize().GetHeight() > 0 {
			ch = int(child.GetSize().GetHeight())
		}
		cr := rect{r.x + px, r.y + py, cw, ch}
		clearRect(grid, styles, cr, m.cols, m.rows)
		m.drawBox(child, cr, grid, styles)
	}
}

func (m *model) selfLines(box *pb.Box) []string {
	if m.demo {
		if p := m.paneByID(box.GetId()); p != nil && len(p.lines) > 0 {
			return p.lines
		}
	}
	return nil
}

func clearRect(grid, styles [][]string, r rect, cols, rows int) {
	for y := maxInt(0, r.y); y < minInt(rows, r.y+r.h); y++ {
		for x := maxInt(0, r.x); x < minInt(cols, r.x+r.w); x++ {
			grid[y][x] = " "
			styles[y][x] = ""
		}
	}
}

func putText(grid, styles [][]string, x, y, width int, text, textStyle string, m *model) {
	if y < 0 || y >= m.rows || width <= 0 {
		return
	}
	cursor := x
	remaining := width
	for _, ch := range text {
		cells := sdk.RuneWidth(ch)
		if cells == 0 {
			continue
		}
		if cells > remaining || cursor >= m.cols {
			break
		}
		if cursor >= 0 {
			grid[y][cursor] = string(ch)
			styles[y][cursor] = textStyle
			for offset := 1; offset < cells; offset++ {
				if cursor+offset < m.cols {
					grid[y][cursor+offset] = ""
					styles[y][cursor+offset] = textStyle
				}
			}
		}
		cursor += cells
		remaining -= cells
	}
}
