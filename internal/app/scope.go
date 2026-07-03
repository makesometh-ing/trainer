package app

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

// sectionOrder is the fixed top-to-bottom order of scope sections in the pane.
var sectionOrder = []skills.Section{skills.SectionGlobal, skills.SectionProject}

// clampScope keeps selectedScope within the current results slice after a
// rescan changes how many scopes exist.
func (m *Model) clampScope() {
	n := len(m.results)
	switch {
	case n == 0:
		m.selectedScope = 0
	case m.selectedScope >= n:
		m.selectedScope = n - 1
	case m.selectedScope < 0:
		m.selectedScope = 0
	}
}

// scopeIndices returns the result indices belonging to a section, in registry
// order (the order ScanAll produced them).
func (m Model) scopeIndices(section skills.Section) []int {
	var idxs []int
	for i, r := range m.results {
		if r.Scope.Section == section {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

// moveScope moves the scope selection by delta across the flat results slice,
// skipping nothing (section headers are render-only, not selectable rows). It is
// inert when there is at most one scope. Changing scope resets the skill
// selection and re-syncs the detail content.
func (m *Model) moveScope(delta int) {
	if len(m.results) < 2 {
		return
	}
	next := m.selectedScope + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.results) {
		next = len(m.results) - 1
	}
	if next != m.selectedScope {
		m.selectedScope = next
		m.selected = 0
		m.fileSel = 0
		m.subfocus = subfocusList
		m.clampSelection()
		m.syncContent()
	}
}

// renderScope draws the two-level Scope pane: one header per section that has
// scopes, and under it one row per scope with its skill count. The selected
// scope row carries an elevated highlight band. Empty sections are omitted.
func (m Model) renderScope() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("(1) Scope")
	lines := []string{title}

	textW := m.scopeWidth() - paneBorderPad
	if textW < 1 {
		textW = 1
	}

	for _, section := range sectionOrder {
		idxs := m.scopeIndices(section)
		if len(idxs) == 0 {
			continue
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.Fg).Bold(true).Render(string(section)))
		for _, i := range idxs {
			lines = append(lines, m.scopeRow(m.results[i], i == m.selectedScope, textW))
		}
	}

	return m.pane(paneScope, m.scopeWidth(), m.paneHeight(), strings.Join(lines, "\n"))
}

// scopeRow renders one scope leaf: an indented label on the left and its skill
// count right-aligned within the pane content width.
func (m Model) scopeRow(r skills.ScanResult, selected bool, textW int) string {
	label := "  " + r.Scope.Name
	count := strconv.Itoa(len(r.Skills))
	gap := textW - lipgloss.Width(label) - lipgloss.Width(count)
	if gap < 1 {
		gap = 1
	}
	row := label + strings.Repeat(" ", gap) + count

	if selected {
		return lipgloss.NewStyle().
			Foreground(m.theme.Accent).
			Background(m.theme.Elevated).
			Bold(true).
			Width(textW).
			Render(truncate(row, textW))
	}
	return lipgloss.NewStyle().Foreground(m.theme.Fg).Render(truncate(row, textW))
}
