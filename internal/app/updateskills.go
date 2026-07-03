package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/actions"
)

// updateFinishedMsg signals the update command completed and the model should rescan.
type updateFinishedMsg struct{}

// runUpdate suspends the TUI to run `npx skills@latest update`, then refreshes
// from disk. Update needs npx, so it is gated on the same capability flag as add.
func (m Model) runUpdate() (tea.Model, tea.Cmd) {
	if !m.addEnabled {
		return m, nil
	}
	if m.addRunner == nil {
		return m, nil
	}
	cmd := actions.UpdateCommand()
	run := m.addRunner(cmd, func(error) tea.Msg { return updateFinishedMsg{} })
	return m, run
}
