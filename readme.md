### geomap

A terminal-based geospatial file viewer in Go.

Visualize GeoJSON, Shapefiles, CSV with coordinates, and KML files directly in your terminal.
Includes a toggleable file explorer, ASCII map view, and optional spatial index visualization in a separate binary.

### Features

- View spatial files (.geojson, .shp, .csv, .kml) in ASCII

- Pan and zoom the map directly in terminal

- Toggle file explorer sidebar to browse directory and open other spatial files

- Inspect feature properties

- Layer visibility toggling

| Key       | Action                                  |
| --------- | --------------------------------------- |
| ↑ ↓ ← →   | Pan map / move cursor                   |
| `+` / `-` | Zoom in / zoom out                      |
| `Tab`     | Toggle sidebar (file explorer)          |
| `Enter`   | Open selected file in explorer          |
| `i`       | Show properties of feature under cursor |
| `l`       | Toggle layer visibility                 |
| `q`       | Quit the application                    |
| `h`       | Show help / keybindings                 |
