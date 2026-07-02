package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestPaletteOpensAsModalWithCommands(t *testing.T) {
	var m tea.Model = NewModel(browseResult())

	m = press(m, ":")

	out := view(m)
	if !strings.Contains(out, "Commands") {
		t.Errorf("expected palette modal title 'Commands', got:\n%s", out)
	}
	if !strings.Contains(out, "add skill") {
		t.Errorf("expected palette to list add command, got:\n%s", out)
	}
	if !strings.Contains(out, "delete skill") {
		t.Errorf("expected palette to list delete command, got:\n%s", out)
	}
}

func TestPaletteQuitsWithQ(t *testing.T) {
	var m tea.Model = NewModel(browseResult())

	m = press(m, ":")
	_, cmd := m.Update(tea.KeyPressMsg{Text: "q"})
	if cmd == nil {
		t.Fatal("expected q to return a command while palette is open")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected q to quit from the palette")
	}
}

func TestPaletteClosesOnEsc(t *testing.T) {
	var m tea.Model = NewModel(browseResult())

	m = press(m, ":")
	m = press(m, "esc")

	if strings.Contains(view(m), "Commands") {
		t.Errorf("expected palette to close on esc, got:\n%s", view(m))
	}
}
