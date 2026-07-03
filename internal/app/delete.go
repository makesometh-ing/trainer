package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/actions"
	"github.com/makesometh-ing/trainer/internal/skills"
)

type deleteConfirm struct {
	skill skills.Skill
	scope skills.Scope
}

// deleteCmdDisabled reports whether deleting the selected skill is unavailable.
// It mirrors runDelete's gate exactly: only an npx-remove skill is blocked, and
// only while npx is unavailable. An on-disk (symlink/directory) delete is always
// available, so it stays enabled.
func (m Model) deleteCmdDisabled() bool {
	s, ok := m.selectedSkill()
	if !ok {
		return false
	}
	return actions.DeleteStrategy(s) == actions.StrategyNPXRemove && !m.lockedDeleteEnabled
}

func (m Model) startDelete() (tea.Model, tea.Cmd) {
	m.palette = false
	s, ok := m.selectedSkill()
	if !ok {
		return m, nil
	}
	m.confirm = &deleteConfirm{skill: s, scope: m.selectedScopeDef()}
	return m, nil
}

// selectedScopeDef returns the scope the selected skill belongs to. The
// selected skill always belongs to the selected scope (visibleSkills never
// leaves it), so the scope is authoritative for the delete's target.
func (m Model) selectedScopeDef() skills.Scope {
	if m.selectedScope < 0 || m.selectedScope >= len(m.results) {
		return skills.Scope{}
	}
	return m.results[m.selectedScope].Scope
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
	global := m.confirm.scope.Section == skills.SectionGlobal
	m.confirm = nil

	switch actions.DeleteStrategy(skill) {
	case actions.StrategyNPXRemove:
		if !m.lockedDeleteEnabled {
			return m, nil
		}
		if m.deleteRunner == nil {
			return m, nil
		}
		cmd := actions.DeleteCommand(skill.Name, global)
		run := m.deleteRunner(cmd, func(error) tea.Msg { return deleteFinishedMsg{} })
		return m, run
	default:
		// A failed on-disk removal shows no message: the skill stays on disk, so
		// the rescan still lists it and the failure is visible by the skill not
		// disappearing.
		_ = actions.RemoveDirectory(skill.Path)
		m = m.refreshFromDisk()
		return m, nil
	}
}

func (m Model) renderConfirm() string {
	title := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Render("Delete skill")

	scope := m.confirm.scope
	where := scope.Name + " (" + string(scope.Section) + ")"

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		"Delete "+m.confirm.skill.Name+" from the "+where+" scope?",
		"This removes it from that scope and may leave broken symlinks.",
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
