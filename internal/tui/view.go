package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	var mapView string
	if m.showAttrs {
		// Render attributes table centered in the map area
		// infer a reasonable width from columns
		colW := 0
		for _, c := range m.tbl.Columns() {
			colW += c.Width + 3
		}
		if colW == 0 {
			colW = min(60, contentWidth-6)
		}
		maxW := min(mapWidth, max(32, colW))
		m.tbl.SetWidth(maxW - 4)
		m.tbl.SetHeight(min(mapHeight-2, 20))
		attrsBox := boxStyle.Width(maxW).Render(m.tbl.View())
		mapView = lipgloss.Place(mapWidth, mapHeight, lipgloss.Center, lipgloss.Center, attrsBox)
	} else {
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
		mapView = lipgloss.NewStyle().Width(mapWidth).Height(mapHeight).Render(ascii)
	}

	// Build inspect popup box (center-left overlay, not in map column)
	popup := ""
	if m.inspectPopup != "" && !m.showAttrs {
		maxPopupW := min(48, contentWidth/2)
		if maxPopupW < 20 {
			maxPopupW = 20
		}
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).MaxWidth(maxPopupW).Render(m.inspectPopup)
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
	// mouse coords at bottom-right
	coords := ""
	if m.hoverHasGeo {
		coords = dimStyle.Render(fmt.Sprintf("  lon=%.5f lat=%.5f  ", m.hoverLon, m.hoverLat))
	}
	left := lipgloss.JoinHorizontal(lipgloss.Bottom, status, help)
	spacerW := max(0, contentWidth-lipgloss.Width(left)-lipgloss.Width(coords))
	right := lipgloss.Place(spacerW+lipgloss.Width(coords), 1, lipgloss.Right, lipgloss.Center, coords)
	footer := lipgloss.NewStyle().Width(contentWidth).Render(lipgloss.JoinHorizontal(lipgloss.Bottom, left, right))

	// Compose UI with popup overlay between header and body
	ui := lipgloss.JoinVertical(lipgloss.Left, header, popup, body, footer)
	return appStyle.Width(contentWidth).Height(m.height).Render(ui)
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
		"a attrs",
		"i inspect",
		"l layers",
		"h help",
		"q quit",
	}
	return dimStyle.Render("  " + strings.Join(keys, "  "))
}
