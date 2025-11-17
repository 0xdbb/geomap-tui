package tui

type brailleBuf struct {
	w, h int       // in cells
	m    [][]uint8 // per-cell 8-bit mask
}

func newBrailleBuf(w, h int) *brailleBuf {
	m := make([][]uint8, h)
	for i := range m {
		m[i] = make([]uint8, w)
	}
	return &brailleBuf{w: w, h: h, m: m}
}

// setPixel sets a micro-pixel at micro coords (2x4 per cell)
func (b *brailleBuf) setPixel(mx, my int) {
	if mx < 0 || my < 0 {
		return
	}
	cx, rx := mx/2, mx%2
	cy, ry := my/4, my%4
	if cy < 0 || cy >= b.h || cx < 0 || cx >= b.w {
		return
	}
	var bit uint8
	if rx == 0 {
		switch ry {
		case 0:
			bit = 0x01
		case 1:
			bit = 0x02
		case 2:
			bit = 0x04
		case 3:
			bit = 0x40
		}
	} else {
		switch ry {
		case 0:
			bit = 0x08
		case 1:
			bit = 0x10
		case 2:
			bit = 0x20
		case 3:
			bit = 0x80
		}
	}
	b.m[cy][cx] |= bit
}

// drawLineMicro draws a line on the microgrid using Bresenham
func (b *brailleBuf) drawLineMicro(x0, y0, x1, y1 int) {
	dx := abs(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -abs(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		b.setPixel(x0, y0)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (b *brailleBuf) toLines() []string {
	out := make([]string, b.h)
	for y := 0; y < b.h; y++ {
		row := make([]rune, b.w)
		for x := 0; x < b.w; x++ {
			mask := b.m[y][x]
			if mask == 0 {
				row[x] = ' '
			} else {
				row[x] = rune(0x2800 + int(mask))
			}
		}
		out[y] = string(row)
	}
	return out
}
