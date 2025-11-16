package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	list "github.com/charmbracelet/bubbles/list"
	textarea "github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"goemap/internal/geom"
)

type Model struct {
	width  int
	height int

	showSidebar bool
	helpVisible bool

	zoom    float64
	offsetX int
	offsetY int

	status string

	// File explorer
	cwd     string
	l       list.Model
	items   []list.Item
	selPath string

	// Data
	points   [][2]float64
	bbox     geom.BBox
	lines    [][][2]float64
	polygons [][][][2]float64

	// last rendered map size (for inspect)
	mapW int
	mapH int

	// paste mode
	pasteMode bool
	ta        textarea.Model

	// layer visibility
	showPoints bool
	showLines  bool
	showPolys  bool

	// inspect popup
	inspectPopup string

	// hover state
	hovering bool
	hoverCellX int
	hoverCellY int
	hoverMicX int
	hoverMicY int
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func New() Model {
	m := Model{
		showSidebar: false,
		helpVisible: true,
		zoom:        1.0,
		status:      "geomap ready",
		showPoints:  true,
		showLines:   true,
		showPolys:   true,
	}
	m.cwd, _ = os.Getwd()
	// list setup
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	m.l = list.New(nil, d, 0, 0)
	m.l.Title = "Files"
	m.l.SetShowHelp(false)
	m.l.SetShowStatusBar(false)
	m.l.SetFilteringEnabled(true)
	// textarea setup
	m.ta = textarea.New()
	m.ta.Placeholder = "Paste WKT here (POINT, MULTIPOINT, LINESTRING, POLYGON). Press Enter to render; Esc to cancel."
	m.ta.CharLimit = 0
	m.ta.SetWidth(50)
	m.ta.SetHeight(6)
	m.refreshDir()
	return m
}

// NewWithPath preloads a file's data at launch.
func NewWithPath(path string) Model {
	m := New()
	m.loadPath(path)
	return m
}

// Styles
var (
	baseFg    = lipgloss.Color("#E6E6E6")
	baseDimFg = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#6B7280"}
	accentFg  = lipgloss.Color("#7C3AED")
	subtleBg  = lipgloss.Color("#0B0F14")
	panelBg   = lipgloss.Color("#0F141A")
	borderCol = lipgloss.Color("#243141")

	appStyle   = lipgloss.NewStyle().Foreground(baseFg)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderCol).Padding(0, 1)
	titleStyle = lipgloss.NewStyle().Foreground(accentFg).Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(baseDimFg)
)

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.showSidebar {
			m.l.SetSize(28-2, m.height-1-2) // provisional; will be refined in View
		}
	case tea.KeyMsg:
		// If list is visible and filtering, send keys to list and ignore global commands
		if m.showSidebar && m.l.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.l, cmd = m.l.Update(msg)
			return m, cmd
		}
		if m.pasteMode {
			switch msg.String() {
			case "esc":
				m.pasteMode = false
				m.ta.Blur()
				return m, nil
			case "enter":
				w := strings.TrimSpace(m.ta.Value())
				if w == "" {
					m.status = "paste: empty"
					return m, nil
				}
				d, err := geom.ParseWKTData(w)
				if err != nil {
					m.status = "wkt error: " + err.Error()
					return m, nil
				}
				m.points, m.lines, m.polygons, m.bbox = d.Points, d.Lines, d.Polygons, d.BBox
				m.status = "rendered WKT"
				m.pasteMode = false
				m.ta.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.ta, cmd = m.ta.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1":
			m.showPoints = !m.showPoints
			m.status = fmt.Sprintf("points: %v", m.showPoints)
		case "2":
			m.showLines = !m.showLines
			m.status = fmt.Sprintf("lines: %v", m.showLines)
		case "3":
			m.showPolys = !m.showPolys
			m.status = fmt.Sprintf("polys: %v", m.showPolys)
		case "+", "=":
			if m.zoom < 64 {
				m.zoom *= 1.2
				m.status = fmt.Sprintf("zoom: %.2fx", m.zoom)
			}
		case "-", "_":
			if m.zoom > 0.05 {
				m.zoom /= 1.2
				m.status = fmt.Sprintf("zoom: %.2fx", m.zoom)
			}
		case "tab":
			m.showSidebar = !m.showSidebar
			if m.showSidebar {
				m.refreshDir()
				m.l.SetSize(28-2, m.height-1-2)
			}
		case "p":
			m.pasteMode = !m.pasteMode
			if m.pasteMode {
				m.ta.SetValue("")
				m.status = "paste mode"
				m.ta.Focus()
			} else {
				m.status = "view mode"
				m.ta.Blur()
			}
		case "h":
			m.helpVisible = !m.helpVisible
		case "i":
			lon, lat, ok := m.inspectNearest()
			if ok {
				// build popup content
				name := filepath.Base(m.selPath)
				if name == "" {
					name = "<unsaved>"
				}
				meta := []string{
					fmt.Sprintf("name: %s", name),
					fmt.Sprintf("path: %s", m.selPath),
					fmt.Sprintf("bbox: [%.5f, %.5f, %.5f, %.5f]", m.bbox.MinX, m.bbox.MinY, m.bbox.MaxX, m.bbox.MaxY),
					fmt.Sprintf("counts: pts=%d ls=%d poly=%d", len(m.points), len(m.lines), len(m.polygons)),
					fmt.Sprintf("nearest: lon=%.6f lat=%.6f", lon, lat),
					"crs: unknown", "datum: unknown",
				}
				m.inspectPopup = strings.Join(meta, "\n")
				m.status = "inspect popup"
			} else {
				m.inspectPopup = "no feature nearby"
				m.status = m.inspectPopup
			}
		case "l":
			// toggle all layers
			all := m.showPoints && m.showLines && m.showPolys
			m.showPoints = !all
			m.showLines = !all
			m.showPolys = !all
			m.status = fmt.Sprintf("layers: pts=%v ls=%v poly=%v", m.showPoints, m.showLines, m.showPolys)
		case "enter":
			if m.showSidebar {
				if it, ok := m.l.SelectedItem().(fileItem); ok {
					m.loadPath(it.path)
				}
			}
		case "up":
			m.offsetY -= 1
		case "down":
			m.offsetY += 1
		case "left":
			m.offsetX -= 2
		case "right":
			m.offsetX += 2
		}
	case tea.MouseMsg:
		// track hover over map area
		// compute map origin and size (must match View layout)
		sidebarWidth := 0
		if m.showSidebar { sidebarWidth = 28 }
		headerHeight := 1
		footerHeight := 2
		contentHeight := m.height - headerHeight - footerHeight
		if contentHeight < 4 { contentHeight = 4 }
		contentWidth := max(10, m.width)

		// Update list size with accurate content height when sidebar visible
		if m.showSidebar {
			m.l.SetSize(28-2, contentHeight-2)
		}

		mapWidth := contentWidth - sidebarWidth - 1
		if mapWidth < 10 { mapWidth = 10 }
		mapHeight := contentHeight
		mapOriginX := sidebarWidth + func() int { if m.showSidebar { return 1 } ; return 0 }()
		mapOriginY := headerHeight
		// mouse cell within map?
		cx, cy := msg.X, msg.Y
		if cx >= mapOriginX && cx < mapOriginX+mapWidth && cy >= mapOriginY && cy < mapOriginY+mapHeight {
			m.hovering = true
			m.hoverCellX = cx - mapOriginX
			m.hoverCellY = cy - mapOriginY
			// find nearest vertex (points + line vertices + polygon vertices) using micro coords
			hxMic := m.hoverCellX * 2
			hyMic := m.hoverCellY * 4
			best := 1<<31 - 1
			bx, by := hxMic, hyMic
			// points
			for _, p := range m.points {
				mx, my, ok := m.screenXYMicro(p[0], p[1], mapWidth, mapHeight)
				if !ok { continue }
				dx := mx - hxMic; dy := my - hyMic; d := dx*dx + dy*dy
				if d < best { best = d; bx, by = mx, my }
			}
			// lines
			for _, ls := range m.lines {
				for _, p := range ls {
					mx, my, ok := m.screenXYMicro(p[0], p[1], mapWidth, mapHeight)
					if !ok { continue }
					dx := mx - hxMic; dy := my - hyMic; d := dx*dx + dy*dy
					if d < best { best = d; bx, by = mx, my }
				}
			}
			// polygons
			for _, poly := range m.polygons {
				for _, ring := range poly {
					for _, p := range ring {
						mx, my, ok := m.screenXYMicro(p[0], p[1], mapWidth, mapHeight)
						if !ok { continue }
						dx := mx - hxMic; dy := my - hyMic; d := dx*dx + dy*dy
						if d < best { best = d; bx, by = mx, my }
					}
				}
			}
			m.hoverMicX, m.hoverMicY = bx, by
		} else {
			m.hovering = false
		}
	}
	// Pass messages to list when visible
	if m.showSidebar {
		var cmd tea.Cmd
		m.l, cmd = m.l.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Layout sizes
	sidebarWidth := 0
	if m.showSidebar {
		sidebarWidth = 28
	}
	headerHeight := 1
	footerHeight := 2
	contentHeight := m.height - headerHeight - footerHeight
	if contentHeight < 4 {
		contentHeight = 4
	}
	contentWidth := max(10, m.width)

	// Update list size with accurate content height when sidebar visible
	if m.showSidebar {
		m.l.SetSize(28-2, contentHeight-2)
	}

	// Header
	header := titleStyle.Render(" geomap ─ terminal geospatial viewer ")
	header = lipgloss.NewStyle().Width(contentWidth).Padding(0).Render(header)

	// Sidebar
	var sidebar string
	if m.showSidebar {
		// fixed sidebar without background highlight
		sidebar = lipgloss.NewStyle().Width(sidebarWidth).Render(m.l.View())
	}

	// Map viewport
	mapWidth := contentWidth - sidebarWidth - 1
	if mapWidth < 10 {
		mapWidth = 10
	}

	mapHeight := contentHeight
	// track map size for inspect (use full area; map canvas has no border)
	m.mapW = max(8, mapWidth)
	m.mapH = max(4, mapHeight)
	var ascii string
	if m.pasteMode {
		// size textarea to map area
		m.ta.SetWidth(m.mapW)
		m.ta.SetHeight(min(m.mapH, 12))
		ascii = m.ta.View()
	} else {
		ascii = m.renderAsciiMap(m.mapW, m.mapH)
	}
	// plain map canvas: no border, no background highlight
	mapView := lipgloss.NewStyle().Width(mapWidth).Height(mapHeight).Render(ascii)

	// Build inspect popup box (center-left overlay, not in map column)
	popup := ""
	if m.inspectPopup != "" {
		maxPopupW := min(48, contentWidth/2)
		if maxPopupW < 20 { maxPopupW = 20 }
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).MaxWidth(maxPopupW).Render(m.inspectPopup)
		// approximate vertical center within content area
		popup = lipgloss.Place(contentWidth, contentHeight, lipgloss.Left, lipgloss.Center, box)
	}

	// Body row
	var mapCol string = mapView
	var body string
	if m.showSidebar {
		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", mapCol)
	} else {
		body = mapCol
	}

	// Footer / help
	help := m.renderHelp()
	status := dimStyle.Render(" " + m.status + " ")
	footer := lipgloss.NewStyle().Width(contentWidth).Render(lipgloss.JoinHorizontal(lipgloss.Bottom, status, help))

	// Compose UI with popup overlay between header and body
	ui := lipgloss.JoinVertical(lipgloss.Left, header, popup, body, footer)
	return appStyle.Width(contentWidth).Height(m.height).Render(ui)
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
					if !okm { continue }
					sp = append(sp, [2]int{sx, sy})
					sm = append(sm, [2]int{mx, my})
				}
				if len(sp) >= 3 {
					rings = append(rings, sp)
				}
				if len(sm) >= 3 { ringsMic = append(ringsMic, sm) }
			}
			if len(rings) == 0 {
				continue
			}
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
							if xstart > xend { xstart, xend = xend, xstart }
							for xMic := max(0, xstart); xMic <= xend; xMic++ {
								br.setPixel(xMic, yMic)
							}
						}
					}
				}
			}
			// draw edges (high-res)
			for idx, ring := range ringsMic {
				_ = ring
				if idx >= len(ringsMic) { continue }
				r := ringsMic[idx]
				for i := 0; i < len(r); i++ {
					a := r[i]
					b := r[(i+1)%len(r)]
					br.drawLineMicro(a[0], a[1], b[0], b[1])
				}
			}
		}
	}

	// Draw data points if loaded
	if m.showPoints && len(m.points) > 0 && m.bbox.MaxX > m.bbox.MinX && m.bbox.MaxY > m.bbox.MinY {
		for _, p := range m.points {
			mx, my, ok := m.screenXYMicro(p[0], p[1], w, h)
			if !ok { continue }
			br.setPixel(mx, my)
		}
	}

	// Draw line strings (high-res)
	if m.showLines && len(m.lines) > 0 {
		for _, ls := range m.lines {
			var prev *[2]int
			for _, p := range ls {
				mx, my, ok := m.screenXYMicro(p[0], p[1], w, h)
				if !ok { continue }
				if prev != nil { br.drawLineMicro(prev[0], prev[1], mx, my) }
				prev = &[2]int{mx, my}
			}
		}
	}
	// Composite braille overlay onto base lines
	braLines := br.toLines()
	for y := 0; y < h && y < len(braLines); y++ {
		if len(braLines[y]) == 0 { continue }
		base := []rune(lines[y])
		over := []rune(braLines[y])
		for x := 0; x < len(base) && x < len(over); x++ {
			if over[x] != ' ' { base[x] = over[x] }
		}
		lines[y] = string(base)
	}
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
	if !(m.bbox.MaxX > m.bbox.MinX && m.bbox.MaxY > m.bbox.MinY) { return 0, 0, false }
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

// braille buffer implementation
type brailleBuf struct {
	w, h int // in cells
	m [][]uint8 // per-cell 8-bit mask
}

func newBrailleBuf(w, h int) *brailleBuf {
	m := make([][]uint8, h)
	for i := range m { m[i] = make([]uint8, w) }
	return &brailleBuf{w:w, h:h, m:m}
}

// setPixel sets a micro-pixel at micro coords (2x4 per cell)
func (b *brailleBuf) setPixel(mx, my int) {
	if mx < 0 || my < 0 { return }
	cx, rx := mx/2, mx%2
	cy, ry := my/4, my%4
	if cy < 0 || cy >= b.h || cx < 0 || cx >= b.w { return }
	var bit uint8
	if rx == 0 {
		switch ry { case 0: bit=0x01; case 1: bit=0x02; case 2: bit=0x04; case 3: bit=0x40 }
	} else {
		switch ry { case 0: bit=0x08; case 1: bit=0x10; case 2: bit=0x20; case 3: bit=0x80 }
	}
	b.m[cy][cx] |= bit
}

// drawLineMicro draws a line on the microgrid using Bresenham
func (b *brailleBuf) drawLineMicro(x0, y0, x1, y1 int) {
	dx := abs(x1 - x0)
	sx := -1
	if x0 < x1 { sx = 1 }
	dy := -abs(y1 - y0)
	sy := -1
	if y0 < y1 { sy = 1 }
	err := dx + dy
	for {
		b.setPixel(x0, y0)
		if x0 == x1 && y0 == y1 { break }
		e2 := 2*err
		if e2 >= dy { err += dy; x0 += sx }
		if e2 <= dx { err += dx; y0 += sy }
	}
}

func (b *brailleBuf) toLines() []string {
	out := make([]string, b.h)
	for y := 0; y < b.h; y++ {
		row := make([]rune, b.w)
		for x := 0; x < b.w; x++ {
			mask := b.m[y][x]
			if mask == 0 { row[x] = ' ' } else { row[x] = rune(0x2800 + int(mask)) }
		}
		out[y] = string(row)
	}
	return out
}

func (m Model) renderHelp() string {
	if !m.helpVisible {
		return ""
	}
	keys := []string{
		"↑↓←→ pan",
		"+/- zoom",
		"Tab sidebar",
		"Enter open",
		"p paste",
		"i inspect",
		"l layers",
		"h help",
		"q quit",
	}
	return dimStyle.Render("  " + strings.Join(keys, "  "))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func padRight(s string, n int) string {
	if n <= 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// file explorer helpers
type fileItem struct {
	title, desc string
	path        string
	isDir       bool
}

func (f fileItem) Title() string       { return f.title }
func (f fileItem) Description() string { return f.desc }
func (f fileItem) FilterValue() string { return f.title }

func (m *Model) refreshDir() {
	entries, err := os.ReadDir(m.cwd)
	if err != nil {
		m.status = "read dir error: " + err.Error()
		return
	}
	var items []list.Item
	for _, e := range entries {
		name := e.Name()
		p := filepath.Join(m.cwd, name)
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".geojson" || ext == ".json" || ext == ".csv" || ext == ".kml" || ext == ".wkt" || ext == ".shp" {
			items = append(items, fileItem{title: name, desc: ext, path: p})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].(fileItem).Title() < items[j].(fileItem).Title() })
	m.items = items
	m.l.SetItems(items)
	if len(items) == 0 {
		m.status = "no supported files in current directory"
	}
}

// loadPath loads supported formats into the model.
func (m *Model) loadPath(p string) {
	m.selPath = p
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".geojson", ".json":
		d, err := geom.LoadGeo(p)
		if err != nil {
			m.status = "load error: " + err.Error()
			return
		}
		m.points, m.lines, m.polygons, m.bbox = d.Points, d.Lines, d.Polygons, d.BBox
		m.status = "loaded: " + filepath.Base(p)
	case ".csv":
		pts, bb, err := geom.LoadCSV(p)
		if err != nil {
			m.status = "load error: " + err.Error()
			return
		}
		m.points, m.lines, m.polygons, m.bbox = pts, nil, nil, bb
		m.status = "loaded: " + filepath.Base(p)
	case ".kml":
		pts, bb, err := geom.LoadKML(p)
		if err != nil {
			m.status = "load error: " + err.Error()
			return
		}
		m.points, m.lines, m.polygons, m.bbox = pts, nil, nil, bb
		m.status = "loaded: " + filepath.Base(p)
	case ".wkt":
		data, err := os.ReadFile(p)
		if err != nil {
			m.status = "load error: " + err.Error()
			return
		}
		d, err := geom.ParseWKTData(string(data))
		if err != nil {
			m.status = "wkt error: " + err.Error()
			return
		}
		m.points, m.lines, m.polygons, m.bbox = d.Points, d.Lines, d.Polygons, d.BBox
		m.status = "loaded: " + filepath.Base(p)
	default:
		m.status = "unsupported file: " + ext
	}
}

// drawLine draws a line between two screen points into the lines buffer using '.'
func drawLine(buf *[]string, x0, y0, x1, y1 int) {
	if y0 < 0 && y1 < 0 {
		return
	}
	if y0 >= len(*buf) && y1 >= len(*buf) {
		return
	}
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
	// step-wise glyph selection
	for {
		// determine next step
		if x0 == x1 && y0 == y1 {
			// draw last point
			if y0 >= 0 && y0 < len(*buf) {
				r := []rune((*buf)[y0])
				if x0 >= 0 && x0 < len(r) {
					r[x0] = '•'
				}
				(*buf)[y0] = string(r)
			}
			break
		}
		e2 := 2 * err
		nx, ny := x0, y0
		moved := false
		if e2 >= dy {
			err += dy
			nx += sx
			moved = true
		}
		if e2 <= dx {
			err += dx
			ny += sy
			moved = true
		}
		// choose glyph based on movement
		glyph := '•'
		if nx != x0 && ny != y0 {
			if (nx-x0 > 0 && ny-y0 > 0) || (nx-x0 < 0 && ny-y0 < 0) {
				glyph = '╲'
			} else {
				glyph = '╱'
			}
		} else if nx != x0 {
			glyph = '─'
		} else if ny != y0 {
			glyph = '│'
		}
		x0, y0 = nx, ny
		if moved && y0 >= 0 && y0 < len(*buf) {
			r := []rune((*buf)[y0])
			if x0 >= 0 && x0 < len(r) {
				r[x0] = glyph
			}
			(*buf)[y0] = string(r)
		}
	}
}
