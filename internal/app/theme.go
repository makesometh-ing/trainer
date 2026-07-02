package app

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	Bg           color.Color
	Panel        color.Color
	Elevated     color.Color
	Fg           color.Color
	Muted        color.Color
	Accent       color.Color
	Secondary    color.Color
	Error        color.Color
	Success      color.Color
	Border       color.Color
	ActiveBorder color.Color
}

func GruvboxDarkHard() Theme {
	return Theme{
		Bg:           lipgloss.Color("#1d2021"),
		Panel:        lipgloss.Color("#282828"),
		Elevated:     lipgloss.Color("#3c3836"),
		Fg:           lipgloss.Color("#ebdbb2"),
		Muted:        lipgloss.Color("#928374"),
		Accent:       lipgloss.Color("#fabd2f"),
		Secondary:    lipgloss.Color("#83a598"),
		Error:        lipgloss.Color("#fb4934"),
		Success:      lipgloss.Color("#b8bb26"),
		Border:       lipgloss.Color("#504945"),
		ActiveBorder: lipgloss.Color("#fe8019"),
	}
}
