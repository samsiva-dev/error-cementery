package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/samsiva-dev/error-cemetery/internal/db"
)

type buryField int

const (
	fieldError buryField = iota
	fieldFix
	fieldContext
	fieldTags
	fieldCount
)

type BuryModel struct {
	fields      [fieldCount]interface{} // textarea or textinput
	active      buryField
	submitted   bool
	cancelled   bool
	tagSuggest  []string
	width       int
	height      int
}

type BuryResult struct {
	Input     db.BuryInput
	Submitted bool
}

func NewBuryModel(prefillError string, existingTags []string) BuryModel {
	m := BuryModel{
		tagSuggest: existingTags,
		width:      80,
		height:     24,
	}

	errTA := textarea.New()
	errTA.Placeholder = "Paste the error message here..."
	errTA.SetWidth(70)
	errTA.SetHeight(5)
	errTA.SetValue(prefillError)
	errTA.Focus()
	m.fields[fieldError] = errTA

	fixTA := textarea.New()
	fixTA.Placeholder = "What did you do to fix it?"
	fixTA.SetWidth(70)
	fixTA.SetHeight(4)
	m.fields[fieldFix] = fixTA

	ctxTI := textinput.New()
	ctxTI.Placeholder = "file/project/notes (optional)"
	ctxTI.Width = 70
	m.fields[fieldContext] = ctxTI

	tagsTI := textinput.New()
	tagsTI.Placeholder = "go,nil-pointer,async (comma-separated)"
	tagsTI.Width = 70
	m.fields[fieldTags] = tagsTI

	return m
}

func (m BuryModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m BuryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.active == fieldError {
				m.cancelled = true
				return m, tea.Quit
			}
			m.active--
			m.focusField(m.active)
		case "tab", "ctrl+n":
			m.active = (m.active + 1) % fieldCount
			m.focusField(m.active)
		case "shift+tab", "ctrl+p":
			if m.active == 0 {
				m.active = fieldCount - 1
			} else {
				m.active--
			}
			m.focusField(m.active)
		case "ctrl+s", "ctrl+d":
			m.submitted = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	switch m.active {
	case fieldError:
		ta := m.fields[fieldError].(textarea.Model)
		ta, cmd = ta.Update(msg)
		m.fields[fieldError] = ta
	case fieldFix:
		ta := m.fields[fieldFix].(textarea.Model)
		ta, cmd = ta.Update(msg)
		m.fields[fieldFix] = ta
	case fieldContext:
		ti := m.fields[fieldContext].(textinput.Model)
		ti, cmd = ti.Update(msg)
		m.fields[fieldContext] = ti
	case fieldTags:
		ti := m.fields[fieldTags].(textinput.Model)
		ti, cmd = ti.Update(msg)
		m.fields[fieldTags] = ti
	}
	return m, cmd
}

func (m *BuryModel) focusField(f buryField) {
	for i := range m.fields {
		switch v := m.fields[i].(type) {
		case textarea.Model:
			v.Blur()
			m.fields[i] = v
		case textinput.Model:
			v.Blur()
			m.fields[i] = v
		}
	}
	switch v := m.fields[f].(type) {
	case textarea.Model:
		v.Focus()
		m.fields[f] = v
	case textinput.Model:
		v.Focus()
		m.fields[f] = v
	}
}

func (m BuryModel) View() string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("  ⚰  Error Cemetery — Bury  ") + "\n\n")

	labels := []string{"Error", "Fix", "Context", "Tags"}
	for i := buryField(0); i < fieldCount; i++ {
		label := styleInputLabel.Render(labels[i])
		sb.WriteString(label + "\n")

		var field string
		isActive := m.active == i
		switch v := m.fields[i].(type) {
		case textarea.Model:
			if isActive {
				field = styleInputActive.Render(v.View())
			} else {
				field = styleInputInactive.Render(v.View())
			}
		case textinput.Model:
			if isActive {
				field = styleInputActive.Render(v.View())
			} else {
				field = styleInputInactive.Render(v.View())
			}
		}
		sb.WriteString(field + "\n\n")
	}

	help := styleHelp.Render("  tab/ctrl+n next  shift+tab/ctrl+p prev  ctrl+s submit  esc/ctrl+c cancel")
	sb.WriteString(help)

	return sb.String()
}

func (m BuryModel) Result() BuryResult {
	return BuryResult{
		Submitted: m.submitted,
		Input: db.BuryInput{
			ErrorText: m.fields[fieldError].(textarea.Model).Value(),
			FixText:   m.fields[fieldFix].(textarea.Model).Value(),
			Context:   m.fields[fieldContext].(textinput.Model).Value(),
			Tags:      m.fields[fieldTags].(textinput.Model).Value(),
		},
	}
}

func RunBury(prefill string, existingTags []string) (BuryResult, error) {
	m := NewBuryModel(prefill, existingTags)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return BuryResult{}, err
	}
	return final.(BuryModel).Result(), nil
}
