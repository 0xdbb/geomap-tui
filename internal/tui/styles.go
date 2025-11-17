package tui

import "github.com/charmbracelet/lipgloss"

// Styles
var (
	baseFg    = lipgloss.Color("#E6E6E6")
	baseDimFg = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#6B7280"}
	accentFg  = lipgloss.Color("#7C3AED")
	subtleBg  = lipgloss.Color("#0B0F14")
	panelBg   = lipgloss.Color("#0F141A")
	borderCol = lipgloss.Color("#243141")

	appStyle   = lipgloss.NewStyle().Foreground(baseFg)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderCol).Padding(0, 1)
	titleStyle = lipgloss.NewStyle().Foreground(accentFg).Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(baseDimFg)
)
