package tui

import "strings"

func abs(v int) int {
    if v < 0 {
        return -v
    }
    return v
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func padRight(s string, n int) string {
    if n <= 0 {
        return s
    }
    return s + strings.Repeat(" ", n)
}

// drawLine draws a line between two screen points into the lines buffer using ASCII glyphs.
func drawLine(buf *[]string, x0, y0, x1, y1 int) {
    if y0 < 0 && y1 < 0 {
        return
    }
    if y0 >= len(*buf) && y1 >= len(*buf) {
        return
    }
    dx := abs(x1 - x0)
    sx := -1
    if x0 < x1 {
        sx = 1
    }
    dy := -abs(y1 - y0)
    sy := -1
    if y0 < y1 {
        sy = 1
    }
    err := dx + dy
    for {
        // draw current point
        if y0 >= 0 && y0 < len(*buf) {
            r := []rune((*buf)[y0])
            if x0 >= 0 && x0 < len(r) {
                r[x0] = '•'
            }
            (*buf)[y0] = string(r)
        }
        if x0 == x1 && y0 == y1 {
            break
        }
        e2 := 2 * err
        movedX, movedY := false, false
        if e2 >= dy {
            err += dy
            x0 += sx
            movedX = true
        }
        if e2 <= dx {
            err += dx
            y0 += sy
            movedY = true
        }
        // choose glyph based on movement
        if y0 >= 0 && y0 < len(*buf) {
            r := []rune((*buf)[y0])
            if x0 >= 0 && x0 < len(r) {
                glyph := '•'
                if movedX && movedY {
                    if (sx > 0 && sy > 0) || (sx < 0 && sy < 0) {
                        glyph = '╲'
                    } else {
                        glyph = '╱'
                    }
                } else if movedX {
                    glyph = '─'
                } else if movedY {
                    glyph = '│'
                }
                r[x0] = glyph
            }
            (*buf)[y0] = string(r)
        }
    }
}
