package app

import (
	"os/exec"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

func TestPaletteUpdateRunsWhenNPXAvailable(t *testing.T) {
	ran := false
	var m tea.Model = NewModel(browseResult(),
		WithAddEnabled(true),
		WithAddRunner(func(_ *exec.Cmd, _ func(error) tea.Msg) tea.Cmd {
			ran = true
			return nil
		}),
	)
	m = press(m, ":")
	press(m, "u") // the runner records the call via the ran closure
	if !ran {
		t.Error("expected :u to run the update command when npx is available")
	}
}

func TestPaletteUpdateDisabledWithoutNPX(t *testing.T) {
	ran := false
	var m tea.Model = NewModel(browseResult(),
		WithAddEnabled(false),
		WithAddRunner(func(_ *exec.Cmd, _ func(error) tea.Msg) tea.Cmd {
			ran = true
			return nil
		}),
	)
	m = press(m, ":")
	m = press(m, "u")
	if ran {
		t.Error("expected :u not to run when npx is unavailable")
	}
	if !strings.Contains(view(m), "disabled") {
		t.Errorf("expected an explanatory message when update is disabled, got:\n%s", view(m))
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

func TestPaletteIsCenteredWithinFrame(t *testing.T) {
	var m tea.Model = NewModel(browseResult())
	m = resize(m, 120, 40)
	m = press(m, ":")

	out := view(m)
	lines := strings.Split(out, "\n")

	titleRow := -1
	var titleCol int
	for i, line := range lines {
		if idx := strings.Index(plain(line), "Commands"); idx >= 0 {
			titleRow = i
			titleCol = idx
			break
		}
	}
	if titleRow < 0 {
		t.Fatalf("expected palette title in output")
	}

	totalRows := len(lines)
	frameWidth := lipgloss.Width(out)

	// The modal should sit in the middle band of the frame, not flush to an edge.
	if titleRow < totalRows/5 || titleRow > totalRows*4/5 {
		t.Errorf("expected palette vertically centered; title at row %d of %d", titleRow, totalRows)
	}
	if titleCol < frameWidth/5 || titleCol > frameWidth*4/5 {
		t.Errorf("expected palette horizontally centered; title at col %d of %d", titleCol, frameWidth)
	}
}
