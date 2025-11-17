package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	list "github.com/charmbracelet/bubbles/list"

	"goemap/internal/geom"
)

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
