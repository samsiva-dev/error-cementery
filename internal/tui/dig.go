package tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/samsiva-dev/error-cemetery/internal/match"
)

type DigModel struct {
	results  []match.MatchResult
	cursor   int
	expanded bool
	width    int
	height   int
	err      string
	copied   bool
}

func NewDigModel(results []match.MatchResult) DigModel {
	return DigModel{results: results, width: 80, height: 24}
}

func (m DigModel) Init() tea.Cmd {
	return nil
}

func (m DigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.copied = false

	case tea.KeyMsg:
		m.copied = false
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.expanded = false
			}
		case "down", "j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
				m.expanded = false
			}
		case "enter":
			m.expanded = !m.expanded
		case "y":
			if len(m.results) > 0 {
				fix := m.results[m.cursor].Burial.FixText
				if err := clipboard.WriteAll(fix); err != nil {
					m.err = "clipboard unavailable"
				} else {
					m.copied = true
				}
			}
		case "D":
			// Delete is handled by the caller via a message; signal via quit with state
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m DigModel) View() string {
	if len(m.results) == 0 {
		return styleSubtle.Render("\n  No matching graves found. Try `cemetery bury` to add one.\n")
	}

	var sb strings.Builder

	header := styleTitle.Render("  ⚰  Error Cemetery — Dig Results  ")
	sb.WriteString(header + "\n\n")

	cardWidth := m.width - 4
	if cardWidth < 40 {
		cardWidth = 40
	}

	for i, r := range m.results {
		selected := i == m.cursor
		sb.WriteString(renderGravestone(r, selected, m.expanded && selected, cardWidth))
		sb.WriteString("\n")
	}

	if m.copied {
		sb.WriteString("\n" + styleFix.Render("  ✓ Fix copied to clipboard!") + "\n")
	}
	if m.err != "" {
		sb.WriteString("\n" + styleError.Render("  ✗ "+m.err) + "\n")
	}

	help := styleHelp.Render("  ↑↓/jk navigate  enter expand  y copy fix  D delete  q quit")
	sb.WriteString("\n" + help)

	return sb.String()
}

func renderGravestone(r match.MatchResult, selected, expanded bool, width int) string {
	b := r.Burial

	badge := matchBadge(r.MatchType)
	errorLine := truncate(b.ErrorText, width-len(badge)-6)

	firstLine := styleError.Render(errorLine) + "  " + badge

	fixLines := b.FixText
	if !expanded {
		fixLines = truncate(fixLines, width-8)
	}

	metaParts := []string{
		fmt.Sprintf("#%d", b.ID),
		b.BuriedAt.Format("02 Jan 2006"),
	}
	if b.TimesDug > 0 {
		metaParts = append(metaParts, fmt.Sprintf("dug %d×", b.TimesDug))
	}
	if b.Context != "" {
		metaParts = append(metaParts, b.Context)
	}
	meta := styleMeta.Render(strings.Join(metaParts, "  ·  "))

	var tags string
	if b.Tags != "" {
		tagList := strings.Split(b.Tags, ",")
		var rendered []string
		for _, t := range tagList {
			t = strings.TrimSpace(t)
			if t != "" {
				rendered = append(rendered, styleTag.Render(t))
			}
		}
		tags = strings.Join(rendered, " ")
	}

	var content strings.Builder
	content.WriteString(firstLine + "\n\n")
	content.WriteString(styleFix.Render("FIX  ") + fixLines + "\n\n")
	content.WriteString(meta)
	if tags != "" {
		content.WriteString("\n" + tags)
	}

	style := styleCard
	if selected {
		style = styleCardSelected
	}
	return style.Width(width).Render(content.String())
}

func matchBadge(t string) string {
	switch t {
	case "exact":
		return styleBadge.Foreground(colorGreen).Render("[exact]")
	case "fts":
		return styleBadge.Foreground(colorYellow).Render("[fts]")
	case "semantic":
		return styleBadge.Foreground(colorPurple).Render("[semantic]")
	}
	return ""
}

func truncate(s string, max int) string {
	// collapse newlines for single-line display
	s = strings.ReplaceAll(s, "\n", " ")
	if max <= 3 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

func RunDig(results []match.MatchResult) error {
	m := NewDigModel(results)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
