package app

import (
	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncSize()
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
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
	case "tab":
		m.toggleSubfocus()
	case "a":
		m.setTab(tabSkill)
	case "b":
		m.setTab(tabReferences)
	case "c":
		m.setTab(tabScripts)
	case "d":
		m.setTab(tabAssets)
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
	return m, nil
}

func (m *Model) moveSelection(delta int) {
	if len(m.skills) == 0 {
		return
	}
	next := m.selected + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.skills) {
		next = len(m.skills) - 1
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

func (m *Model) setTab(t tab) {
	m.tab = t
	m.fileSel = 0
	m.subfocus = subfocusList
	m.syncContent()
}
