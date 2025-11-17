package geom

import (
	"errors"
	"strconv"
	"strings"
)

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
