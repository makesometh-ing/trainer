package app

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
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

// renderHelp draws the help modal listing every key binding, grouped by the
// context it applies in. The key/description rows come from the Bubbles help
// component; the group headings and framing are themed to match the app.
func (m Model) renderHelp() string {
	kb := defaultKeyBindings()
	h := help.New()
	h.SetWidth(56)

	head := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
	section := func(title string, binds []key.Binding) string {
		return lipgloss.JoinVertical(lipgloss.Left,
			head.Render(title),
			h.FullHelpView([][]key.Binding{binds}),
		)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		head.Render("Keys  (? or esc to close)"),
		"",
		section("Global", kb.global),
		"",
		section("Skills pane", kb.skills),
		"",
		section("Details pane", kb.detail),
		"",
		section("Command palette", kb.palette),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ActiveBorder).
		Background(m.theme.Elevated).
		Padding(0, 2).
		Render(content)
}
