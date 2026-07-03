package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

func browseResult() skills.ScanResult {
	return skills.ScanResult{
		Scope: skills.Scope{Name: ".agents", Section: skills.SectionGlobal, Path: "/root"},
		Skills: []skills.Skill{
			{
				Name:        "alpha",
				Description: "First skill",
				Path:        "/root/alpha",
				Lock:        &skills.LockEntry{Source: "owner/alpha"},
			},
			{
				Name:        "bravo",
				Description: "Second skill",
				Path:        "/root/bravo",
			},
		},
	}
}

func press(m tea.Model, key string) tea.Model {
	next, _ := m.Update(tea.KeyPressMsg{Text: key})
	return next
}

func view(m tea.Model) string {
	return m.View().Content
}

func TestInitialViewShowsScopeAndSkills(t *testing.T) {
	m := newTestModel(browseResult())

	out := view(m)

	if !strings.Contains(out, "Global") {
		t.Error("expected scope pane to show Global")
	}
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "bravo") {
		t.Error("expected skill list to show all skill names")
	}
	if !strings.Contains(out, "owner/alpha") {
		t.Error("expected detail header to show first skill source")
	}
}

func TestSelectionMovesWithJK(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())

	m = press(m, "j")
	out := view(m)
	if !strings.Contains(out, "bravo") {
		t.Fatal("expected bravo selectable")
	}
	if !strings.Contains(out, "/root/bravo") {
		t.Error("after j, detail header should reflect bravo")
	}

	m = press(m, "k")
	out = view(m)
	if !strings.Contains(out, "/root/alpha") {
		t.Error("after k, detail header should reflect alpha again")
	}
}

func TestRowShowsSourceOrLocalLabel(t *testing.T) {
	m := newTestModel(browseResult())

	out := view(m)
	if !strings.Contains(out, "owner/alpha") {
		t.Error("skill with a lockfile source should show that source")
	}
	if !strings.Contains(out, "local") {
		t.Error("skill without a source should show the 'local' label")
	}
	if strings.Contains(out, "/root/bravo") {
		t.Error("skill without a source should not show its filesystem path in the list")
	}
}

func lineContaining(s, substr string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(plain(line), substr) {
			return line
		}
	}
	return ""
}

func TestSelectedRowIsStyledDifferently(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())

	whenSelected := lineContaining(view(m), "alpha")

	m = press(m, "j")
	whenNotSelected := lineContaining(view(m), "alpha")

	if whenSelected == "" || whenNotSelected == "" {
		t.Fatalf("expected alpha row to render in both states")
	}
	if whenSelected == whenNotSelected {
		t.Errorf("expected the alpha row to render differently when selected vs not, got %q both times", whenSelected)
	}
}

func TestSelectedRowHasNoCaret(t *testing.T) {
	m := newTestModel(browseResult())

	if strings.Contains(plain(view(m)), "> alpha") {
		t.Errorf("expected no caret before the selected skill name, got:\n%s", plain(view(m)))
	}
}

func TestMetaBlockOmitsDescription(t *testing.T) {
	m := newTestModel(browseResult())

	if strings.Contains(plain(view(m)), "First skill") {
		t.Errorf("expected the description to be omitted from the Details meta block, got:\n%s", plain(view(m)))
	}
}

func TestQuitReturnsQuitCmd(t *testing.T) {
	m := newTestModel(browseResult())

	_, cmd := m.Update(tea.KeyPressMsg{Text: "q"})
	if cmd == nil {
		t.Fatal("expected q to return a command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected q to return tea.Quit")
	}
}
