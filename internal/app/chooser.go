package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// chooserKind names which chooser the modal is showing: the add-flow entry
// chooser, or the post-install chooser shown after a Skill Search install.
type chooserKind int

const (
	chooserEntry chooserKind = iota
	chooserPostInstall
)

// Entry-chooser option indices, in render order.
const (
	entryManual = iota
	entrySearch
	entryOptionCount
)

// Post-install-chooser option indices, in render order.
const (
	postFindMore = iota
	postFinish
	postOptionCount
)

// addChooser is the add-flow entry step: pick the manual "Enter skill URL or
// repository" path (the existing Huh wizard) or "Search for skills" (Skill
// Search). cursor is the highlighted option.
type addChooser struct {
	kind   chooserKind
	cursor int
	// failed marks a post-install chooser opened after a failed install, so it is
	// titled honestly (Install failed) and offers Try again / Back instead of
	// claiming success.
	failed bool
}

// updateChooser handles a key press while a chooser is open. Navigation and
// enter are shared; esc closes the entry chooser but on the post-install chooser
// means Finish.
func (m Model) updateChooser(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.chooser.kind == chooserPostInstall {
			// On a failed install, esc backs out to the search box without rescanning
			// or claiming success; on success it Finishes (rescan + land on the skill).
			if m.chooser.failed {
				return m.findMoreSkills()
			}
			return m.finishFromSearch()
		}
		m.chooser = nil
		return m, nil
	case "j", "down":
		if m.chooser.cursor < m.chooserOptionCount()-1 {
			m.chooser.cursor++
		}
		return m, nil
	case "k", "up":
		if m.chooser.cursor > 0 {
			m.chooser.cursor--
		}
		return m, nil
	case "enter":
		return m.chooserPick()
	}
	return m, nil
}

// chooserOptionCount is how many options the open chooser lists, bounding cursor
// movement.
func (m Model) chooserOptionCount() int {
	if m.chooser.kind == chooserPostInstall {
		return postOptionCount
	}
	return entryOptionCount
}

// chooserPick acts on the highlighted option. Picking the manual option opens
// the existing add wizard (returning its Init cmd, since Huh delivers its first
// group's focus via a cmd). Picking Search opens Skill Search once a Marketplace
// client is injected; without one it is inert. On the post-install chooser it
// picks Find more skills or Finish.
func (m Model) chooserPick() (tea.Model, tea.Cmd) {
	if m.chooser.kind == chooserPostInstall {
		if m.chooser.failed {
			// Failure options: Try again re-fires the install; Back returns to the
			// search box (no rescan, no false success, no landing on an absent skill).
			switch m.chooser.cursor {
			case postFindMore:
				return m.retryInstall()
			case postFinish:
				return m.findMoreSkills()
			}
			return m, nil
		}
		switch m.chooser.cursor {
		case postFindMore:
			return m.findMoreSkills()
		case postFinish:
			return m.finishFromSearch()
		}
		return m, nil
	}
	switch m.chooser.cursor {
	case entryManual:
		m.chooser = nil
		m.wizard = newAddWizard(m.sshKeys, m.theme)
		return m, m.wizard.form.Init()
	case entrySearch:
		if m.market == nil {
			return m, nil
		}
		// Grow the overlay from the chooser's footprint to near-full-window.
		startW := lipgloss.Width(m.renderChooser())
		startH := lipgloss.Height(m.renderChooser())
		m.chooser = nil
		m.skillSearch = newSearchOverlay(startW, startH, m.searchTargetW(), m.searchTargetH())
		return m, animTick()
	}
	return m, nil
}

func (m Model) renderChooser() string {
	if m.chooser.kind == chooserPostInstall {
		return m.renderPostInstallChooser()
	}

	title := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Render("Add skill")

	option := func(idx int, label string) string {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(m.theme.Fg)
		if m.chooser.cursor == idx {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
		}
		return style.Render(prefix + label)
	}

	// The Search option is dimmed and tagged when no Marketplace client is
	// injected, mirroring the palette's disabled-command pattern.
	searchOption := func() string {
		if m.market != nil {
			return option(entrySearch, "Search for skills")
		}
		prefix := "  "
		if m.chooser.cursor == entrySearch {
			prefix = "> "
		}
		dim := lipgloss.NewStyle().Foreground(m.theme.Muted)
		return dim.Render(prefix+"Search for skills") + "  " + dim.Render("unavailable")
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		option(entryManual, "Enter skill URL or repository"),
		searchOption(),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ActiveBorder).
		Padding(0, 2).
		Render(body)
}

// renderPostInstallChooser draws the post-install step: after a Skill Search
// install, offer to keep searching (state preserved) or finish (rescan + close).
func (m Model) renderPostInstallChooser() string {
	// A failed install is titled honestly and offers Try again / Back; a
	// successful one offers Find more skills / Finish.
	titleText := "Skill installed"
	firstLabel, secondLabel := "Find more skills", "Finish"
	if m.chooser.failed {
		titleText = "Install failed"
		firstLabel, secondLabel = "Try again", "Back"
	}

	title := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Render(titleText)

	option := func(idx int, label string) string {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(m.theme.Fg)
		if m.chooser.cursor == idx {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
		}
		return style.Render(prefix + label)
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		option(postFindMore, firstLabel),
		option(postFinish, secondLabel),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ActiveBorder).
		Padding(0, 2).
		Render(body)
}
