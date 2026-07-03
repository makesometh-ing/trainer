package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestHelpModalListsBindings(t *testing.T) {
	var m tea.Model = NewModel(browseResult())
	m = resize(m, 120, 40)
	m = press(m, "?")

	out := plain(view(m))
	for _, want := range []string{"search", "filter", "quit", "update all skills"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected help modal to list %q, got:\n%s", want, out)
		}
	}
}

func TestHelpModalKeysAreAccurate(t *testing.T) {
	var m tea.Model = NewModel(browseResult())
	m = resize(m, 120, 40)
	m = press(m, "?")

	out := plain(view(m))
	if strings.Contains(out, "gg/G") {
		t.Errorf("help lists gg/G but the handler jumps to top on a single g, got:\n%s", out)
	}
	for _, want := range []string{"g/G", "ctrl+f/b"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected help to list %q, got:\n%s", want, out)
		}
	}
}

func TestHelpModalClosesOnEsc(t *testing.T) {
	var m tea.Model = NewModel(browseResult())
	m = resize(m, 120, 40)
	m = press(m, "?")
	if !strings.Contains(plain(view(m)), "update all skills") {
		t.Fatalf("expected the help modal to be open")
	}

	m = press(m, "esc")
	if strings.Contains(plain(view(m)), "update all skills") {
		t.Errorf("expected esc to close the help modal")
	}
}
