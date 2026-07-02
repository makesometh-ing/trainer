package app

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/render"
	"github.com/makesometh-ing/trainer/internal/skills"
)

func (m Model) View() tea.View {
	if m.tooSmall() {
		v := tea.NewView(m.renderTooSmall())
		v.AltScreen = true
		return v
	}

	scope := m.renderScope()
	list := m.renderSkillList()
	detail := m.renderDetail()

	body := lipgloss.JoinHorizontal(lipgloss.Top, scope, list, detail)
	if m.wizard != nil {
		body = lipgloss.JoinVertical(lipgloss.Left, body, m.renderWizard())
	}
	if m.palette {
		body = m.overlayCenter(body, m.renderPalette())
	}
	if m.confirm != nil {
		body = m.overlayCenter(body, m.renderConfirm())
	}
	if m.status != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, body, m.renderStatus())
	}
	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

func (m Model) tooSmall() bool {
	return m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight)
}

func (m Model) renderTooSmall() string {
	msg := "[Too small] Resize terminal to view the full app"
	styled := lipgloss.NewStyle().Foreground(m.theme.Accent).Render(msg)
	if m.width <= 0 || m.height <= 0 {
		return styled
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, styled)
}

func (m Model) overlayCenter(base, modal string) string {
	baseW := lipgloss.Width(base)
	baseH := lipgloss.Height(base)
	modalW := lipgloss.Width(modal)
	modalH := lipgloss.Height(modal)

	x := max(0, (baseW-modalW)/2)
	y := max(0, (baseH-modalH)/2)

	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(base).Z(0),
		lipgloss.NewLayer(modal).X(x).Y(y).Z(1),
	)
	return comp.Render()
}

func (m Model) renderPalette() string {
	title := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Render("Commands")

	cmd := func(key, label string) string {
		k := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("(" + key + ")")
		l := lipgloss.NewStyle().Foreground(m.theme.Fg).Render(label)
		return k + " " + l
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		cmd("a", "add skill"),
		cmd("d", "delete skill"),
		"",
		cmd("esc", "cancel"),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ActiveBorder).
		Background(m.theme.Elevated).
		Padding(0, 2).
		Render(body)
}

func (m Model) renderStatus() string {
	return lipgloss.NewStyle().
		Foreground(m.theme.Error).
		Render(m.status)
}

func (m Model) renderScope() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("(1) Scope")
	item := lipgloss.NewStyle().Foreground(m.theme.Fg).Render("Global")
	return m.pane(paneScope, m.scopeWidth(), m.paneHeight(), strings.Join([]string{title, item}, "\n"))
}

func (m Model) renderSkillList() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("(2) Skills")

	rowsPerSkill := 2
	avail := m.paneContentHeight() - 1
	if avail < rowsPerSkill {
		avail = rowsPerSkill
	}
	capacity := avail / rowsPerSkill

	start, end := windowBounds(len(m.skills), m.selected, capacity)

	lines := []string{title}
	for i := start; i < end; i++ {
		s := m.skills[i]
		lines = append(lines, m.skillRow(s, i == m.selected)...)
	}

	return m.pane(paneSkills, m.listWidth(), m.paneHeight(), strings.Join(lines, "\n"))
}

// windowBounds returns the [start, end) slice of a list of length n that keeps
// the selected index visible within a window of the given capacity.
func windowBounds(n, selected, capacity int) (int, int) {
	if capacity < 1 {
		capacity = 1
	}
	if n <= capacity {
		return 0, n
	}
	start := selected - capacity/2
	if start < 0 {
		start = 0
	}
	end := start + capacity
	if end > n {
		end = n
		start = end - capacity
	}
	return start, end
}

func (m Model) skillRow(s skills.Skill, selected bool) []string {
	textW := m.listWidth() - paneBorderPad
	if textW < 1 {
		textW = 1
	}
	name := "  " + s.Name
	meta := "  " + skillMeta(s)

	if selected {
		name = "> " + s.Name
		nameStyle := lipgloss.NewStyle().
			Foreground(m.theme.Accent).
			Background(m.theme.Elevated).
			Bold(true).
			Width(textW)
		metaStyle := lipgloss.NewStyle().
			Foreground(m.theme.Fg).
			Background(m.theme.Elevated).
			Width(textW)
		return []string{
			nameStyle.Render(truncate(name, textW)),
			metaStyle.Render(truncate(meta, textW)),
		}
	}

	nameStyle := lipgloss.NewStyle().Foreground(m.theme.Fg)
	metaStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	return []string{
		nameStyle.Render(truncate(name, textW)),
		metaStyle.Render(truncate(meta, textW)),
	}
}

func skillMeta(s skills.Skill) string {
	if s.Lock != nil && s.Lock.Source != "" {
		return s.Lock.Source
	}
	return "local"
}

func (m Model) renderDetail() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("(3) Detail")
	s, ok := m.selectedSkill()
	if !ok {
		return m.pane(paneDetail, m.detailPaneWidth(), m.paneHeight(), title+"\nNo skill selected")
	}

	textW := m.detailWidth()
	header := m.detailHeader(title, s, textW)

	fileLines := m.detailFileLines()

	var contentHeader []string
	if m.tab != tabSkill && (m.tab == tabAssets || m.hasContent()) {
		contentHeader = []string{m.sectionLabel("Content", m.subfocus == subfocusContent), ""}
	}

	usedRows := len(header) + len(fileLines) + len(contentHeader)
	contentRows := m.paneContentHeight() - usedRows
	if contentRows < 1 {
		contentRows = 1
	}

	m.content.SetWidth(textW)
	m.content.SetHeight(contentRows)
	m.content.SetContent(m.currentContent())

	if len(contentHeader) > 0 {
		contentHeader[0] = m.contentSectionLabel(m.subfocus == subfocusContent)
	}

	body := append([]string{}, header...)
	body = append(body, fileLines...)
	body = append(body, contentHeader...)
	if m.tab == tabAssets || m.hasContent() {
		body = append(body, m.content.View())
	}

	return m.pane(paneDetail, m.detailPaneWidth(), m.paneHeight(), strings.Join(body, "\n"))
}

// sectionLabel renders a subfocus section heading, marking the active one with
// a leading pointer so the focused section (file list vs content) is visible.
func (m Model) sectionLabel(name string, active bool) string {
	if active {
		return lipgloss.NewStyle().
			Foreground(m.theme.Accent).
			Bold(true).
			Render("▸ " + name)
	}
	return lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Render("  " + name)
}

// contentSectionLabel renders the Content heading with a scroll percentage so
// the user can see how far through scrollable content they are.
func (m Model) contentSectionLabel(active bool) string {
	pct := int(m.content.ScrollPercent() * 100)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	label := fmt.Sprintf("Content (%d%%)", pct)
	return m.sectionLabel(label, active)
}

func (m Model) detailHeader(title string, s skills.Skill, textW int) []string {
	dim := lipgloss.NewStyle().Foreground(m.theme.Muted)
	lines := []string{
		title,
		lipgloss.NewStyle().Foreground(m.theme.Fg).Bold(true).Render(truncate(s.Name, textW)),
	}
	if s.Description != "" {
		lines = append(lines, dim.Render(truncate(s.Description, textW)))
	}
	if s.Lock != nil {
		if s.Lock.Source != "" {
			lines = append(lines, dim.Render(truncate("source: "+s.Lock.Source, textW)))
		}
		if s.Lock.SourceURL != "" {
			lines = append(lines, dim.Render(truncate("sourceUrl: "+s.Lock.SourceURL, textW)))
		}
		if s.Lock.SkillPath != "" {
			lines = append(lines, dim.Render(truncate("skillPath: "+s.Lock.SkillPath, textW)))
		}
	}
	lines = append(lines, dim.Render(truncate("path: "+s.Path, textW)))
	lines = append(lines, "", m.renderTabs(), "")
	return lines
}

func (m Model) detailFileLines() []string {
	if m.tab == tabSkill {
		return nil
	}
	files, _ := m.currentFilesAndRenderer()
	if files == nil && m.tab != tabAssets {
		return nil
	}
	lines := []string{m.sectionLabel("Files", m.subfocus == subfocusList)}
	lines = append(lines, m.renderFileList(files)...)
	lines = append(lines, "")
	return lines
}

func (m Model) hasContent() bool {
	if m.tab == tabSkill {
		return true
	}
	files, _ := m.currentFilesAndRenderer()
	return len(files) > 0
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > w {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func (m Model) renderTabs() string {
	labels := []struct {
		t     tab
		label string
	}{
		{tabSkill, "(i) SKILL.md"},
		{tabReferences, "(r) References"},
		{tabScripts, "(s) Scripts"},
		{tabAssets, "(a) Assets"},
	}
	parts := make([]string, 0, len(labels))
	for _, l := range labels {
		style := lipgloss.NewStyle().Foreground(m.theme.Muted)
		if m.tab == l.t {
			style = lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
		}
		parts = append(parts, style.Render(l.label))
	}
	return strings.Join(parts, " ")
}

func (m Model) currentContent() string {
	if m.tab == tabSkill {
		s, ok := m.selectedSkill()
		if !ok {
			return ""
		}
		out, err := render.Markdown(s.Body, m.detailWidth())
		if err != nil {
			return s.Body
		}
		return out
	}
	files, content := m.currentFilesAndRenderer()
	if content == nil || len(files) == 0 {
		return ""
	}
	sel := m.fileSel
	if sel < 0 || sel >= len(files) {
		sel = 0
	}
	return content(files[sel])
}

func (m Model) currentFiles() []skills.SkillFile {
	files, _ := m.currentFilesAndRenderer()
	return files
}

func (m Model) currentFilesAndRenderer() ([]skills.SkillFile, func(skills.SkillFile) string) {
	s, ok := m.selectedSkill()
	if !ok {
		return nil, nil
	}
	switch m.tab {
	case tabReferences:
		return s.References, m.renderReferenceContent
	case tabScripts:
		return s.Scripts, m.renderScriptContent
	case tabAssets:
		return s.Assets, func(skills.SkillFile) string { return "No preview available" }
	default:
		return nil, nil
	}
}

func (m Model) renderReferenceContent(f skills.SkillFile) string {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return "unable to read " + f.Name
	}
	if strings.HasSuffix(strings.ToLower(f.Name), ".md") {
		out, rerr := render.Markdown(string(data), m.detailWidth())
		if rerr == nil {
			return out
		}
	}
	return string(data)
}

func (m Model) renderScriptContent(f skills.SkillFile) string {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return "unable to read " + f.Name
	}
	out, rerr := render.Code(string(data), f.Name)
	if rerr != nil {
		return string(data)
	}
	return out
}

func (m Model) detailWidth() int {
	if m.width <= 0 {
		return 80
	}
	w := m.detailPaneWidth() - paneBorderPad
	if w < 1 {
		w = 1
	}
	return w
}

func (m Model) renderFileList(files []skills.SkillFile) []string {
	if len(files) == 0 {
		return []string{"No files"}
	}
	lines := make([]string, 0, len(files))
	for i, f := range files {
		prefix := "  "
		if i == m.fileSel {
			prefix = "> "
		}
		lines = append(lines, prefix+f.Name)
	}
	return lines
}

func (m Model) pane(p pane, width, height int, content string) string {
	border := m.theme.Border
	if m.focus == p {
		border = m.theme.ActiveBorder
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
	if width > 0 {
		style = style.Width(width)
	}
	if height > 0 {
		style = style.Height(height)
	}
	return style.Render(content)
}

const (
	scopePaneWidth = 18
	minListWidth   = 16
)

// paneBorderPad is the horizontal overhead a pane adds around its content
// width: a 1-cell rounded border and 1-cell padding on each side.
const paneBorderPad = 4

// paneVerticalChrome is the vertical overhead a pane adds: top and bottom
// border rows (padding is 0 vertically).
const paneVerticalChrome = 2

// frameMargin is the blank margin left around the whole app frame.
const frameMargin = 1

// paneHeight is the outer height of each pane (including its border), sized so
// the whole frame plus its margin fits within the terminal height.
func (m Model) paneHeight() int {
	if m.height <= 0 {
		return defaultContentHeight + paneVerticalChrome
	}
	h := m.height - frameMargin
	if h < paneVerticalChrome+1 {
		h = paneVerticalChrome + 1
	}
	return h
}

// paneContentHeight is the number of content rows available inside a pane.
func (m Model) paneContentHeight() int {
	h := m.paneHeight() - paneVerticalChrome
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) scopeWidth() int {
	return scopePaneWidth
}

func (m Model) listWidth() int {
	if m.width <= 0 {
		return 24
	}
	remaining := m.width - (scopePaneWidth + paneBorderPad)
	w := remaining / 3
	if w < minListWidth {
		w = minListWidth
	}
	return w
}

func (m Model) detailPaneWidth() int {
	if m.width <= 0 {
		return defaultContentWidth
	}
	used := (scopePaneWidth + paneBorderPad) + (m.listWidth() + paneBorderPad)
	w := m.width - used - paneBorderPad
	if w < 20 {
		w = 20
	}
	return w
}
