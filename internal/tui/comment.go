package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// CommentModel is a minimal TUI for adding a comment to a buried error.
type CommentModel struct {
	input     textarea.Model
	header    string
	submitted bool
	cancelled bool
	width     int
	height    int
}

// CommentResult holds the result of the comment TUI session.
type CommentResult struct {
	Text      string
	Submitted bool
}

func NewCommentModel(header string) CommentModel {
	ta := textarea.New()
	ta.Placeholder = "Add your comment here..."
	ta.SetWidth(70)
	ta.SetHeight(6)
	ta.Focus()

	return CommentModel{
		input:  ta,
		header: header,
		width:  80,
		height: 24,
	}
}

func (m CommentModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m CommentModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "esc":
			m.cancelled = true
			return m, tea.Quit
		case "ctrl+s", "ctrl+d":
			m.submitted = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m CommentModel) View() string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("  ⚰  Error Cemetery — Add Comment  ") + "\n\n")

	if m.header != "" {
		sb.WriteString(styleSubtle.Render(fmt.Sprintf("  on: %s", m.header)) + "\n\n")
	}

	sb.WriteString(styleInputLabel.Render("Comment") + "\n")
	sb.WriteString(styleInputActive.Render(m.input.View()) + "\n\n")

	help := styleHelp.Render("  ctrl+s save  esc/ctrl+c cancel")
	sb.WriteString(help)

	return sb.String()
}

func (m CommentModel) Result() CommentResult {
	return CommentResult{
		Text:      m.input.Value(),
		Submitted: m.submitted,
	}
}

// RunComment launches the comment TUI and returns the entered text and whether it was submitted.
func RunComment(header string) (CommentResult, error) {
	m := NewCommentModel(header)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return CommentResult{}, err
	}
	return final.(CommentModel).Result(), nil
}
