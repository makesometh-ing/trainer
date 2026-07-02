package render

import (
	"charm.land/glamour/v2"
)

func Markdown(content string, width int) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return r.Render(content)
}
