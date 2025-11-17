### geomap

A terminal-based geospatial file viewer in Go.

Visualize GeoJSON files directly in your terminal.
Includes a toggleable file explorer, ASCII map view, and optional spatial index visualization in a separate binary.

### Demo

[![Watch the demo](https://ik.imagekit.io/routing/screenshot-2025-11-16_22-05-05.png)](https://youtu.be/OZKMqFq0YLc)

### Features

- View spatial files (currently supports only .geojson) in ASCII

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
| `p`       | Paste wkt to render                  |

### Quickstart

1. Install dependencies

```
go mod tidy
```

2. Run the app

```
go run ./cmd/geomap
```

3. Build (optional)

```
go build -o geomap ./cmd/geomap
```

### Usage

```
geomap
```

- Run with a spatial file to render on launch:

```
geomap spatial_line.geojson
```

- Toggle the file explorer with `Tab`. The explorer lists only files in the current working directory (no parent or subdirectories) and filters to supported types.
