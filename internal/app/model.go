package app

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

const (
	defaultContentWidth  = 80
	defaultContentHeight = 20
	detailChromeHeight   = 10
)

type Model struct {
	theme    Theme
	scope    skills.Scope
	skills   []skills.Skill
	warnings []string

	focus    pane
	selected int

	tab      tab
	fileSel  int
	subfocus subfocus

	content viewport.Model

	width  int
	height int
}

func NewModel(result skills.ScanResult) Model {
	m := Model{
		theme:    GruvboxDarkHard(),
		scope:    result.Scope,
		skills:   result.Skills,
		warnings: result.Warnings,
		focus:    paneSkills,
		content: viewport.New(
			viewport.WithWidth(defaultContentWidth),
			viewport.WithHeight(defaultContentHeight),
		),
	}
	m.syncContent()
	return m
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

func (m *Model) syncSize() {
	m.content.SetWidth(m.detailWidth())
	m.content.SetHeight(m.contentHeight())
	m.content.SetContent(m.currentContent())
}

func (m *Model) syncContent() {
	m.content.SetContent(m.currentContent())
	m.content.GotoTop()
}
