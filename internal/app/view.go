package app

import (
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
	return m.pane(paneScope, m.scopeWidth(), strings.Join([]string{title, item}, "\n"))
}

func (m Model) renderSkillList() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("(2) Skills")
	lines := []string{title}

	for i, s := range m.skills {
		name := s.Name
		if i == m.selected {
			name = "> " + name
		} else {
			name = "  " + name
		}
		meta := skillMeta(s)
		lines = append(lines, name, "    "+meta)
	}

	return m.pane(paneSkills, m.listWidth(), strings.Join(lines, "\n"))
}

func skillMeta(s skills.Skill) string {
	if s.Lock != nil && s.Lock.Source != "" {
		return s.Lock.Source
	}
	return s.Path
}

func (m Model) renderDetail() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("(3) Detail")
	s, ok := m.selectedSkill()
	if !ok {
		return m.pane(paneDetail, m.detailPaneWidth(), title+"\nNo skill selected")
	}

	lines := []string{
		title,
		lipgloss.NewStyle().Foreground(m.theme.Fg).Bold(true).Render(s.Name),
	}
	if s.Description != "" {
		lines = append(lines, s.Description)
	}
	if s.Lock != nil {
		if s.Lock.Source != "" {
			lines = append(lines, "source: "+s.Lock.Source)
		}
		if s.Lock.SourceURL != "" {
			lines = append(lines, "sourceUrl: "+s.Lock.SourceURL)
		}
		if s.Lock.SkillPath != "" {
			lines = append(lines, "skillPath: "+s.Lock.SkillPath)
		}
	}
	lines = append(lines, "path: "+s.Path)

	lines = append(lines, m.renderTabs())
	lines = append(lines, m.renderTabBody()...)

	return m.pane(paneDetail, m.detailPaneWidth(), strings.Join(lines, "\n"))
}

func (m Model) renderTabs() string {
	labels := []struct {
		t     tab
		label string
	}{
		{tabSkill, "(a) SKILL.md"},
		{tabReferences, "(b) References"},
		{tabScripts, "(c) Scripts"},
		{tabAssets, "(d) Assets"},
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

func (m Model) renderTabBody() []string {
	if m.tab == tabSkill {
		return []string{m.content.View()}
	}
	files, _ := m.currentFilesAndRenderer()
	if files == nil && m.tab != tabAssets {
		return nil
	}
	lines := m.renderFileList(files)
	if len(files) == 0 {
		return lines
	}
	lines = append(lines, "", m.content.View())
	return lines
}

func (m Model) contentHeight() int {
	if m.height <= 0 {
		return defaultContentHeight
	}
	h := m.height - detailChromeHeight
	if h < 3 {
		return 3
	}
	return h
}

func (m Model) currentContent() string {
	if m.tab == tabSkill {
		s, ok := m.selectedSkill()
		if !ok || s.SkillPath == "" {
			return ""
		}
		return m.renderReferenceContent(skills.SkillFile{Name: "SKILL.md", Path: s.SkillPath})
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
	return m.detailPaneWidth()
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

func (m Model) pane(p pane, width int, content string) string {
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
	return style.Render(content)
}

const (
	scopePaneWidth = 18
	minListWidth   = 16
)

// paneBorderPad is the horizontal overhead a pane adds around its content
// width: a 1-cell rounded border and 1-cell padding on each side.
const paneBorderPad = 4

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
