package geom

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

// LoadGeo reads a GeoJSON file and returns Data (points, lines, polygons)
func LoadGeo(path string) (Data, error) {
	f, err := os.Open(path)
	if err != nil {
		return Data{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return Data{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Data{}, err
	}
	var d Data
	addPt := func(pt [2]float64) {
		if len(d.Points)+len(d.Lines)+len(d.Polygons) == 0 {
			d.BBox = BBox{MinX: pt[0], MinY: pt[1], MaxX: pt[0], MaxY: pt[1]}
		} else {
			if pt[0] < d.BBox.MinX {
				d.BBox.MinX = pt[0]
			}
			if pt[1] < d.BBox.MinY {
				d.BBox.MinY = pt[1]
			}
			if pt[0] > d.BBox.MaxX {
				d.BBox.MaxX = pt[0]
			}
			if pt[1] > d.BBox.MaxY {
				d.BBox.MaxY = pt[1]
			}
		}
		d.Points = append(d.Points, pt)
	}
	addLine := func(ls [][2]float64) {
		d.Lines = append(d.Lines, ls)
		for _, p := range ls {
			addPt(p)
		}
	}
	addPoly := func(poly [][][2]float64) {
		d.Polygons = append(d.Polygons, poly)
		for _, ring := range poly {
			for _, p := range ring {
				addPt(p)
			}
		}
	}
	parsePoint := func(v any) (pt [2]float64, ok bool) {
		if a, ok := v.([]any); ok && len(a) >= 2 {
			lon, lok := a[0].(float64)
			lat, aok := a[1].(float64)
			if lok && aok {
				return [2]float64{lon, lat}, true
			}
		}
		return [2]float64{}, false
	}
	parseArrayPoints := func(v any) (pts [][2]float64, ok bool) {
		arr, ok := v.([]any)
		if !ok {
			return nil, false
		}
		for _, el := range arr {
			if pt, ok := parsePoint(el); ok {
				pts = append(pts, pt)
			}
		}
		return pts, true
	}
	parseLineString := func(v any) (ls [][2]float64, ok bool) { return parseArrayPoints(v) }
	parseMultiLineString := func(v any) (m [][][2]float64, ok bool) {
		arr, ok := v.([]any)
		if !ok {
			return nil, false
		}
		for _, el := range arr {
			if ls, ok := parseLineString(el); ok {
				m = append(m, ls)
			}
		}
		return m, true
	}
	parsePolygon := func(v any) (poly [][][2]float64, ok bool) {
		arr, ok := v.([]any)
		if !ok {
			return nil, false
		}
		for _, ring := range arr {
			if ls, ok := parseLineString(ring); ok {
				poly = append(poly, ls)
			}
		}
		return poly, true
	}
	parseMultiPolygon := func(v any) (mp [][][][2]float64, ok bool) {
		arr, ok := v.([]any)
		if !ok {
			return nil, false
		}
		for _, el := range arr {
			if poly, ok := parsePolygon(el); ok {
				mp = append(mp, poly)
			}
		}
		return mp, true
	}
	var walkGeom func(g map[string]any)
	walkGeom = func(g map[string]any) {
		gt, _ := g["type"].(string)
		switch gt {
		case "Point":
			if pt, ok := parsePoint(g["coordinates"]); ok {
				addPt(pt)
			}
		case "MultiPoint":
			if pts, ok := parseArrayPoints(g["coordinates"]); ok {
				for _, p := range pts {
					addPt(p)
				}
			}
		case "LineString":
			if ls, ok := parseLineString(g["coordinates"]); ok {
				addLine(ls)
			}
		case "MultiLineString":
			if mls, ok := parseMultiLineString(g["coordinates"]); ok {
				for _, ls := range mls {
					addLine(ls)
				}
			}
		case "Polygon":
			if poly, ok := parsePolygon(g["coordinates"]); ok {
				addPoly(poly)
			}
		case "MultiPolygon":
			if mp, ok := parseMultiPolygon(g["coordinates"]); ok {
				for _, poly := range mp {
					addPoly(poly)
				}
			}
		}
	}
	t, _ := raw["type"].(string)
	switch t {
	case "Feature":
		if g, ok := raw["geometry"].(map[string]any); ok {
			walkGeom(g)
		}
	case "FeatureCollection":
		if fs, ok := raw["features"].([]any); ok {
			for _, f := range fs {
				if fm, ok := f.(map[string]any); ok {
					if g, ok := fm["geometry"].(map[string]any); ok {
						walkGeom(g)
					}
				}
			}
		}
	default:
		if len(raw) > 0 {
			walkGeom(raw)
		}
	}
	if len(d.Points) == 0 && len(d.Lines) == 0 && len(d.Polygons) == 0 {
		return Data{}, errors.New("no geometries found")
	}
	return d, nil
}

// LoadGeoJSON extracts point coordinates from a GeoJSON file.
// Supports: Point, MultiPoint, Feature, FeatureCollection of Points/MultiPoints.
func LoadGeoJSON(path string) (points [][2]float64, bbox BBox, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, BBox{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, BBox{}, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, BBox{}, err
	}
	t, _ := raw["type"].(string)
	if t == "" {
		return nil, BBox{}, errors.New("invalid geojson: missing type")
	}

	add := func(pt [2]float64) {
		points = append(points, pt)
		if len(points) == 1 {
			bbox = BBox{MinX: pt[0], MinY: pt[1], MaxX: pt[0], MaxY: pt[1]}
		} else {
			if pt[0] < bbox.MinX {
				bbox.MinX = pt[0]
			}
			if pt[1] < bbox.MinY {
				bbox.MinY = pt[1]
			}
			if pt[0] > bbox.MaxX {
				bbox.MaxX = pt[0]
			}
			if pt[1] > bbox.MaxY {
				bbox.MaxY = pt[1]
			}
		}
	}

	// Helpers to parse coordinates
	parsePoint := func(v any) (pt [2]float64, ok bool) {
		if a, ok := v.([]any); ok && len(a) >= 2 {
			lon, lok := a[0].(float64)
			lat, aok := a[1].(float64)
			if lok && aok {
				return [2]float64{lon, lat}, true
			}
		}
		return [2]float64{}, false
	}
	parseMulti := func(v any) (pts [][2]float64, ok bool) {
		arr, ok := v.([]any)
		if !ok {
			return nil, false
		}
		for _, el := range arr {
			if pt, ok := parsePoint(el); ok {
				pts = append(pts, pt)
			}
		}
		return pts, true
	}

	switch t {
	case "Point":
		if pt, ok := parsePoint(raw["coordinates"]); ok {
			add(pt)
		}
	case "MultiPoint":
		if pts, ok := parseMulti(raw["coordinates"]); ok {
			for _, p := range pts {
				add(p)
			}
		}
	case "Feature":
		if g, ok := raw["geometry"].(map[string]any); ok {
			gt, _ := g["type"].(string)
			switch gt {
			case "Point":
				if pt, ok := parsePoint(g["coordinates"]); ok {
					add(pt)
				}
			case "MultiPoint":
				if pts, ok := parseMulti(g["coordinates"]); ok {
					for _, p := range pts {
						add(p)
					}
				}
			}
		}
	case "FeatureCollection":
		if fs, ok := raw["features"].([]any); ok {
			for _, f := range fs {
				fm, _ := f.(map[string]any)
				if g, ok := fm["geometry"].(map[string]any); ok {
					gt, _ := g["type"].(string)
					switch gt {
					case "Point":
						if pt, ok := parsePoint(g["coordinates"]); ok {
							add(pt)
						}
					case "MultiPoint":
						if pts, ok := parseMulti(g["coordinates"]); ok {
							for _, p := range pts {
								add(p)
							}
						}
					}
				}
			}
		}
	default:
		return nil, BBox{}, errors.New("unsupported geojson type: " + t)
	}

	if len(points) == 0 {
		return nil, BBox{}, errors.New("no points found in geojson")
	}
	return points, bbox, nil
}
