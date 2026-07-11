package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestHelpModalListsBindings(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
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
	var m tea.Model = newTestModel(browseResult())
	m = resize(m, 120, 40)
	m = press(m, "?")

	out := plain(view(m))
	if strings.Contains(out, "gg/G") {
		t.Errorf("help lists gg/G but the handler jumps to top on a single g, got:\n%s", out)
	}
	for _, want := range []string{"g/G", "ctrl+d/u"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected help to list %q, got:\n%s", want, out)
		}
	}
	// Full-page scroll was removed (its ctrl+b clashed with the tmux/herdr prefix).
	if strings.Contains(out, "ctrl+f/b") || strings.Contains(out, "full-page") {
		t.Errorf("did not expect full-page scroll in help, got:\n%s", out)
	}
}

// Slice 14, cycle 6: the help modal lists a Skill Search binding group so the
// overlay's keys are discoverable, sourced from the same keymap the overlay
// handlers match.
func TestHelpModalListsSkillSearchGroup(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = resize(m, 120, 40)
	m = press(m, "?")

	out := plain(view(m))
	if !strings.Contains(out, "Skill Search") {
		t.Errorf("expected a Skill Search group heading in help, got:\n%s", out)
	}
	for _, want := range []string{"install skill", "sort by relevance / popularity / name", "back to results"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected help to list Skill Search binding %q, got:\n%s", want, out)
		}
	}
}

func TestHelpModalClosesOnEsc(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
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
