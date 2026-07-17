package app

import "charm.land/bubbles/v2/key"

// keymap is the single source of truth for key bindings. The handlers match
// against these bindings with key.Matches and the help modal renders the same
// bindings, so the documented keys and the handled keys are one definition and
// cannot list different keys. Each binding carries its real keys (WithKeys) and
// the label shown in help (WithHelp).
type keymap struct {
	// Global.
	focusPanes key.Binding // 1 / 2 / 3
	moveFocus  key.Binding // h / l (+ arrows, enter)
	palette    key.Binding // :
	help       key.Binding // ?
	quit       key.Binding // q, ctrl+c

	// Skills pane.
	move   key.Binding // j / k
	search key.Binding // /
	filter key.Binding // f
	reset  key.Binding // r

	// Filter, while the filter is focused.
	filterMove  key.Binding // h / l
	filterApply key.Binding // space
	filterClear key.Binding // c

	// Details pane.
	tabs       key.Binding // i / r / s / a
	subfocus   key.Binding // tab
	detailMove key.Binding // j / k (move file / scroll content)
	halfPage   key.Binding // ctrl+d / ctrl+u
	topBottom  key.Binding // g / G

	// Command palette.
	addCmd    key.Binding // a
	deleteCmd key.Binding // d
	updateCmd key.Binding // u

	// Skill Search pane focus.
	mktFocusPanes key.Binding // 1 / 2 / 3

	// Skill Search results list.
	mktSort key.Binding // r / p / n

	// Skill Search detail navigation.
	mktToDetail key.Binding // l
	mktToList   key.Binding // h

	// Skill Search install.
	mktInstall key.Binding // enter

	// Skill Search retry, in an empty/error state.
	mktRetry key.Binding // space
}

func newKeymap() keymap {
	return keymap{
		focusPanes: key.NewBinding(key.WithKeys("1", "2", "3"), key.WithHelp("1/2/3", "focus Scope / Skills / Details")),
		moveFocus:  key.NewBinding(key.WithKeys("h", "l", "left", "right", "enter"), key.WithHelp("h/l", "move focus left / right")),
		palette:    key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command palette")),
		help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle this help")),
		quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),

		move:   key.NewBinding(key.WithKeys("j", "k", "up", "down"), key.WithHelp("j/k", "move selection")),
		search: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		filter: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "focus filter")),
		reset:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reset search + filter")),

		filterMove:  key.NewBinding(key.WithKeys("h", "l", "left", "right"), key.WithHelp("h/l", "move filter option (filter focused)")),
		filterApply: key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "apply filter option (filter focused)")),
		filterClear: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear filter (filter focused)")),

		tabs:       key.NewBinding(key.WithKeys("i", "r", "s", "a"), key.WithHelp("i/r/s/a", "SKILL.md / References / Scripts / Assets")),
		subfocus:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle file list / content")),
		detailMove: key.NewBinding(key.WithKeys("j", "k", "up", "down"), key.WithHelp("j/k", "move file / scroll content")),
		halfPage:   key.NewBinding(key.WithKeys("ctrl+d", "ctrl+u"), key.WithHelp("ctrl+d/u", "half-page scroll (any pane)")),
		topBottom:  key.NewBinding(key.WithKeys("g", "G"), key.WithHelp("g/G", "top / bottom of content")),

		addCmd:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add skill")),
		deleteCmd: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete skill")),
		updateCmd: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update all skills")),

		mktFocusPanes: key.NewBinding(key.WithKeys("1", "2", "3"), key.WithHelp("1/2/3", "panes")),

		mktSort: key.NewBinding(key.WithKeys("r", "p", "n"), key.WithHelp("r/p/n", "sort by relevance / popularity / name")),

		mktToDetail: key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "open detail")),
		mktToList:   key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "back to results")),

		mktInstall: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "install skill")),

		mktRetry: key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "retry")),
	}
}

type helpGroup struct {
	title string
	binds []key.Binding
}

// helpGroups is what the help modal renders, grouped by context. It draws from
// the same bindings the handlers match, so a key shown here is a key handled.
func (k keymap) helpGroups() []helpGroup {
	return []helpGroup{
		{"Global", []key.Binding{k.focusPanes, k.moveFocus, k.palette, k.help, k.quit}},
		{"Skills pane", []key.Binding{k.move, k.search, k.filter, k.reset, k.filterMove, k.filterApply, k.filterClear}},
		{"Details pane", []key.Binding{k.tabs, k.subfocus, k.detailMove, k.halfPage, k.topBottom}},
		{"Command palette", []key.Binding{k.addCmd, k.deleteCmd, k.updateCmd}},
		{"Skill Search", []key.Binding{k.mktFocusPanes, k.mktSort, k.mktToDetail, k.mktToList, k.mktInstall, k.mktRetry}},
	}
}
