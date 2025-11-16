package geom

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
)

type BBox struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

// Data is a minimal geometry container for rendering
type Data struct {
	Points   [][2]float64
	Lines    [][][2]float64
	Polygons [][][][2]float64 // polygons with rings (first outer, following holes)
	BBox     BBox
}

// ParseWKT parses a subset of WKT and returns point vertices and bbox.
// Supported: POINT(x y), MULTIPOINT(x y, ...), LINESTRING(x y, ...), POLYGON((x y, ...))
func ParseWKT(wkt string) (points [][2]float64, bbox BBox, err error) {
	s := strings.TrimSpace(wkt)
	if s == "" {
		return nil, BBox{}, errors.New("empty wkt")
	}
	upper := strings.ToUpper
	advanceBBox := func(lon, lat float64) {
		pt := [2]float64{lon, lat}
		points = append(points, pt)
		if len(points) == 1 {
			bbox = BBox{MinX: lon, MinY: lat, MaxX: lon, MaxY: lat}
		} else {
			if lon < bbox.MinX {
				bbox.MinX = lon
			}
			if lat < bbox.MinY {
				bbox.MinY = lat
			}
			if lon > bbox.MaxX {
				bbox.MaxX = lon
			}
			if lat > bbox.MaxY {
				bbox.MaxY = lat
			}
		}
	}
	parseCoords := func(block string) {
		block = strings.TrimSpace(block)
		// split by comma into tuples "x y"
		for _, tup := range strings.Split(block, ",") {
			parts := strings.Fields(strings.TrimSpace(tup))
			if len(parts) < 2 {
				continue
			}
			x, err1 := strconv.ParseFloat(parts[0], 64)
			y, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 != nil || err2 != nil {
				continue
			}
			advanceBBox(x, y)
		}
	}
	switch {
	case strings.HasPrefix(upper(s), "POINT"):
		i := strings.Index(s, "(")
		j := strings.LastIndex(s, ")")
		if i < 0 || j <= i {
			return nil, BBox{}, errors.New("wkt point: invalid")
		}
		parseCoords(s[i+1 : j])
	case strings.HasPrefix(upper(s), "MULTIPOINT"):
		i := strings.Index(s, "(")
		j := strings.LastIndex(s, ")")
		if i < 0 || j <= i {
			return nil, BBox{}, errors.New("wkt multipoint: invalid")
		}
		parseCoords(s[i+1 : j])
	case strings.HasPrefix(upper(s), "LINESTRING"):
		i := strings.Index(s, "(")
		j := strings.LastIndex(s, ")")
		if i < 0 || j <= i {
			return nil, BBox{}, errors.New("wkt linestring: invalid")
		}
		parseCoords(s[i+1 : j])
	case strings.HasPrefix(upper(s), "POLYGON"):
		i := strings.Index(s, "((")
		j := strings.LastIndex(s, "))")
		if i < 0 || j <= i {
			return nil, BBox{}, errors.New("wkt polygon: invalid")
		}
		parseCoords(s[i+2 : j])
	default:
		return nil, BBox{}, errors.New("unsupported wkt type")
	}
	if len(points) == 0 {
		return nil, BBox{}, errors.New("wkt: no coordinates parsed")
	}
	return points, bbox, nil
}

// ParseWKTData returns Data for LINESTRING/POLYGON, or points for POINT/MULTIPOINT
func ParseWKTData(wkt string) (Data, error) {
	s := strings.TrimSpace(wkt)
	if s == "" {
		return Data{}, errors.New("empty wkt")
	}
	up := strings.ToUpper(s)
	var d Data
	mkbb := func(lon, lat float64) {
		if len(d.Points)+len(d.Lines)+len(d.Polygons) == 0 {
			d.BBox = BBox{MinX: lon, MinY: lat, MaxX: lon, MaxY: lat}
		} else {
			if lon < d.BBox.MinX {
				d.BBox.MinX = lon
			}
			if lat < d.BBox.MinY {
				d.BBox.MinY = lat
			}
			if lon > d.BBox.MaxX {
				d.BBox.MaxX = lon
			}
			if lat > d.BBox.MaxY {
				d.BBox.MaxY = lat
			}
		}
	}
	parseTuples := func(block string) [][2]float64 {
		var out [][2]float64
		for _, tup := range strings.Split(block, ",") {
			parts := strings.Fields(strings.TrimSpace(tup))
			if len(parts) < 2 {
				continue
			}
			x, e1 := strconv.ParseFloat(parts[0], 64)
			y, e2 := strconv.ParseFloat(parts[1], 64)
			if e1 != nil || e2 != nil {
				continue
			}
			out = append(out, [2]float64{x, y})
		}
		return out
	}
	switch {
	case strings.HasPrefix(up, "POINT"):
		i := strings.Index(s, "(")
		j := strings.LastIndex(s, ")")
		if i < 0 || j <= i {
			return Data{}, errors.New("wkt point: invalid")
		}
		pts := parseTuples(s[i+1 : j])
		d.Points = append(d.Points, pts...)
		for _, p := range pts {
			mkbb(p[0], p[1])
		}
		return d, nil
	case strings.HasPrefix(up, "MULTIPOINT"):
		i := strings.Index(s, "(")
		j := strings.LastIndex(s, ")")
		if i < 0 || j <= i {
			return Data{}, errors.New("wkt multipoint: invalid")
		}
		pts := parseTuples(s[i+1 : j])
		d.Points = append(d.Points, pts...)
		for _, p := range pts {
			mkbb(p[0], p[1])
		}
		return d, nil
	case strings.HasPrefix(up, "LINESTRING"):
		i := strings.Index(s, "(")
		j := strings.LastIndex(s, ")")
		if i < 0 || j <= i {
			return Data{}, errors.New("wkt linestring: invalid")
		}
		ls := parseTuples(s[i+1 : j])
		d.Lines = append(d.Lines, ls)
		for _, p := range ls {
			mkbb(p[0], p[1])
		}
		return d, nil
	case strings.HasPrefix(up, "POLYGON"):
		i := strings.Index(s, "((")
		j := strings.LastIndex(s, "))")
		if i < 0 || j <= i {
			return Data{}, errors.New("wkt polygon: invalid")
		}
		ringsStr := s[i+2 : j]
		// normalize spaces around ring separators
		ringsNorm := strings.ReplaceAll(ringsStr, "), (", "),(")
		ringsNorm = strings.ReplaceAll(ringsNorm, ") , (", "),(")
		ringParts := strings.Split(ringsNorm, "),(")
		var poly [][][2]float64
		for _, rp := range ringParts {
			pts := parseTuples(rp)
			poly = append(poly, pts)
			for _, p := range pts {
				mkbb(p[0], p[1])
			}
		}
		d.Polygons = append(d.Polygons, poly)
		return d, nil
	}
	return Data{}, errors.New("unsupported wkt type")
}

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

// LoadKML extracts Point coordinates from a KML file (Placemark > Point > coordinates).
// KML coordinates are "lon,lat[,alt]"; we ignore altitude.
func LoadKML(path string) (points [][2]float64, bbox BBox, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, BBox{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, BBox{}, err
	}

	type kmlPoint struct {
		Coordinates string `xml:"coordinates"`
	}
	type kmlPlacemark struct {
		Point *kmlPoint `xml:"Point"`
	}
	type kmlDoc struct {
		Placemarks []kmlPlacemark `xml:"Placemark"`
	}

	var doc kmlDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, BBox{}, err
	}
	add := func(lon, lat float64) {
		pt := [2]float64{lon, lat}
		points = append(points, pt)
		if len(points) == 1 {
			bbox = BBox{MinX: lon, MinY: lat, MaxX: lon, MaxY: lat}
		} else {
			if lon < bbox.MinX {
				bbox.MinX = lon
			}
			if lat < bbox.MinY {
				bbox.MinY = lat
			}
			if lon > bbox.MaxX {
				bbox.MaxX = lon
			}
			if lat > bbox.MaxY {
				bbox.MaxY = lat
			}
		}
	}
	for _, pm := range doc.Placemarks {
		if pm.Point == nil {
			continue
		}
		// coordinates may contain multiple tuples separated by spaces
		parts := strings.Fields(pm.Point.Coordinates)
		for _, tuple := range parts {
			vals := strings.Split(tuple, ",")
			if len(vals) < 2 {
				continue
			}
			lon, err1 := strconv.ParseFloat(strings.TrimSpace(vals[0]), 64)
			lat, err2 := strconv.ParseFloat(strings.TrimSpace(vals[1]), 64)
			if err1 != nil || err2 != nil {
				continue
			}
			add(lon, lat)
		}
	}
	if len(points) == 0 {
		return nil, BBox{}, errors.New("kml: no points found")
	}
	return points, bbox, nil
}

// LoadCSV reads a CSV with latitude/longitude columns and returns points.
// Column detection: lat|latitude|y and lon|lng|long|longitude|x (case-insensitive).
func LoadCSV(path string) (points [][2]float64, bbox BBox, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, BBox{}, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	recs, err := r.ReadAll()
	if err != nil {
		return nil, BBox{}, err
	}
	if len(recs) == 0 {
		return nil, BBox{}, errors.New("empty csv")
	}
	header := recs[0]
	idxLat, idxLon := -1, -1
	lower := func(s string) string {
		b := make([]rune, 0, len(s))
		for _, ch := range s {
			if ch >= 'A' && ch <= 'Z' {
				ch += 32
			}
			b = append(b, ch)
		}
		return string(b)
	}
	for i, h := range header {
		lh := lower(h)
		switch lh {
		case "lat", "latitude", "y":
			if idxLat == -1 {
				idxLat = i
			}
		case "lon", "lng", "long", "longitude", "x":
			if idxLon == -1 {
				idxLon = i
			}
		}
	}
	if idxLat == -1 || idxLon == -1 {
		return nil, BBox{}, errors.New("csv: latitude/longitude columns not found")
	}
	for _, row := range recs[1:] {
		if idxLon >= len(row) || idxLat >= len(row) {
			continue
		}
		lon, err1 := strconv.ParseFloat(strings.TrimSpace(row[idxLon]), 64)
		lat, err2 := strconv.ParseFloat(strings.TrimSpace(row[idxLat]), 64)
		if err1 != nil || err2 != nil {
			continue
		}
		pt := [2]float64{lon, lat}
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
	if len(points) == 0 {
		return nil, BBox{}, errors.New("csv: no valid points parsed")
	}
	return points, bbox, nil
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
