package render

import (
	"regexp"
	"strings"
	"testing"
)

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func TestMarkdownPreservesHeadingText(t *testing.T) {
	out, err := Markdown("# Getting Started\n\nSome body text.\n", 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stripANSI(out), "Getting Started") {
		t.Errorf("expected rendered markdown to preserve heading text, got %q", out)
	}
}

// The SKILL.md tab renders frontmatter as a fenced YAML block. Its lines must
// sit flush at the left edge, aligned with the divider above them, not pushed
// right by a document + code-block margin.
func TestMarkdownFrontmatterBlockIsLeftAligned(t *testing.T) {
	md := "```yaml\nname: find-skills\ndescription: does things\n```\n"
	out, err := Markdown(md, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasLeftAlignedLine(stripANSI(out), "name: find-skills") {
		t.Errorf("expected the frontmatter line to be flush left, got:\n%q", out)
	}
}

// hasLeftAlignedLine reports whether some line equals content once trailing
// padding (code-block background fill) is removed and has no leading indent.
func hasLeftAlignedLine(s, content string) bool {
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimRight(line, " ") == content {
			return true
		}
	}
	return false
}

func TestGruvboxStyleUsesSpecPalette(t *testing.T) {
	style := GruvboxDarkHard()

	// Independent source of truth: the design spec's Gruvbox Dark Hard palette.
	const (
		accent = "#fabd2f" // yellow: active/accent, used for headings
		fg     = "#ebdbb2" // fg1: document text
	)

	if style.Heading.Color == nil || *style.Heading.Color != accent {
		t.Errorf("expected heading color %s, got %v", accent, ptrStr(style.Heading.Color))
	}
	if style.Document.Color == nil || *style.Document.Color != fg {
		t.Errorf("expected document color %s, got %v", fg, ptrStr(style.Document.Color))
	}
}

func ptrStr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func TestCodeHighlightPreservesSource(t *testing.T) {
	src := "package main\n\nfunc main() {}\n"
	out, err := Code(src, "main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stripANSI(out), "func main") {
		t.Errorf("expected highlighted code to contain source token, got %q", out)
	}
}

func TestTrimSurroundingBlankLines(t *testing.T) {
	in := "\n\n  \nfirst\n\nsecond\n \n\n"
	got := TrimSurroundingBlankLines(in)
	want := "first\n\nsecond"
	if got != want {
		t.Errorf("TrimSurroundingBlankLines = %q, want %q", got, want)
	}
}

func TestCodeUnknownExtensionFallsBackToPlainText(t *testing.T) {
	src := "just some raw text\nwith lines\n"
	out, err := Code(src, "notes.weirdext")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != src {
		t.Errorf("expected unknown extension to return raw source, got %q", out)
	}
}
