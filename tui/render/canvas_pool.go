package render

import "sync"

const maxRetainedCanvas = 2

var canvasPool = struct {
	sync.Mutex
	items []*canvas
}{}

func acquireCanvas(width int, height int) *canvas {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	c := takeCanvas(width, height)
	c.width = width
	c.height = height
	if len(c.rows) < height {
		c.rows = make([][]canvasCell, height)
	}
	for row := 0; row < height; row++ {
		if cap(c.rows[row]) < width {
			c.rows[row] = make([]canvasCell, width)
			continue
		}
		c.rows[row] = c.rows[row][:width]
		clear(c.rows[row])
	}
	for row := height; row < len(c.rows); row++ {
		// 中文说明：归还池前断开不可见行，避免上一帧字符串和样式引用被长期保留。
		c.rows[row] = nil
	}
	c.rows = c.rows[:height]
	return c
}

func takeCanvas(width int, height int) *canvas {
	canvasPool.Lock()
	defer canvasPool.Unlock()
	for i := len(canvasPool.items) - 1; i >= 0; i-- {
		candidate := canvasPool.items[i]
		if candidate == nil {
			continue
		}
		if candidate.canReuse(width, height) {
			canvasPool.items[i] = canvasPool.items[len(canvasPool.items)-1]
			canvasPool.items[len(canvasPool.items)-1] = nil
			canvasPool.items = canvasPool.items[:len(canvasPool.items)-1]
			return candidate
		}
	}
	return &canvas{}
}

func (c *canvas) canReuse(width int, height int) bool {
	if c == nil || len(c.rows) < height {
		return false
	}
	for row := 0; row < height; row++ {
		if cap(c.rows[row]) < width {
			return false
		}
	}
	return true
}

func releaseCanvas(c *canvas) {
	if c == nil {
		return
	}
	for row := range c.rows {
		clear(c.rows[row])
	}
	c.width = 0
	c.height = 0
	canvasPool.Lock()
	defer canvasPool.Unlock()
	if len(canvasPool.items) >= maxRetainedCanvas {
		return
	}
	canvasPool.items = append(canvasPool.items, c)
}
