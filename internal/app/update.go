package app

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// While the add wizard is open it owns every message: Huh delivers group
	// transitions and completion as non-key messages produced by its own cmds,
	// so routing only key presses to it would strand those transitions.
	if m.wizard != nil {
		// The wizard is a fixed-width modal. A window size is consumed for the
		// app's own layout but not forwarded to the form: Huh would size every
		// group to the tallest group's height, padding the short source step up
		// to the SSH step and making the modal jump after it loads.
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = ws.Width
			m.height = ws.Height
			m.syncSize()
			return m, nil
		}
		return m.updateWizard(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncSize()
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case addFinishedMsg:
		m = m.refreshFromDisk()
		return m, nil
	case deleteFinishedMsg:
		m = m.refreshFromDisk()
		return m, nil
	case updateFinishedMsg:
		m = m.refreshFromDisk()
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.help {
		return m.handleHelpKey(msg)
	}
	if m.confirm != nil {
		return m.handleConfirmKey(msg)
	}
	if m.palette {
		return m.handlePaletteKey(msg)
	}
	if m.skillsMode == modeSearch {
		return m.handleSearchKey(msg)
	}
	if m.skillsMode == modeFilter {
		return m.handleFilterKey(msg)
	}
	// Dispatch matches against the shared keymap so the handled keys are the same
	// definitions the help modal shows. reset (r) is matched before the detail
	// tabs (which also bind r) so r stays context-dependent.
	switch {
	case key.Matches(msg, m.keys.quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.palette):
		m.palette = true
		m.status = ""
		return m, nil
	case key.Matches(msg, m.keys.help):
		m.help = true
		return m, nil
	case key.Matches(msg, m.keys.search):
		// Search is a Skills-pane key: it acts only while that pane is focused,
		// the same way the tab keys act only in the Details pane.
		if m.focus == paneSkills {
			m.skillsMode = modeSearch
			return m, m.search.Focus()
		}
	case key.Matches(msg, m.keys.filter):
		if m.focus == paneSkills {
			m.skillsMode = modeFilter
			m.filterCursor = m.filter
		}
	case key.Matches(msg, m.keys.focusPanes):
		switch msg.String() {
		case "1":
			m.focus = paneScope
		case "2":
			m.focus = paneSkills
		case "3":
			m.focus = paneDetail
		}
	case key.Matches(msg, m.keys.reset):
		// r resets search and filter in the Skills pane, and selects the
		// References tab in the Details pane.
		switch m.focus {
		case paneSkills:
			m.resetSearchFilter()
		case paneDetail:
			m.setTab(tabReferences)
		}
	case key.Matches(msg, m.keys.move):
		if s := msg.String(); s == "j" || s == "down" {
			m.moveDown()
		} else {
			m.moveUp()
		}
	case key.Matches(msg, m.keys.moveFocus):
		if s := msg.String(); s == "h" || s == "left" {
			m.focusLeft()
		} else {
			m.focusRight()
		}
	case key.Matches(msg, m.keys.tabs),
		key.Matches(msg, m.keys.subfocus),
		key.Matches(msg, m.keys.halfPage),
		key.Matches(msg, m.keys.fullPage),
		key.Matches(msg, m.keys.topBottom):
		// Tab, subfocus, and scroll keys act on the Details pane, so they apply
		// only while it is focused.
		if m.focus == paneDetail {
			m.applyDetailKey(msg.String())
		}
	}
	return m, nil
}

// applyDetailKey runs a Details-pane tab, subfocus, or scroll key. The caller
// gates this on the Details pane being focused.
func (m *Model) applyDetailKey(key string) {
	switch key {
	case "i":
		m.setTab(tabSkill)
	case "s":
		m.setTab(tabScripts)
	case "a":
		m.setTab(tabAssets)
	case "tab":
		m.toggleSubfocus()
	case "ctrl+d":
		m.content.HalfPageDown()
	case "ctrl+u":
		m.content.HalfPageUp()
	case "ctrl+f":
		m.content.PageDown()
	case "ctrl+b":
		m.content.PageUp()
	case "g":
		m.content.GotoTop()
	case "G":
		m.content.GotoBottom()
	}
}

func (m *Model) moveSelection(delta int) {
	// Selection walks the visible (searched + filtered) list, not the full list,
	// so it can never land on a skill the list is not showing.
	vis := m.visibleSkills()
	if len(vis) == 0 {
		return
	}
	next := m.selected + delta
	if next < 0 {
		next = 0
	}
	if next >= len(vis) {
		next = len(vis) - 1
	}
	if next != m.selected {
		m.selected = next
		m.syncContent()
	}
}

func (m *Model) moveDown() {
	m.moveContent(1)
}

func (m *Model) moveUp() {
	m.moveContent(-1)
}

func (m *Model) moveContent(delta int) {
	switch m.focus {
	case paneDetail:
		if m.onFileTab() {
			if m.subfocus == subfocusList {
				m.moveFileSelection(delta)
				return
			}
			m.scrollLines(delta)
			return
		}
		if m.tab == tabSkill {
			m.scrollLines(delta)
		}
	case paneSkills:
		m.moveSelection(delta)
	case paneScope:
		m.moveScope(delta)
	}
}

func (m *Model) scrollLines(delta int) {
	if delta > 0 {
		m.content.ScrollDown(delta)
	} else if delta < 0 {
		m.content.ScrollUp(-delta)
	}
}

func (m Model) onFileTab() bool {
	return m.tab == tabReferences || m.tab == tabScripts || m.tab == tabAssets
}

func (m *Model) moveFileSelection(delta int) {
	files := m.currentFiles()
	if len(files) == 0 {
		return
	}
	next := m.fileSel + delta
	if next < 0 {
		next = 0
	}
	if next >= len(files) {
		next = len(files) - 1
	}
	if next != m.fileSel {
		m.fileSel = next
		m.syncContent()
	}
}

func (m *Model) toggleSubfocus() {
	if m.subfocus == subfocusList {
		m.subfocus = subfocusContent
	} else {
		m.subfocus = subfocusList
	}
}

func (m *Model) focusLeft() {
	if m.focus > paneScope {
		m.focus--
	}
}

func (m *Model) focusRight() {
	if m.focus < paneDetail {
		m.focus++
	}
}

func (m *Model) setTab(t tab) {
	m.tab = t
	m.fileSel = 0
	m.subfocus = subfocusList
	m.syncContent()
}

func (m Model) handlePaletteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.quit):
		return m, tea.Quit
	case msg.String() == "esc":
		m.palette = false
	case key.Matches(msg, m.keys.addCmd):
		m.palette = false
		if !m.addEnabled {
			m.status = "Adding skills is disabled: npx is not available."
			return m, nil
		}
		m.wizard = newAddWizard(m.sshKeys, m.theme)
		return m, m.wizard.form.Init()
	case key.Matches(msg, m.keys.deleteCmd):
		m.palette = false
		return m.startDelete()
	case key.Matches(msg, m.keys.updateCmd):
		m.palette = false
		return m.runUpdate()
	}
	return m, nil
}
