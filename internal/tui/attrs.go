package tui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	table "github.com/charmbracelet/bubbles/table"
)

// refreshAttrsFromCurrent rebuilds the table columns/rows from the currently selected path
func (m *Model) refreshAttrsFromCurrent() {
	cols, rows := m.buildAttributes()
	// If there are no columns or rows, disable attributes view to avoid rendering panics
	if len(cols) == 0 || len(rows) == 0 {
		// Do not touch table internals here to avoid re-render during SetColumns
		m.showAttrs = false
		m.status = "no attributes for current dataset"
		return
	}
	// map to bubbles table columns/rows
	tcols := make([]table.Column, 0, len(cols)+1)
	tcols = append(tcols, table.Column{Title: "#", Width: 4})
	maxColW := 24
	for _, c := range cols {
		w := len(c) + 2
		if w > maxColW {
			w = maxColW
		}
		tcols = append(tcols, table.Column{Title: c, Width: w})
	}
	trows := make([]table.Row, 0, len(rows))
	for i, r := range rows {
		row := make([]string, 0, len(r)+1)
		row = append(row, fmt.Sprintf("%d", i+1))
		row = append(row, r...)
		trows = append(trows, table.Row(row))
	}
	// Normalize each row to match the number of table columns
	colCount := len(tcols)
	for i := range trows {
		cells := []string(trows[i])
		if len(cells) < colCount {
			// pad
			pad := make([]string, colCount-len(cells))
			cells = append(cells, pad...)
		} else if len(cells) > colCount {
			// truncate
			cells = cells[:colCount]
		}
		trows[i] = table.Row(cells)
	}
    // Avoid transient mismatch: clear rows, set columns, then set rows
    m.tbl.SetRows(nil)
    m.tbl.SetColumns(tcols)
    m.tbl.SetRows(trows)
}

// buildAttributes inspects the current dataset and returns (columns, rows)
func (m *Model) buildAttributes() ([]string, [][]string) {
	p := m.selPath
	if p == "" {
		// pasted WKT or ephemeral data: no attributes available
		return []string{}, [][]string{}
	}
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".geojson", ".json":
		return buildAttrsGeoJSON(p)
	case ".csv":
		return buildAttrsCSV(p)
	default:
		// fallback: just bbox/summary as a single-row table
		cols := []string{"name", "path", "bbox", "points", "lines", "polygons"}
		vals := []string{filepath.Base(p), p, fmt.Sprintf("[%.5f,%.5f,%.5f,%.5f]", m.bbox.MinX, m.bbox.MinY, m.bbox.MaxX, m.bbox.MaxY), fmt.Sprintf("%d", len(m.points)), fmt.Sprintf("%d", len(m.lines)), fmt.Sprintf("%d", len(m.polygons))}
		return cols, [][]string{vals}
	}
}

// buildAttrsGeoJSON collects properties across all features and unions the keys
func buildAttrsGeoJSON(path string) ([]string, [][]string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return []string{}, [][]string{}
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return []string{}, [][]string{}
	}
	// collect features array
	var features []any
	if t, _ := raw["type"].(string); t == "FeatureCollection" {
		if fs, ok := raw["features"].([]any); ok {
			features = fs
		}
	} else if t == "Feature" {
		features = []any{raw}
	} else {
		// geometry only; no attributes
		return []string{"lon", "lat"}, [][]string{}
	}
	// union property keys
	order := []string{}
	seen := map[string]bool{}
	propsList := make([]map[string]any, 0, len(features))
	for _, f := range features {
		fm, ok := f.(map[string]any)
		if !ok {
			continue
		}
		pm, _ := fm["properties"].(map[string]any)
		if pm == nil {
			pm = map[string]any{}
		}
		propsList = append(propsList, pm)
		for k := range pm {
			if !seen[k] {
				seen[k] = true
				order = append(order, k)
			}
		}
	}
	// rows
	rows := make([][]string, 0, len(propsList))
	for _, pm := range propsList {
		vals := make([]string, 0, len(order))
		for _, k := range order {
			v := pm[k]
			switch t := v.(type) {
			case nil:
				vals = append(vals, "")
			case string:
				vals = append(vals, t)
			case float64:
				vals = append(vals, fmt.Sprintf("%g", t))
			case bool:
				if t {
					vals = append(vals, "true")
				} else {
					vals = append(vals, "false")
				}
			default:
				bs, _ := json.Marshal(t)
				vals = append(vals, string(bs))
			}
		}
		rows = append(rows, vals)
	}
	return order, rows
}

// buildAttrsCSV returns header as columns and each row as values
func buildAttrsCSV(path string) ([]string, [][]string) {
	f, err := os.Open(path)
	if err != nil {
		return []string{}, [][]string{}
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	recs, err := r.ReadAll()
	if err != nil || len(recs) == 0 {
		return []string{}, [][]string{}
	}
	header := recs[0]
	rows := make([][]string, 0, len(recs)-1)
	for _, row := range recs[1:] {
		vals := make([]string, len(header))
		copy(vals, row)
		rows = append(rows, vals)
	}
	return header, rows
}
