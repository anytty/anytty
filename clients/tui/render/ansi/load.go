package ansi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/internal/screenmodel"
	"github.com/anytty/anytty/proto/access/apipb"
)

// LoadScreen 用共享 live-screen 模型替换解析器网格。
//
// 这是远端推屏（wire.TypeScreen）的落地方式：delta 只由 screenmodel 应用一次，
// 这里把结果投影成解析器网格，之后的渲染/滚动/复制/搜索全部复用同一个终端
// 模型，不再把 cells 编码成 ANSI 再解析回来。
//
// changed 为 nil 表示整屏有效；否则只写入这些行（其余行保持不变）。
// 返回 false 表示这一帧与解析器当前尺寸不一致（resize 在途），调用方应忽略它，
// 等下一帧整屏快照。
func (p *Parser) LoadScreen(screen *screenmodel.Screen, changed []int) bool {
	if p == nil || screen == nil || screen.Cols <= 0 || screen.Rows <= 0 {
		return false
	}
	if screen.Cols != p.cols || screen.Rows != p.rows {
		// 尺寸切换在途（宿主 resize 与 daemon 快照不同步）：以 daemon 的
		// 权威尺寸为准，整屏重载，避免丢掉一帧后 delta 链对不上。
		p.Resize(screen.Cols, screen.Rows)
		changed = nil
	}
	alt := screen.AlternateScreen()
	if alt != p.modes.AltScreen {
		p.modes.AltScreen = alt
		p.clear(p.grid())
		// 切换缓冲区后整屏内容都变了。
		changed = nil
	}
	grid := p.grid()
	if changed == nil {
		for y := 0; y < p.rows; y++ {
			p.loadRow(grid, y, screen.Line(y))
		}
	} else {
		for _, y := range changed {
			if y < 0 || y >= p.rows {
				continue
			}
			p.loadRow(grid, y, screen.Line(y))
		}
	}

	p.x = clampInt(screen.CursorX, 0, maxInt(p.cols-1, 0))
	p.y = clampInt(screen.CursorY, 0, maxInt(p.rows-1, 0))
	p.modes.CursorVisible = screen.CursorVisible
	p.cursorShape = screen.CursorShape
	// 远端屏帧是完整状态，不携带 DEC 2026 批次；清掉上一代可能残留的批次位，
	// 否则宿主会一直压着不出帧。
	p.modes.SynchronizedOutput = false
	p.pendingWrap = false
	p.pending = p.pending[:0]
	if modes := screen.Modes; modes != nil {
		p.modes.MouseCell = modes.GetMouseNormal()
		p.modes.MouseDrag = modes.GetMouseButtonEvent()
		p.modes.MouseAny = modes.GetMouseAnyEvent()
		p.modes.MouseSGR = modes.GetMouseSgr()
		p.modes.BracketPaste = modes.GetBracketedPaste()
	}
	return true
}

// loadRow 把模型的一行投影进网格。模型行长度已归一为 cols。
func (p *Parser) loadRow(grid []Cell, y int, line []screenmodel.Cell) {
	base := y * p.cols
	for x := 0; x < p.cols; x++ {
		if x >= len(line) {
			grid[base+x] = Cell{}
			continue
		}
		source := line[x]
		width := source.Width
		if width <= 0 && !source.Continuation {
			width = 1
		}
		if source.Text == "" && !source.Continuation && source.Style == nil {
			// 未写过的空白单元：解析器里就是零值（Blank() 语义）。
			grid[base+x] = Cell{}
			continue
		}
		cell := Cell{
			Text:         source.Text,
			Width:        width,
			Continuation: source.Continuation,
			Style:        StyleToken(source.Style),
		}
		if cell.Text == "" && !cell.Continuation {
			cell.Text = " "
			if cell.Width <= 0 {
				cell.Width = 1
			}
		}
		grid[base+x] = cell
	}
}

// StyleToken 把服务端结构化样式投影成解析器使用的 token。
//
// token 归一化规则与 SGR 解析器保持一致（只保留 bold/italic/underline/
// reverse/strikethrough + 前景/背景），保证远端推屏与原始字节流两条路径
// 得到完全相同的 token。
func StyleToken(style *apipb.CellStyle) render.Token {
	if style == nil {
		return render.TokenDefault
	}
	parts := make([]string, 0, 8)
	if style.GetBold() {
		parts = append(parts, "1")
	}
	if style.GetItalic() {
		parts = append(parts, "3")
	}
	if style.GetUnderline() {
		parts = append(parts, "4")
	}
	if style.GetReverse() {
		parts = append(parts, "7")
	}
	if style.GetStrikethrough() {
		parts = append(parts, "9")
	}
	if code := ColorParams(style.GetForeground(), true); code != "" {
		parts = append(parts, code)
	}
	if code := ColorParams(style.GetBackground(), false); code != "" {
		parts = append(parts, code)
	}
	return render.ANSIToken(strings.Join(parts, ";"))
}

// ColorParams 把服务端颜色编码（"ansi:N" / "idx:N" / "#rrggbb"）转成 SGR
// 参数片段；未知编码返回空串，与旧渲染路径一致。
func ColorParams(value string, foreground bool) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "ansi:") {
		if index, err := strconv.Atoi(strings.TrimPrefix(value, "ansi:")); err == nil && index >= 0 && index <= 15 {
			if index < 8 {
				if foreground {
					return strconv.Itoa(30 + index)
				}
				return strconv.Itoa(40 + index)
			}
			if foreground {
				return strconv.Itoa(90 + index - 8)
			}
			return strconv.Itoa(100 + index - 8)
		}
	}
	if strings.HasPrefix(value, "idx:") {
		if index, err := strconv.Atoi(strings.TrimPrefix(value, "idx:")); err == nil && index >= 0 && index <= 255 {
			if foreground {
				return fmt.Sprintf("38;5;%d", index)
			}
			return fmt.Sprintf("48;5;%d", index)
		}
	}
	if r, g, b, ok := parseHexColor(value); ok {
		if foreground {
			return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
		}
		return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
	}
	return ""
}

func parseHexColor(value string) (uint8, uint8, uint8, bool) {
	if !strings.HasPrefix(value, "#") {
		return 0, 0, 0, false
	}
	hex := strings.TrimPrefix(value, "#")
	switch len(hex) {
	case 3:
		r, err1 := strconv.ParseUint(hex[0:1]+hex[0:1], 16, 8)
		g, err2 := strconv.ParseUint(hex[1:2]+hex[1:2], 16, 8)
		b, err3 := strconv.ParseUint(hex[2:3]+hex[2:3], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, false
		}
		return uint8(r), uint8(g), uint8(b), true
	case 6:
		r, err := strconv.ParseUint(hex[0:2], 16, 8)
		g, err := strconv.ParseUint(hex[2:4], 16, 8)
		b, err := strconv.ParseUint(hex[4:6], 16, 8)
		if err != nil {
			return 0, 0, 0, false
		}
		return uint8(r), uint8(g), uint8(b), true
	default:
		return 0, 0, 0, false
	}
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
