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

	addEnabled          bool
	lockedDeleteEnabled bool

	palette bool
	status  string

	width  int
	height int
}

type Option func(*Model)

func WithAddEnabled(enabled bool) Option {
	return func(m *Model) {
		m.addEnabled = enabled
	}
}

func WithLockedDeleteEnabled(enabled bool) Option {
	return func(m *Model) {
		m.lockedDeleteEnabled = enabled
	}
}

func NewModel(result skills.ScanResult, opts ...Option) Model {
	m := Model{
		theme:               GruvboxDarkHard(),
		scope:               result.Scope,
		skills:              result.Skills,
		warnings:            result.Warnings,
		focus:               paneSkills,
		addEnabled:          true,
		lockedDeleteEnabled: true,
		content: viewport.New(
			viewport.WithWidth(defaultContentWidth),
			viewport.WithHeight(defaultContentHeight),
		),
	}
	for _, opt := range opts {
		opt(&m)
	}
	m.syncContent()
	return m
}

func (m Model) AddEnabled() bool {
	return m.addEnabled
}

func (m Model) LockedDeleteEnabled() bool {
	return m.lockedDeleteEnabled
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
