package app

import (
	"os/exec"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
	"github.com/makesometh-ing/trainer/internal/ssh"
)

const (
	defaultContentWidth  = 80
	defaultContentHeight = 20

	minWidth  = 60
	minHeight = 15
)

type Model struct {
	theme    Theme
	scope    skills.Scope
	skills   []skills.Skill
	warnings []string

	search       textinput.Model
	filter       originFilter
	filterCursor originFilter
	skillsMode   skillsMode

	focus    pane
	selected int

	tab      tab
	fileSel  int
	subfocus subfocus

	content viewport.Model

	addEnabled          bool
	lockedDeleteEnabled bool

	sshKeys []ssh.KeyPair

	addRunner    AddRunner
	deleteRunner AddRunner
	rescan       RescanFunc

	palette bool
	help    bool
	status  string

	wizard  *addWizard
	confirm *deleteConfirm

	width  int
	height int
}

type Option func(*Model)

// AddRunner runs the add command, invoking done with the command's exit error
// once it completes. It returns a tea.Cmd so execution integrates with the
// Bubble Tea runtime (suspending the TUI for interactive npx in production).
type AddRunner func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd

// RescanFunc reloads skills from disk after an add or delete action.
type RescanFunc func() skills.ScanResult

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

func WithSSHKeys(keys []ssh.KeyPair) Option {
	return func(m *Model) {
		m.sshKeys = keys
	}
}

func WithAddRunner(runner AddRunner) Option {
	return func(m *Model) {
		m.addRunner = runner
	}
}

func WithDeleteRunner(runner AddRunner) Option {
	return func(m *Model) {
		m.deleteRunner = runner
	}
}

func WithRescan(rescan RescanFunc) Option {
	return func(m *Model) {
		m.rescan = rescan
	}
}

func NewModel(result skills.ScanResult, opts ...Option) Model {
	m := Model{
		theme:               GruvboxDarkHard(),
		scope:               result.Scope,
		skills:              result.Skills,
		warnings:            result.Warnings,
		search:              newSearchInput(),
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
	vis := m.visibleSkills()
	if m.selected < 0 || m.selected >= len(vis) {
		return skills.Skill{}, false
	}
	return vis[m.selected], true
}

func (m *Model) syncSize() {
	m.content.SetWidth(m.contentWidth())
	m.content.SetHeight(m.paneContentHeight())
	m.content.SetContent(m.currentContent())
}

func (m *Model) syncContent() {
	m.content.SetContent(m.currentContent())
	m.content.GotoTop()
}
