package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

func (m Model) View() tea.View {
	scope := m.renderScope()
	list := m.renderSkillList()
	detail := m.renderDetail()

	body := lipgloss.JoinHorizontal(lipgloss.Top, scope, list, detail)
	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

func (m Model) renderScope() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("Scope")
	item := lipgloss.NewStyle().Foreground(m.theme.Fg).Render("Global")
	return m.pane(paneScope, strings.Join([]string{title, item}, "\n"))
}

func (m Model) renderSkillList() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("Skills")
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

	return m.pane(paneSkills, strings.Join(lines, "\n"))
}

func skillMeta(s skills.Skill) string {
	if s.Lock != nil && s.Lock.Source != "" {
		return s.Lock.Source
	}
	return s.Path
}

func (m Model) renderDetail() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("Detail")
	s, ok := m.selectedSkill()
	if !ok {
		return m.pane(paneDetail, title+"\nNo skill selected")
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

	return m.pane(paneDetail, strings.Join(lines, "\n"))
}

func (m Model) pane(p pane, content string) string {
	border := m.theme.Border
	if m.focus == p {
		border = m.theme.ActiveBorder
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
	return style.Render(content)
}
