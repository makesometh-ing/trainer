package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

type Model struct {
	theme    Theme
	scope    skills.Scope
	skills   []skills.Skill
	warnings []string

	focus    pane
	selected int

	width  int
	height int
}

func NewModel(result skills.ScanResult) Model {
	return Model{
		theme:    GruvboxDarkHard(),
		scope:    result.Scope,
		skills:   result.Skills,
		warnings: result.Warnings,
		focus:    paneSkills,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) selectedSkill() (skills.Skill, bool) {
	if m.selected < 0 || m.selected >= len(m.skills) {
		return skills.Skill{}, false
	}
	return m.skills[m.selected], true
}
