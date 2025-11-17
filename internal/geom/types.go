package geom

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
