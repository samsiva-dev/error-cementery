package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorGray      = lipgloss.Color("#666666")
	colorDim       = lipgloss.Color("#444444")
	colorGreen     = lipgloss.Color("#50fa7b")
	colorYellow    = lipgloss.Color("#f1fa8c")
	colorRed       = lipgloss.Color("#ff5555")
	colorPurple    = lipgloss.Color("#bd93f9")
	colorCyan      = lipgloss.Color("#8be9fd")
	colorWhite     = lipgloss.Color("#f8f8f2")
	colorBg        = lipgloss.Color("#1a1a2e")
	colorBorder    = lipgloss.Color("#44475a")
	colorHighlight = lipgloss.Color("#6272a4")

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple)

	styleSubtle = lipgloss.NewStyle().
			Foreground(colorGray)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	styleFix = lipgloss.NewStyle().
			Foreground(colorGreen)

	styleMeta = lipgloss.NewStyle().
			Foreground(colorGray)

	styleTag = lipgloss.NewStyle().
			Foreground(colorCyan).
			Background(colorDim).
			Padding(0, 1)

	styleBadge = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	styleSelected = lipgloss.NewStyle().
			Background(colorHighlight).
			Foreground(colorWhite)

	styleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	styleCardSelected = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPurple).
				Padding(0, 1)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorDim)

	styleInputLabel = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true)

	styleInputActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPurple).
				Padding(0, 1)

	styleInputInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorder).
				Padding(0, 1)
)
