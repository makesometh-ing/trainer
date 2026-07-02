package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/actions"
	"github.com/makesometh-ing/trainer/internal/skills"
)

type deleteConfirm struct {
	skill skills.Skill
}

func (m Model) startDelete() (tea.Model, tea.Cmd) {
	m.palette = false
	s, ok := m.selectedSkill()
	if !ok {
		return m, nil
	}
	m.confirm = &deleteConfirm{skill: s}
	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "y", "Y":
		return m.runDelete()
	default:
		m.confirm = nil
		return m, nil
	}
}

// deleteFinishedMsg signals a delete command completed and the model should
// rescan.
type deleteFinishedMsg struct{}

func (m Model) runDelete() (tea.Model, tea.Cmd) {
	skill := m.confirm.skill
	m.confirm = nil

	switch actions.DeleteStrategy(skill) {
	case actions.StrategyNPXRemove:
		if !m.lockedDeleteEnabled {
			m.status = "Deleting this skill is disabled: npx is not available."
			return m, nil
		}
		if m.deleteRunner == nil {
			return m, nil
		}
		cmd := actions.DeleteCommand(skill.Name)
		run := m.deleteRunner(cmd, func(error) tea.Msg { return deleteFinishedMsg{} })
		return m, run
	default:
		if err := actions.RemoveDirectory(skill.Path); err != nil {
			m.status = "Failed to delete " + skill.Name + ": " + err.Error()
			return m, nil
		}
		m = m.refreshFromDisk()
		return m, nil
	}
}

func (m Model) renderConfirm() string {
	title := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Render("Delete skill")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		"Delete "+m.confirm.skill.Name+"?",
		"This removes it from the global skills directory and may leave broken symlinks.",
		"",
		lipgloss.NewStyle().Foreground(m.theme.Muted).Render("y confirm  any other key cancel"),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ActiveBorder).
		Background(m.theme.Elevated).
		Padding(0, 2).
		Render(body)
}
