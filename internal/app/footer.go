package app

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// footerCtx is the context the footer names and draws keys for. It resolves from
// the model's focus, skills mode, and Details tab/subfocus.
type footerCtx int

const (
	ctxHidden footerCtx = iota
	ctxScope
	ctxSkills
	ctxDetails
	ctxSearch
	ctxFilter
	// Skill Search add-flow zones. The footer stays live across the whole flow
	// (design decision 3), naming the focused zone and its keys.
	ctxAddChooser         // the add-flow entry chooser
	ctxSkillSearchBox     // the search box
	ctxSkillSearchResults // the results list
	ctxSkillSearchDetail  // the Skill Detail
	ctxPostInstall        // the post-install chooser
)

// footerItem is one key + its footer-specific description. The key text is
// sourced from a keymap binding (b.Help().Key) so a key shown is a key handled;
// the description is footer copy, distinct from the help modal's wording.
type footerItem struct {
	key  string
	desc string
}

func item(b key.Binding, desc string) footerItem {
	return footerItem{key: b.Help().Key, desc: desc}
}

// footerContext resolves which context the footer describes. The palette, help,
// confirm, and the manual Huh wizard hide the footer; the Skill Search add flow
// keeps it live, resolving the chooser and the overlay's focused zone. The
// chooser is checked before the overlay so the post-install chooser (which sits
// on top of the still-open overlay) names its own keys.
func (m Model) footerContext() footerCtx {
	if m.palette || m.help || m.confirm != nil || m.wizard != nil {
		return ctxHidden
	}
	if m.chooser != nil {
		if m.chooser.kind == chooserPostInstall {
			return ctxPostInstall
		}
		return ctxAddChooser
	}
	if m.skillSearch != nil {
		switch m.skillSearch.zone {
		case zoneList:
			return ctxSkillSearchResults
		case zoneDetail:
			return ctxSkillSearchDetail
		default:
			return ctxSkillSearchBox
		}
	}
	if m.skillsMode == modeSearch {
		return ctxSearch
	}
	if m.skillsMode == modeFilter {
		return ctxFilter
	}
	switch m.focus {
	case paneScope:
		return ctxScope
	case paneDetail:
		return ctxDetails
	default:
		return ctxSkills
	}
}

// globalTail is the trailing run of keys available from every navigable context:
// move focus, command palette, help, quit. It follows the context-specific keys.
func (m Model) globalTail() []footerItem {
	return []footerItem{
		item(m.keys.moveFocus, "move focus"),
		item(m.keys.palette, "commands"),
		item(m.keys.help, "keys"),
		item(m.keys.quit, "quit"),
	}
}

// footerParts returns the chip label and the ordered key items for the current
// context. The pane digits (1/2/3) and Details tab keys (i/r/s/a) are omitted
// because they are already shown in the pane titles and tab bar.
func (m Model) footerParts() (chip string, items []footerItem) {
	switch m.footerContext() {
	case ctxScope:
		items = []footerItem{item(m.keys.move, "switch scope")}
		items = append(items, m.globalTail()...)
		return "SCOPE", items
	case ctxSkills:
		items = []footerItem{
			item(m.keys.move, "select"),
			item(m.keys.search, "search"),
			item(m.keys.filter, "filter"),
			item(m.keys.reset, "reset"),
		}
		items = append(items, m.globalTail()...)
		return "SKILLS", items
	case ctxDetails:
		if !m.onFileTab() {
			// SKILL.md tab: the content scrolls and there is no file list.
			items = []footerItem{
				item(m.keys.detailMove, "scroll"),
				item(m.keys.halfPage, "half-page"),
				item(m.keys.topBottom, "top/bottom"),
			}
			items = append(items, m.globalTail()...)
			return "DETAILS", items
		}
		if m.subfocus == subfocusList {
			// File tab, file list active: move the selection and focus the content.
			items = []footerItem{
				item(m.keys.detailMove, "select file"),
				item(m.keys.subfocus, "focus content"),
			}
			items = append(items, m.globalTail()...)
			return "DETAILS", items
		}
		// File tab, content active: the content scrolls and tab returns to files.
		items = []footerItem{
			item(m.keys.detailMove, "scroll"),
			item(m.keys.halfPage, "half-page"),
			item(m.keys.topBottom, "top/bottom"),
			item(m.keys.subfocus, "focus files"),
		}
		items = append(items, m.globalTail()...)
		return "DETAILS", items
	case ctxSearch:
		// Search and filter are input modes with their own keys and no global
		// tail. enter/esc are handled as literals by the search handler.
		return "SEARCH", []footerItem{
			{desc: "type to filter"},
			{key: "enter", desc: "apply"},
			{key: "esc", desc: "clear"},
		}
	case ctxFilter:
		return "FILTER", []footerItem{
			item(m.keys.filterMove, "move option"),
			item(m.keys.filterApply, "apply"),
			item(m.keys.filterClear, "clear"),
			{key: "esc", desc: "done"},
		}
	case ctxAddChooser:
		// The chooser is a self-contained modal step: enter and esc are handled as
		// literals by updateChooser, like the search/filter input modes above.
		return "ADD", []footerItem{
			item(m.keys.move, "select"),
			{key: "enter", desc: "choose"},
			{key: "esc", desc: "cancel"},
		}
	case ctxSkillSearchBox:
		// The search box is an input mode: typing edits the query, enter/↓ hands
		// off to the results list, esc backs out to the entry chooser.
		return "SEARCH", []footerItem{
			{desc: "type to search"},
			{key: "enter/↓", desc: "results"},
			{key: "esc", desc: "back"},
		}
	case ctxSkillSearchResults:
		items := []footerItem{
			item(m.keys.move, "select"),
			item(m.keys.mktSort, "sort"),
			item(m.keys.mktToDetail, "detail"),
			item(m.keys.mktInstall, "install"),
			item(m.keys.search, "search"),
			{key: "esc", desc: "back"},
		}
		// The retry key is live only while the search is in its error state, so it
		// is shown only then (mirrors the inline `space to retry` hint).
		if m.skillSearch != nil && m.skillSearch.state == searchError {
			items = append(items, item(m.keys.mktRetry, "retry"))
		}
		return "RESULTS", items
	case ctxSkillSearchDetail:
		items := []footerItem{
			item(m.keys.tabs, "tabs"),
			item(m.keys.subfocus, "files/content"),
			item(m.keys.detailMove, "move"),
			item(m.keys.mktToList, "list"),
			item(m.keys.mktInstall, "install"),
			item(m.keys.search, "search"),
			{key: "esc", desc: "list"},
		}
		// The retry key is live only while the download is in its error state.
		if m.skillSearch != nil && m.skillSearch.dlError {
			items = append(items, item(m.keys.mktRetry, "retry"))
		}
		return "DETAIL", items
	case ctxPostInstall:
		// The post-install chooser mirrors the entry chooser. On success esc means
		// Finish; on a failed install the chooser offers Try again / Back, so esc
		// means Back.
		escDesc := "finish"
		chip := "ADDED"
		if m.chooser != nil && m.chooser.failed {
			escDesc = "back"
			chip = "FAILED"
		}
		return chip, []footerItem{
			item(m.keys.move, "select"),
			{key: "enter", desc: "choose"},
			{key: "esc", desc: escDesc},
		}
	}
	return "", nil
}

// renderFooter builds the context chip and key hints as a single bottom row. It
// is empty while an overlay modal is open, so the reserved row stays blank.
func (m Model) renderFooter() string {
	chip, items := m.footerParts()
	if chip == "" {
		return ""
	}

	chipStyle := lipgloss.NewStyle().Foreground(m.theme.Bg).Background(m.theme.Accent).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(m.theme.Secondary)
	descStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	sep := descStyle.Render(" · ")

	rendered := make([]string, 0, len(items))
	pinnedIdx := -1
	helpKey := m.keys.help.Help().Key
	for i, it := range items {
		if it.key == helpKey {
			pinnedIdx = i
		}
		if it.key == "" {
			rendered = append(rendered, descStyle.Render(it.desc))
			continue
		}
		rendered = append(rendered, keyStyle.Render(it.key)+" "+descStyle.Render(it.desc))
	}

	chipStr := chipStyle.Render(" "+chip+" ") + " "
	full := chipStr + strings.Join(rendered, sep)

	// Full line fits (or width is unknown): render it as is.
	if m.width <= 0 || lipgloss.Width(full) <= m.width || pinnedIdx < 0 {
		return full
	}

	// Too narrow: keep the chip and a left prefix of the context keys, drop the
	// run in the middle to an ellipsis, and pin "? keys" as the final item. The
	// help item is excluded from the droppable prefix so it is never dropped.
	cand := make([]string, 0, len(rendered)-1)
	cand = append(cand, rendered[:pinnedIdx]...)
	cand = append(cand, rendered[pinnedIdx+1:]...)

	ell := descStyle.Render("…")
	suffixW := lipgloss.Width(sep) + lipgloss.Width(ell) + lipgloss.Width(sep) + lipgloss.Width(rendered[pinnedIdx])

	prefix := []string{cand[0]}
	used := lipgloss.Width(chipStr) + lipgloss.Width(cand[0])
	for i := 1; i < len(cand); i++ {
		next := lipgloss.Width(sep) + lipgloss.Width(cand[i])
		if used+next+suffixW > m.width {
			break
		}
		prefix = append(prefix, cand[i])
		used += next
	}

	return chipStr + strings.Join(prefix, sep) + sep + ell + sep + rendered[pinnedIdx]
}
