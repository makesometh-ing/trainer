package app

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"

	"github.com/makesometh-ing/trainer/internal/skills"
)

// skillsMode is which control in the Skills pane currently takes key input.
type skillsMode int

const (
	modeList skillsMode = iota
	modeSearch
	modeFilter
)

// originFilter narrows the skill list by where each skill came from.
type originFilter int

const (
	originAll originFilter = iota
	originRemote
	originLocal
)

func (f originFilter) label() string {
	switch f {
	case originRemote:
		return "Remote"
	case originLocal:
		return "Local"
	default:
		return "All"
	}
}

var originOptions = []originFilter{originAll, originRemote, originLocal}

func newSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "type to filter…"
	return ti
}

// visibleSkills returns the skills that pass the current origin filter and the
// current search text, in list order. It is the single source of truth for what
// the Skills pane shows; selection indexes into this slice.
func (m Model) visibleSkills() []skills.Skill {
	byOrigin := make([]skills.Skill, 0, len(m.skills))
	for _, s := range m.skills {
		switch m.filter {
		case originRemote:
			if s.Lock == nil {
				continue
			}
		case originLocal:
			if s.Lock != nil {
				continue
			}
		}
		byOrigin = append(byOrigin, s)
	}

	query := strings.TrimSpace(m.search.Value())
	if query == "" {
		return byOrigin
	}

	targets := make([]string, len(byOrigin))
	for i, s := range byOrigin {
		targets[i] = s.Name + " " + skillMeta(s)
	}
	// FindNoSort keeps the skills in name order rather than reordering by score.
	out := make([]skills.Skill, 0, len(byOrigin))
	for _, match := range fuzzy.FindNoSort(query, targets) {
		out = append(out, byOrigin[match.Index])
	}
	return out
}

// clampSelection keeps the selection on a listed skill after the visible set
// changes (a search or filter narrowed or widened it).
func (m *Model) clampSelection() {
	n := len(m.visibleSkills())
	switch {
	case n == 0:
		m.selected = 0
	case m.selected >= n:
		m.selected = n - 1
	case m.selected < 0:
		m.selected = 0
	}
}

// resetSearchFilter clears the search text and returns the filter to All.
func (m *Model) resetSearchFilter() {
	m.search.SetValue("")
	m.search.Blur()
	m.filter = originAll
	m.filterCursor = originAll
	m.skillsMode = modeList
	m.clampSelection()
	m.syncContent()
}

func (m Model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.search.Blur()
		m.skillsMode = modeList
		return m, nil
	case "esc":
		m.search.SetValue("")
		m.search.Blur()
		m.skillsMode = modeList
		m.clampSelection()
		m.syncContent()
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.clampSelection()
	m.syncContent()
	return m, cmd
}

func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "enter", "f":
		m.skillsMode = modeList
	case "l", "right":
		// The filter is laid out left to right (All Remote Local), so l/h move
		// the cursor along it; j/k stay list navigation.
		if m.filterCursor < originLocal {
			m.filterCursor++
		}
	case "h", "left":
		if m.filterCursor > originAll {
			m.filterCursor--
		}
	case "j", "down":
		m.moveDown()
	case "k", "up":
		m.moveUp()
	case "space":
		m.filter = m.filterCursor
		m.clampSelection()
		m.syncContent()
	case "c":
		m.filter = originAll
		m.filterCursor = originAll
		m.clampSelection()
		m.syncContent()
	}
	return m, nil
}

func (m Model) renderSearchBox() string {
	label := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("Search ")
	// Bound the input to the list content width so a long query scrolls within
	// the box rather than wrapping and growing the pane.
	in := m.search
	w := m.listWidth() - paneBorderPad - lipgloss.Width("Search ")
	if w < 1 {
		w = 1
	}
	in.SetWidth(w)
	return label + in.View()
}

func (m Model) renderFilter() string {
	label := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("Filter ")
	opts := make([]string, 0, len(originOptions))
	for _, o := range originOptions {
		marker := "○"
		if m.filter == o {
			marker = "●"
		}
		style := lipgloss.NewStyle().Foreground(m.theme.Fg)
		if m.skillsMode == modeFilter && m.filterCursor == o {
			style = lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
		}
		// A non-breaking space keeps the bullet visually separated from the label
		// while still making each option one token, so a narrow pane wraps between
		// options, never inside one.
		opts = append(opts, style.Render(marker+"\u00a0"+o.label()))
	}
	w := m.listWidth() - paneBorderPad
	if w < 1 {
		w = 1
	}
	return lipgloss.NewStyle().Width(w).Render(label + strings.Join(opts, " "))
}
