package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"goemap/internal/tui"
)

func main() {
	var m tea.Model
	if len(os.Args) > 1 {
		m = tui.NewWithPath(os.Args[1])
	} else {
		m = tui.New()
	}
	if err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion()).Start(); err != nil {
		log.Fatal(err)
	}
}
