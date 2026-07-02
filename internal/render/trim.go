package render

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// TrimSurroundingBlankLines removes leading and trailing blank lines (lines that
// are empty once ANSI styling is stripped). Markdown and syntax-highlight
// output is framed with margin newlines; trimming keeps rendered content flush
// with the section above it and lets a scrollbar reach the bottom when the last
// line of real text is in view.
func TrimSurroundingBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	blank := func(l string) bool { return strings.TrimSpace(ansi.Strip(l)) == "" }
	start := 0
	for start < len(lines) && blank(lines[start]) {
		start++
	}
	end := len(lines)
	for end > start && blank(lines[end-1]) {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}
