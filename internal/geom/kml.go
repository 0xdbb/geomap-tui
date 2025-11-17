package geom

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
)

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
