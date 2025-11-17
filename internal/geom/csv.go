package geom

import (
	"encoding/csv"
	"errors"
	"os"
	"strconv"
	"strings"
)

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
