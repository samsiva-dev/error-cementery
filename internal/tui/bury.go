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
	tagHints    []string
	tagHintIdx  int
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

// computeTagHints updates tagHints based on the current partial tag being typed.
func (m *BuryModel) computeTagHints() {
	ti := m.fields[fieldTags].(textinput.Model)
	val := ti.Value()

	parts := strings.Split(val, ",")
	partial := strings.TrimSpace(parts[len(parts)-1])

	if partial == "" {
		m.tagHints = nil
		m.tagHintIdx = 0
		return
	}

	lp := strings.ToLower(partial)
	var hints []string
	for _, tag := range m.tagSuggest {
		lt := strings.ToLower(tag)
		if strings.HasPrefix(lt, lp) && lt != lp {
			hints = append(hints, tag)
			if len(hints) >= 5 {
				break
			}
		}
	}
	m.tagHints = hints
	if m.tagHintIdx >= len(hints) {
		m.tagHintIdx = 0
	}
}

// acceptTagSuggestion replaces the last partial tag token with the given suggestion.
func (m *BuryModel) acceptTagSuggestion(tag string) {
	ti := m.fields[fieldTags].(textinput.Model)
	val := ti.Value()

	idx := strings.LastIndex(val, ",")
	var newVal string
	if idx == -1 {
		newVal = tag + ","
	} else {
		newVal = val[:idx+1] + tag + ","
	}
	ti.SetValue(newVal)
	m.fields[fieldTags] = ti
	m.tagHints = nil
	m.tagHintIdx = 0
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
			m.tagHints = nil
			m.tagHintIdx = 0
			m.focusField(m.active)
		case "up":
			if m.active == fieldTags && len(m.tagHints) > 0 {
				if m.tagHintIdx > 0 {
					m.tagHintIdx--
				}
				return m, nil
			}
		case "down":
			if m.active == fieldTags && len(m.tagHints) > 0 {
				if m.tagHintIdx < len(m.tagHints)-1 {
					m.tagHintIdx++
				}
				return m, nil
			}
		case "enter":
			if m.active == fieldTags && len(m.tagHints) > 0 {
				m.acceptTagSuggestion(m.tagHints[m.tagHintIdx])
				return m, nil
			}
		case "tab", "ctrl+n":
			if m.active == fieldTags && len(m.tagHints) > 0 {
				m.acceptTagSuggestion(m.tagHints[m.tagHintIdx])
				return m, nil
			}
			m.tagHints = nil
			m.tagHintIdx = 0
			m.active = (m.active + 1) % fieldCount
			m.focusField(m.active)
			if m.active == fieldTags {
				m.computeTagHints()
			}
		case "shift+tab", "ctrl+p":
			m.tagHints = nil
			m.tagHintIdx = 0
			if m.active == 0 {
				m.active = fieldCount - 1
			} else {
				m.active--
			}
			m.focusField(m.active)
			if m.active == fieldTags {
				m.computeTagHints()
			}
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
		m.computeTagHints()
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
		sb.WriteString(field + "\n")

		// Show tag suggestions below the Tags field when active
		if i == fieldTags && m.active == fieldTags && len(m.tagHints) > 0 {
			sb.WriteString(styleHelp.Render("  suggestions: "))
			for j, hint := range m.tagHints {
				if j == m.tagHintIdx {
					sb.WriteString(styleSelected.Render(" " + hint + " "))
				} else {
					sb.WriteString(styleTag.Render(hint))
				}
				if j < len(m.tagHints)-1 {
					sb.WriteString("  ")
				}
			}
			sb.WriteString("\n" + styleHelp.Render("  ↑↓ navigate  tab/enter accept") + "\n")
		}

		sb.WriteString("\n")
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
