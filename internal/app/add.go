package app

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/actions"
	"github.com/makesometh-ing/trainer/internal/ssh"
)

type wizardStep int

const (
	stepSource wizardStep = iota
	stepSSHKey
)

type addWizard struct {
	step   wizardStep
	source textinput.Model

	keys   []ssh.KeyPair
	keySel int
}

func newAddWizard() *addWizard {
	src := textinput.New()
	src.Prompt = "source: "
	src.Placeholder = "owner/repo"
	src.Focus()
	return &addWizard{
		step:   stepSource,
		source: src,
	}
}

func (m Model) handleWizardKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.wizard = nil
		return m, nil
	case "enter":
		return m.advanceWizard()
	}
	if m.wizard.step == stepSSHKey {
		switch msg.String() {
		case "j", "down":
			m.wizard.moveKey(1)
		case "k", "up":
			m.wizard.moveKey(-1)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.wizard.source, cmd = m.wizard.source.Update(msg)
	return m, cmd
}

func (w *addWizard) moveKey(delta int) {
	next := w.keySel + delta
	if next < 0 {
		next = 0
	}
	if next >= len(w.keys) {
		next = len(w.keys) - 1
	}
	w.keySel = next
}

func (m Model) advanceWizard() (tea.Model, tea.Cmd) {
	switch m.wizard.step {
	case stepSource:
		source := m.wizard.source.Value()
		if ssh.IsSSHGitSource(source) && len(m.sshKeys) >= 2 {
			m.wizard.step = stepSSHKey
			m.wizard.keys = m.sshKeys
			return m, nil
		}
		return m.runAdd(source, "")
	case stepSSHKey:
		source := m.wizard.source.Value()
		keyPath := ""
		if m.wizard.keySel >= 0 && m.wizard.keySel < len(m.wizard.keys) {
			keyPath = m.wizard.keys[m.wizard.keySel].PrivatePath
		}
		return m.runAdd(source, keyPath)
	}
	return m, nil
}

// addFinishedMsg signals the add command completed and the model should rescan.
type addFinishedMsg struct{}

func (m Model) runAdd(source, keyPath string) (tea.Model, tea.Cmd) {
	m.wizard = nil
	if m.addRunner == nil {
		return m, nil
	}
	cmd := actions.AddCommand(source, keyPath)
	run := m.addRunner(cmd, func(error) tea.Msg { return addFinishedMsg{} })
	return m, run
}

func (m Model) refreshFromDisk() Model {
	if m.rescan == nil {
		return m
	}
	result := m.rescan()
	m.skills = result.Skills
	m.scope = result.Scope
	m.warnings = result.Warnings
	if m.selected >= len(m.skills) {
		m.selected = 0
	}
	m.syncContent()
	return m
}

func (m Model) renderWizard() string {
	title := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Render("Add skill")

	if m.wizard.step == stepSSHKey {
		lines := []string{title, "Select SSH key:"}
		for i, k := range m.wizard.keys {
			prefix := "  "
			if i == m.wizard.keySel {
				prefix = "> "
			}
			lines = append(lines, prefix+k.Name)
		}
		lines = append(lines,
			lipgloss.NewStyle().Foreground(m.theme.Muted).Render("enter confirm  esc cancel"))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.wizard.source.View(),
		lipgloss.NewStyle().Foreground(m.theme.Muted).Render("enter confirm  esc cancel"),
	)
}
