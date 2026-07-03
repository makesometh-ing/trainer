package app

import (
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	huh "charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/actions"
	"github.com/makesometh-ing/trainer/internal/ssh"
)

// wizardFormWidth is the inner width the add form renders at inside its modal.
const wizardFormWidth = 46

var errEmptySource = errors.New("enter a skill source")

// addWizard is the add-skill form: a source input and, for SSH git sources with
// a choice of keys, an SSH-key select. It is a huh.Form embedded in the model so
// the form drives entirely through Model.Update.
type addWizard struct {
	form   *huh.Form
	source *string
	key    *string
	keys   []ssh.KeyPair
}

func newAddWizard(keys []ssh.KeyPair, theme Theme) *addWizard {
	source, key := new(string), new(string)

	opts := make([]huh.Option[string], 0, len(keys))
	for _, k := range keys {
		opts = append(opts, huh.NewOption(k.Name, k.PrivatePath))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("source").Title("Skill source").
				Placeholder("owner/repo").
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errEmptySource
					}
					return nil
				}).
				Value(source),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Key("sshkey").Title("SSH key").
				Options(opts...).
				Value(key),
		).WithHideFunc(func() bool {
			return !sshStepApplies(*source, keys)
		}),
	).WithShowHelp(false).WithWidth(wizardFormWidth).WithTheme(gruvboxHuhTheme(theme))

	return &addWizard{form: form, source: source, key: key, keys: keys}
}

// sshStepApplies reports whether the SSH-key select should be shown: the source
// is an SSH git source and there is more than one key to choose between.
func sshStepApplies(source string, keys []ssh.KeyPair) bool {
	return ssh.IsSSHGitSource(source) && len(keys) >= 2
}

// updateWizard forwards a message to the embedded form and reacts to its state.
// The form owns every message while open because Huh delivers group transitions
// and completion as messages produced by its own cmds, not synchronously.
func (m Model) updateWizard(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ctrl+c and esc are handled before the form sees them: Huh binds quit to
	// ctrl+c only (so esc would otherwise do nothing), and the app's convention
	// is that esc cancels a modal.
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.wizard = nil
			return m, nil
		}
	}

	form, cmd := m.wizard.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.wizard.form = f
	}

	switch m.wizard.form.State {
	case huh.StateCompleted:
		return m.finishWizard()
	case huh.StateAborted:
		m.wizard = nil
		return m, nil
	}
	return m, cmd
}

// finishWizard runs the add for the completed form. The SSH key is read only
// when the SSH step applied; a hidden Select still defaults its bound value to
// the first option, so reading it unconditionally would attach a key to a
// non-SSH source.
func (m Model) finishWizard() (tea.Model, tea.Cmd) {
	source := strings.TrimSpace(*m.wizard.source)
	keyPath := ""
	if sshStepApplies(source, m.wizard.keys) {
		keyPath = *m.wizard.key
	}
	return m.runAdd(source, keyPath)
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
	m.clampSelection()
	m.fileSel = 0
	m.subfocus = subfocusList
	m.syncContent()
	return m
}

func (m Model) renderWizard() string {
	title := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Render("Add skill")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		m.wizard.form.View(),
		lipgloss.NewStyle().Foreground(m.theme.Muted).Render("enter confirm  esc cancel"),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ActiveBorder).
		Padding(0, 2).
		Render(body)
}
