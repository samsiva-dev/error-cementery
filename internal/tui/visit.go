package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/samsiva-dev/error-cemetery/internal/db"
	"github.com/samsiva-dev/error-cemetery/internal/match"
)

type VisitModel struct {
	all      []db.Burial
	filtered []db.Burial
	cursor   int
	filter   textinput.Model
	expanded bool
	width    int
	height   int
}

func NewVisitModel(burials []db.Burial) VisitModel {
	fi := textinput.New()
	fi.Placeholder = "filter by tag or keyword..."
	fi.Width = 40
	fi.Prompt = "> "
	return VisitModel{
		all:      burials,
		filtered: burials,
		filter:   fi,
		width:    80,
		height:   24,
	}
}

func (m VisitModel) Init() tea.Cmd {
	return nil
}

func (m VisitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.filter.Focused() {
			switch msg.String() {
			case "esc", "enter":
				m.filter.Blur()
			default:
				var cmd tea.Cmd
				m.filter, cmd = m.filter.Update(msg)
				m.applyFilter()
				m.cursor = 0
				return m, cmd
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.expanded = false
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.expanded = false
			}
		case "enter":
			m.expanded = !m.expanded
		case "/":
			m.filter.Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m *VisitModel) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	if q == "" {
		m.filtered = m.all
		return
	}
	var result []db.Burial
	for _, b := range m.all {
		if strings.Contains(strings.ToLower(b.ErrorText), q) ||
			strings.Contains(strings.ToLower(b.FixText), q) ||
			strings.Contains(strings.ToLower(b.Tags), q) ||
			strings.Contains(strings.ToLower(b.Context), q) {
			result = append(result, b)
		}
	}
	m.filtered = result
}

func (m VisitModel) View() string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("  ⚰  Error Cemetery — Visit  ") + "\n\n")

	if m.filter.Focused() {
		sb.WriteString(styleInputActive.Render(m.filter.View()) + "\n\n")
	} else {
		hint := styleHelp.Render(" (press / to filter)")
		if val := m.filter.Value(); val != "" {
			hint = styleSubtle.Render(" filter: ") + styleBadge.Render(val)
		}
		sb.WriteString(styleInputInactive.Render(m.filter.View()) + hint + "\n\n")
	}

	if len(m.filtered) == 0 {
		sb.WriteString(styleSubtle.Render("  No graves match the filter.\n"))
	}

	cardWidth := m.width - 4
	if cardWidth < 40 {
		cardWidth = 40
	}

	visibleLines := m.height - 10
	if visibleLines < 5 {
		visibleLines = 5
	}

	start := 0
	if m.cursor > 5 {
		start = m.cursor - 5
	}

	for i := start; i < len(m.filtered); i++ {
		r := match.MatchResult{Burial: m.filtered[i], MatchType: ""}
		selected := i == m.cursor
		card := renderGravestone(r, selected, m.expanded && selected, cardWidth)
		sb.WriteString(card + "\n")
		lines := strings.Count(card, "\n") + 1
		visibleLines -= lines
		if visibleLines <= 0 {
			break
		}
	}

	topTag := topTagFrom(m.all)
	footer := fmt.Sprintf("  %d graves", len(m.all))
	if topTag != "" {
		footer += "  |  most tagged: " + topTag
	}
	sb.WriteString("\n" + styleMeta.Render(footer))
	sb.WriteString("\n" + styleHelp.Render("  ↑↓/jk navigate  enter expand  / filter  q quit"))

	return sb.String()
}

func topTagFrom(burials []db.Burial) string {
	counts := map[string]int{}
	for _, b := range burials {
		for _, t := range strings.Split(b.Tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				counts[t]++
			}
		}
	}
	best, bestCount := "", 0
	for t, c := range counts {
		if c > bestCount {
			best, bestCount = t, c
		}
	}
	if best != "" {
		return fmt.Sprintf("%s (%d×)", best, bestCount)
	}
	return ""
}

func RunVisit(burials []db.Burial) error {
	m := NewVisitModel(burials)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
