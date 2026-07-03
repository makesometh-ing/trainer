package app

import (
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
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case ":":
		m.palette = true
		m.status = ""
		return m, nil
	case "?":
		m.help = true
		return m, nil
	case "/":
		m.focus = paneSkills
		m.skillsMode = modeSearch
		return m, m.search.Focus()
	case "f":
		m.focus = paneSkills
		m.skillsMode = modeFilter
		m.filterCursor = m.filter
		return m, nil
	case "1":
		m.focus = paneScope
	case "2":
		m.focus = paneSkills
	case "3":
		m.focus = paneDetail
	case "j", "down":
		m.moveDown()
	case "k", "up":
		m.moveUp()
	case "h", "left":
		m.focusLeft()
	case "l", "right", "enter":
		m.focusRight()
	case "r":
		// r is the one context-dependent key: it resets search and filter in the
		// Skills pane, and selects the References tab in the Details pane.
		switch m.focus {
		case paneSkills:
			m.resetSearchFilter()
		case paneDetail:
			m.setTab(tabReferences)
		}
	case "i", "s", "a", "tab", "ctrl+d", "ctrl+u", "ctrl+f", "ctrl+b", "g", "G":
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
	if m.focus == paneDetail {
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
			return
		}
	}
	m.moveSelection(delta)
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
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.palette = false
	case "a":
		m.palette = false
		if !m.addEnabled {
			m.status = "Adding skills is disabled: npx is not available."
			return m, nil
		}
		m.wizard = newAddWizard(m.sshKeys, m.theme)
		return m, m.wizard.form.Init()
	case "d":
		m.palette = false
		return m.startDelete()
	case "u":
		m.palette = false
		return m.runUpdate()
	}
	return m, nil
}
