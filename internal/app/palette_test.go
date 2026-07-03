package app

import (
	"os/exec"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestPaletteOpensAsModalWithCommands(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())

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
	var m tea.Model = newTestModel(browseResult())

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
	var m tea.Model = newTestModel(browseResult(),
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

// errorColorANSI is the SGR sequence Error (#fb4934) renders as. The status line
// is gone, so no view may contain red text.
const errorColorANSI = "251;73;52"

func hasRedText(rawView string) bool {
	return strings.Contains(rawView, errorColorANSI)
}

func TestPaletteUpdateDisabledWithoutNPX(t *testing.T) {
	ran := false
	var m tea.Model = newTestModel(browseResult(),
		WithAddEnabled(false),
		WithAddRunner(func(_ *exec.Cmd, _ func(error) tea.Msg) tea.Cmd {
			ran = true
			return nil
		}),
	)
	m = press(m, ":")
	// The update command is dimmed with a tag while npx is unavailable.
	if !strings.Contains(plain(view(m)), "disabled without npx") {
		t.Errorf("expected the 'disabled without npx' tag in the palette, got:\n%s", plain(view(m)))
	}
	m = press(m, "u")
	if ran {
		t.Error("expected :u not to run when npx is unavailable")
	}
	if hasRedText(view(m)) {
		t.Errorf("expected no red status text after a dimmed command, got:\n%s", view(m))
	}
}

func TestPaletteClosesOnEsc(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())

	m = press(m, ":")
	m = press(m, "esc")

	if strings.Contains(view(m), "Commands") {
		t.Errorf("expected palette to close on esc, got:\n%s", view(m))
	}
}

func TestPaletteIsCenteredWithinFrame(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
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
