package render

import (
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/quick"
)

func Code(content string, filename string) (string, error) {
	lexer := lexers.Match(filename)
	if lexer == nil || lexer == lexers.Fallback {
		return content, nil
	}

	var buf strings.Builder
	if err := quick.Highlight(&buf, content, lexer.Config().Name, "terminal256", "gruvbox"); err != nil {
		return "", err
	}
	return buf.String(), nil
}
