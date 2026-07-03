package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

func TestPaneLabelsShowShortcuts(t *testing.T) {
	m := newTestModel(browseResult())

	out := view(m)
	for _, want := range []string{"(1) Scope", "(2) Skills", "(3) Details"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected pane label %q, got:\n%s", want, out)
		}
	}
}

func TestDetailTabLabelsShowShortcuts(t *testing.T) {
	m := newTestModel(browseResult())

	out := view(m)
	for _, want := range []string{"(i) SKILL.md", "(r) References", "(s) Scripts", "(a) Assets"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected tab label %q, got:\n%s", want, out)
		}
	}
}

func resize(m tea.Model, w, h int) tea.Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return next
}

func TestTooSmallTerminalShowsResizeMessage(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())

	m = resize(m, 20, 5)

	out := view(m)
	if !strings.Contains(out, "Too small") {
		t.Errorf("expected too-small message, got:\n%s", out)
	}
	for _, unwanted := range []string{"(1) Scope", "(2) Skills", "(3) Details"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("did not expect pane title %q while too small, got:\n%s", unwanted, out)
		}
	}
}

func TestGrowingBackRestoresLayout(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())

	m = resize(m, 20, 5)
	m = resize(m, 120, 40)

	out := view(m)
	if strings.Contains(out, "Too small") {
		t.Errorf("did not expect too-small message after growing, got:\n%s", out)
	}
	for _, want := range []string{"(1) Scope", "(2) Skills", "(3) Details"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected pane title %q restored, got:\n%s", want, out)
		}
	}
}

func TestPanesReflowToWidth(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())

	m = resize(m, 100, 40)
	narrow := lipgloss.Width(view(m))

	m = resize(m, 160, 40)
	wide := lipgloss.Width(view(m))

	if wide <= narrow {
		t.Errorf("expected wider terminal to produce a wider frame: narrow=%d wide=%d", narrow, wide)
	}
	if wide > 160 {
		t.Errorf("frame width %d exceeds terminal width 160", wide)
	}
}

func TestFrameFillsTerminalWidth(t *testing.T) {
	for _, w := range []int{100, 120, 161} {
		var m tea.Model = newTestModel(browseResult())
		m = resize(m, w, 40)
		mm := m.(Model)
		t.Logf("w=%d scope=%d list=%d detail=%d", w, mm.scopeWidth(), mm.listWidth(), mm.detailPaneWidth())
		if gotW := lipgloss.Width(view(m)); gotW != w {
			t.Errorf("frame width %d != terminal width %d (dead space or overflow)", gotW, w)
		}
	}
}

func manySkills(n int) skills.ScanResult {
	list := make([]skills.Skill, 0, n)
	for i := 0; i < n; i++ {
		list = append(list, skills.Skill{
			Name: fmt.Sprintf("skill-%03d", i),
			Path: fmt.Sprintf("/root/skill-%03d", i),
		})
	}
	return skills.ScanResult{
		Scope:  skills.Scope{Name: ".agents", Section: skills.SectionGlobal, Path: "/root"},
		Skills: list,
	}
}

func TestFrameFitsWithinTerminal(t *testing.T) {
	const w, h = 100, 30
	var m tea.Model = newTestModel(manySkills(80))
	m = resize(m, w, h)

	out := view(m)
	if gotH := lipgloss.Height(out); gotH > h {
		t.Errorf("frame height %d exceeds terminal height %d (overflow)", gotH, h)
	}
	if gotW := lipgloss.Width(out); gotW > w {
		t.Errorf("frame width %d exceeds terminal width %d (overflow)", gotW, w)
	}
}
