package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// cellToLonLat converts a map cell coordinate back to lon/lat using bbox, zoom, and pan.
func (m Model) cellToLonLat(cx, cy, w, h int) (float64, float64, bool) {
	if !(m.bbox.MaxX > m.bbox.MinX && m.bbox.MaxY > m.bbox.MinY) {
		return 0, 0, false
	}
	if w <= 1 || h <= 1 {
		return 0, 0, false
	}
	zx := float64(cx-m.offsetX) / float64(w-1)
	zy := 1.0 - float64(cy-m.offsetY)/float64(h-1)
	nx := 0.5 + (zx-0.5)/m.zoom
	ny := 0.5 + (zy-0.5)/m.zoom
	lon := m.bbox.MinX + nx*(m.bbox.MaxX-m.bbox.MinX)
	lat := m.bbox.MinY + ny*(m.bbox.MaxY-m.bbox.MinY)
	return lon, lat, true
}

func (m Model) renderAsciiMap(w, h int) string {
	// Plain background (no grid)
	lines := make([]string, h)
	for y := 0; y < h; y++ {
		row := make([]rune, w)
		for x := 0; x < w; x++ {
			row[x] = ' '
		}
		lines[y] = string(row)
	}
	// High-resolution braille buffer for crisp lines/edges
	br := newBrailleBuf(w, h)

	// No ASCII outline: use braille-only rendering for polygons

	// Draw polygons (fill then edges)
	if m.showPolys && len(m.polygons) > 0 {
		for _, poly := range m.polygons {
			// project rings to screen (cell coords for fill, micro for edges)
			var rings [][][2]int
			var ringsMic [][][2]int
			for _, ring := range poly {
				var sp [][2]int
				var sm [][2]int
				for _, p := range ring {
					sx, sy, ok := m.screenXY(p[0], p[1], w, h)
					if !ok {
						continue
					}
					mx, my, okm := m.screenXYMicro(p[0], p[1], w, h)
					if !okm {
						continue
					}
					sp = append(sp, [2]int{sx, sy})
					sm = append(sm, [2]int{mx, my})
				}
				if len(sp) >= 3 {
					rings = append(rings, sp)
				}
				if len(sm) >= 3 {
					ringsMic = append(ringsMic, sm)
				}
			}
			if len(rings) == 0 {
				continue
			}
			// No ASCII outline collection
			// fill using even-odd rule per scanline on outer ring (microgrid, holes ignored for now)
			if len(ringsMic) > 0 {
				outerMic := ringsMic[0]
				hMic := h * 4
				for yMic := 0; yMic < hMic; yMic++ {
					var xs []int
					for i := 0; i < len(outerMic); i++ {
						a := outerMic[i]
						b := outerMic[(i+1)%len(outerMic)]
						if a[1] == b[1] { // horizontal edge: skip
							continue
						}
						y0, y1 := a[1], b[1]
						x0, x1 := a[0], b[0]
						if (yMic >= y0 && yMic < y1) || (yMic >= y1 && yMic < y0) {
							t := float64(yMic-y0) / float64(y1-y0)
							x := int(float64(x0) + t*float64(x1-x0))
							xs = append(xs, x)
						}
					}
					if len(xs) >= 2 {
						sort.Ints(xs)
						for i := 0; i+1 < len(xs); i += 2 {
							xstart := xs[i]
							xend := xs[i+1]
							if xstart > xend {
								xstart, xend = xend, xstart
							}
							for xMic := max(0, xstart); xMic <= xend; xMic++ {
								br.setPixel(xMic, yMic)
							}
						}
					}
				}
			}
			// draw edges (high-res)
			for idx := range ringsMic {
				r := ringsMic[idx]
				for i := 0; i < len(r); i++ {
					a := r[i]
					b := r[(i+1)%len(r)]
					br.drawLineMicro(a[0], a[1], b[0], b[1])
				}
			}
		}
	}

	// Draw points only when dataset has no lines or polygons
	if m.showPoints && len(m.lines) == 0 && len(m.polygons) == 0 && len(m.points) > 0 && m.bbox.MaxX > m.bbox.MinX && m.bbox.MaxY > m.bbox.MinY {
		for _, p := range m.points {
			mx, my, ok := m.screenXYMicro(p[0], p[1], w, h)
			if !ok {
				continue
			}
			br.setPixel(mx, my)
		}
	}

	// Draw line strings (high-res)
	if m.showLines && len(m.lines) > 0 {
		for _, ls := range m.lines {
			var prev *[2]int
			for _, p := range ls {
				mx, my, ok := m.screenXYMicro(p[0], p[1], w, h)
				if !ok {
					continue
				}
				if prev != nil {
					br.drawLineMicro(prev[0], prev[1], mx, my)
				}
				prev = &[2]int{mx, my}
			}
		}
	}
	// Composite braille overlay onto base lines
	braLines := br.toLines()
	for y := 0; y < h && y < len(braLines); y++ {
		if len(braLines[y]) == 0 {
			continue
		}
		base := []rune(lines[y])
		over := []rune(braLines[y])
		for x := 0; x < len(base) && x < len(over); x++ {
			if over[x] != ' ' {
				base[x] = over[x]
			}
			if over[x] != ' ' {
				base[x] = over[x]
			}
		}
		lines[y] = string(base)
	}
	// No ASCII outline pass: boundaries are drawn via braille high-res edges

	// Hover highlight: draw an orange circle at the hovered vertex cell
	if m.hovering {
		cx := m.hoverMicX / 2
		cy := m.hoverMicY / 4
		if cy >= 0 && cy < len(lines) {
			r := []rune(lines[cy])
			if cx >= 0 && cx < len(r) {
				circle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("◯")
				// replace the cell with a colored circle
				// rebuild line with ANSI sequence at position cx
				pre := string(r[:cx])
				post := string(r[cx+1:])
				lines[cy] = pre + circle + post
			}
		}
	}
	return strings.Join(lines, "\n")
}

// screenXYMicro maps lon/lat into a 2x4 microgrid per cell for braille rendering.
func (m Model) screenXYMicro(lon, lat float64, w, h int) (int, int, bool) {
	if !(m.bbox.MaxX > m.bbox.MinX && m.bbox.MaxY > m.bbox.MinY) {
		return 0, 0, false
	}
	nx := (lon - m.bbox.MinX) / (m.bbox.MaxX - m.bbox.MinX)
	ny := (lat - m.bbox.MinY) / (m.bbox.MaxY - m.bbox.MinY)
	zx := 0.5 + (nx-0.5)*m.zoom
	zy := 0.5 + (ny-0.5)*m.zoom
	wMic := w * 2
	hMic := h * 4
	sx := int(zx*float64(wMic-1)) + m.offsetX*2
	sy := int((1.0-zy)*float64(hMic-1)) + m.offsetY*4
	return sx, sy, true
}

// screenXY maps lon/lat to current screen integer coordinates considering zoom and pan.
func (m Model) screenXY(lon, lat float64, w, h int) (int, int, bool) {
	if !(m.bbox.MaxX > m.bbox.MinX && m.bbox.MaxY > m.bbox.MinY) {
		return 0, 0, false
	}
	nx := (lon - m.bbox.MinX) / (m.bbox.MaxX - m.bbox.MinX)
	ny := (lat - m.bbox.MinY) / (m.bbox.MaxY - m.bbox.MinY)
	// Apply zoom around center (0.5, 0.5)
	zx := 0.5 + (nx-0.5)*m.zoom
	zy := 0.5 + (ny-0.5)*m.zoom
	sx := int(zx*float64(w-1)) + m.offsetX
	sy := int((1.0-zy)*float64(h-1)) + m.offsetY
	return sx, sy, true
}

// inspectNearest finds the point closest to the viewport center and returns lon/lat.
func (m Model) inspectNearest() (lon, lat float64, ok bool) {
	if len(m.points) == 0 {
		return 0, 0, false
	}
	w, h := m.mapW, m.mapH
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	cx, cy := w/2, h/2
	bestD := 1<<31 - 1
	var best [2]float64
	for _, p := range m.points {
		sx, sy, ok2 := m.screenXY(p[0], p[1], w, h)
		if !ok2 {
			continue
		}
		dx := sx - cx
		dy := sy - cy
		d := dx*dx + dy*dy
		if d < bestD {
			bestD = d
			best = p
		}
	}
	if bestD == 1<<31-1 {
		return 0, 0, false
	}
	return best[0], best[1], true
}
