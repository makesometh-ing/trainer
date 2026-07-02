package app

import "charm.land/bubbles/v2/key"

// keyBindings groups every key binding by the context it applies in. The help
// modal renders these, so the documented keys come from one definition.
type keyBindings struct {
	global  []key.Binding
	skills  []key.Binding
	detail  []key.Binding
	palette []key.Binding
}

func binding(keys, help string) key.Binding {
	return key.NewBinding(key.WithKeys(keys), key.WithHelp(keys, help))
}

func defaultKeyBindings() keyBindings {
	return keyBindings{
		global: []key.Binding{
			binding("1/2/3", "focus Scope / Skills / Details"),
			binding("h/l", "move focus left / right"),
			binding(":", "command palette"),
			binding("?", "toggle this help"),
			binding("q", "quit"),
		},
		skills: []key.Binding{
			binding("j/k", "move selection"),
			binding("/", "search"),
			binding("f", "focus filter"),
			binding("h/l", "move filter option (when focused)"),
			binding("space", "apply filter option"),
			binding("c", "clear filter"),
			binding("r", "reset search + filter"),
		},
		detail: []key.Binding{
			binding("i/r/s/a", "SKILL.md / References / Scripts / Assets"),
			binding("tab", "toggle file list / content"),
			binding("j/k", "move file / scroll content"),
			binding("ctrl+d/u", "half-page scroll"),
			binding("gg/G", "top / bottom of content"),
		},
		palette: []key.Binding{
			binding("a", "add skill"),
			binding("d", "delete skill"),
			binding("u", "update all skills"),
		},
	}
}
