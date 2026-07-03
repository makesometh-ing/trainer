package app

import (
	huh "charm.land/huh/v2"
)

// gruvboxHuhTheme adapts the app's Gruvbox palette into a huh.Theme so the add
// wizard's form matches the rest of the UI. Huh's default theme colors titles
// indigo and the select selector fuchsia; this overrides the accent-bearing
// styles on huh's neutral base theme with the app's colors.
func gruvboxHuhTheme(t Theme) huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		s := huh.ThemeBase(isDark)

		f := &s.Focused
		f.Base = f.Base.BorderForeground(t.ActiveBorder)
		f.Title = f.Title.Foreground(t.Accent).Bold(true)
		f.NoteTitle = f.NoteTitle.Foreground(t.Accent).Bold(true)
		f.Description = f.Description.Foreground(t.Muted)
		f.ErrorIndicator = f.ErrorIndicator.Foreground(t.Error)
		f.ErrorMessage = f.ErrorMessage.Foreground(t.Error)
		f.SelectSelector = f.SelectSelector.Foreground(t.ActiveBorder)
		f.NextIndicator = f.NextIndicator.Foreground(t.Accent)
		f.PrevIndicator = f.PrevIndicator.Foreground(t.Accent)
		f.Option = f.Option.Foreground(t.Fg)
		f.SelectedOption = f.SelectedOption.Foreground(t.Accent).Bold(true)
		f.MultiSelectSelector = f.MultiSelectSelector.Foreground(t.Accent)
		f.SelectedPrefix = f.SelectedPrefix.Foreground(t.Accent)
		f.TextInput.Cursor = f.TextInput.Cursor.Foreground(t.Accent)
		f.TextInput.Prompt = f.TextInput.Prompt.Foreground(t.Accent)
		f.TextInput.Placeholder = f.TextInput.Placeholder.Foreground(t.Muted)
		f.TextInput.Text = f.TextInput.Text.Foreground(t.Fg)

		// Blurred fields (a group not currently active) inherit the focused styles
		// but dimmed, so only the active field carries the accent.
		s.Blurred = s.Focused
		s.Blurred.Title = s.Blurred.Title.Foreground(t.Muted)
		s.Blurred.Base = s.Blurred.Base.BorderForeground(t.Border)

		return s
	})
}
