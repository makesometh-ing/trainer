package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func focusOf(m tea.Model) pane {
	return m.(Model).focus
}

func TestHLMoveFocusBetweenPanes(t *testing.T) {
	var m tea.Model = NewModel(browseResult())
	if focusOf(m) != paneSkills {
		t.Fatalf("expected initial focus on skills pane, got %v", focusOf(m))
	}

	m = press(m, "l")
	if focusOf(m) != paneDetail {
		t.Errorf("expected l to move focus to detail pane, got %v", focusOf(m))
	}

	m = press(m, "l")
	if focusOf(m) != paneDetail {
		t.Errorf("expected l at rightmost pane to stay on detail, got %v", focusOf(m))
	}

	m = press(m, "h")
	if focusOf(m) != paneSkills {
		t.Errorf("expected h to move focus back to skills pane, got %v", focusOf(m))
	}

	m = press(m, "h")
	if focusOf(m) != paneScope {
		t.Errorf("expected h to move focus to scope pane, got %v", focusOf(m))
	}

	m = press(m, "h")
	if focusOf(m) != paneScope {
		t.Errorf("expected h at leftmost pane to stay on scope, got %v", focusOf(m))
	}
}

func TestEnterMovesFocusIntoDetail(t *testing.T) {
	var m tea.Model = NewModel(browseResult())

	m = press(m, "enter")
	if focusOf(m) != paneDetail {
		t.Errorf("expected enter to move focus into the detail pane, got %v", focusOf(m))
	}
}
