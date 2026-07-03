package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) handleHelpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "?":
		m.help = false
	}
	return m, nil
}

// renderHelp draws the help modal: every key binding (from the shared key.Binding
// definitions) grouped by context, with one key column width across all groups so
// the descriptions line up, one accent for headings, one color for keys, and a
// single elevated surface.
func (m Model) renderHelp() string {
	groups := m.keys.helpGroups()

	keyW := 0
	for _, g := range groups {
		for _, b := range g.binds {
			if w := lipgloss.Width(b.Help().Key); w > keyW {
				keyW = w
			}
		}
	}

	head := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(m.theme.Secondary).Width(keyW)
	descStyle := lipgloss.NewStyle().Foreground(m.theme.Fg)
	dim := lipgloss.NewStyle().Foreground(m.theme.Muted)

	rows := []string{head.Render("Keys") + dim.Render("  (? or esc to close)")}
	for _, g := range groups {
		rows = append(rows, "", head.Render(g.title))
		for _, b := range g.binds {
			rows = append(rows, keyStyle.Render(b.Help().Key)+"  "+descStyle.Render(b.Help().Desc))
		}
	}

	// No background fill: a filled surface leaves inconsistent gaps where the
	// per-span color resets clear it. The border defines the modal instead, so
	// everything sits on one uniform background.
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ActiveBorder).
		Padding(0, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
