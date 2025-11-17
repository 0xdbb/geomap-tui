package tui

import (
	"fmt"
	list "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"path/filepath"
	"strings"

	"goemap/internal/geom"
)

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
				// reset viewport and focus layers for immediate visibility
				m.zoom = 1.0
				m.offsetX, m.offsetY = 0, 0
				m.showPolys = len(m.polygons) > 0 || (strings.HasPrefix(strings.ToUpper(w), "POLYGON"))
				m.showLines = len(m.lines) > 0 && !m.showPolys
				m.showPoints = len(m.points) > 0 && !m.showPolys
				m.status = fmt.Sprintf("rendered WKT  counts: pts=%d ls=%d poly=%d", len(m.points), len(m.lines), len(m.polygons))
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
		case "a":
			m.showAttrs = !m.showAttrs
			if m.showAttrs {
				m.refreshAttrsFromCurrent()
			}
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

		mapWidth := contentWidth - sidebarWidth - 1
		if mapWidth < 10 {
			mapWidth = 10
		}
		mapHeight := contentHeight
		mapOriginX := sidebarWidth + func() int {
			if m.showSidebar {
				return 1
			}
			return 0
		}()
		mapOriginY := headerHeight
		// mouse cell within map?
		cx, cy := msg.X, msg.Y
		if cx >= mapOriginX && cx < mapOriginX+mapWidth && cy >= mapOriginY && cy < mapOriginY+mapHeight {
			m.hovering = true
			m.hoverCellX = cx - mapOriginX
			m.hoverCellY = cy - mapOriginY
			// compute lon/lat for footer
			if lon, lat, ok := m.cellToLonLat(m.hoverCellX, m.hoverCellY, mapWidth, mapHeight); ok {
				m.hoverHasGeo = true
				m.hoverLon = lon
				m.hoverLat = lat
			} else {
				m.hoverHasGeo = false
			}
			// find nearest vertex (points + line vertices + polygon vertices) using micro coords
			hxMic := m.hoverCellX * 2
			hyMic := m.hoverCellY * 4
			best := 1<<31 - 1
			bx, by := hxMic, hyMic
			// points
			for _, p := range m.points {
				mx, my, ok := m.screenXYMicro(p[0], p[1], mapWidth, mapHeight)
				if !ok {
					continue
				}
				dx := mx - hxMic
				dy := my - hyMic
				d := dx*dx + dy*dy
				if d < best {
					best = d
					bx, by = mx, my
				}
			}
			// lines
			for _, ls := range m.lines {
				for _, p := range ls {
					mx, my, ok := m.screenXYMicro(p[0], p[1], mapWidth, mapHeight)
					if !ok {
						continue
					}
					dx := mx - hxMic
					dy := my - hyMic
					d := dx*dx + dy*dy
					if d < best {
						best = d
						bx, by = mx, my
					}
				}
			}
			// polygons
			for _, poly := range m.polygons {
				for _, ring := range poly {
					for _, p := range ring {
						mx, my, ok := m.screenXYMicro(p[0], p[1], mapWidth, mapHeight)
						if !ok {
							continue
						}
						dx := mx - hxMic
						dy := my - hyMic
						d := dx*dx + dy*dy
						if d < best {
							best = d
							bx, by = mx, my
						}
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
