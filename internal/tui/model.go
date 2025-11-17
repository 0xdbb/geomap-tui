package tui

import (
	list "github.com/charmbracelet/bubbles/list"
	table "github.com/charmbracelet/bubbles/table"
	textarea "github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"os"

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
	hovering    bool
	hoverCellX  int
	hoverCellY  int
	hoverMicX   int
	hoverMicY   int
	hoverHasGeo bool
	hoverLon    float64
	hoverLat    float64

	// attributes table
	showAttrs bool
	tbl       table.Model
	attrCols  []string
	attrRows  []table.Row
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
	// attributes table setup (columns will be inferred per dataset)
	m.tbl = table.New(table.WithFocused(true))
	m.tbl.SetHeight(12)
	m.refreshDir()
	return m
}

// NewWithPath preloads a file's data at launch.
func NewWithPath(path string) Model {
	m := New()
	m.loadPath(path)
	return m
}

func (m Model) Init() tea.Cmd { return nil }
