package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

func TestSearchBoxDoesNotOverflowFrame(t *testing.T) {
	const w, h = 120, 40
	var m tea.Model = newTestModel(browseResult())
	m = resize(m, w, h)
	m = press(m, "/")
	m = typeString(m, strings.Repeat("x", 200))

	if gotW := lipgloss.Width(view(m)); gotW != w {
		t.Errorf("long search query changed the frame width: got %d want %d", gotW, w)
	}
	if gotH := lipgloss.Height(view(m)); gotH > h {
		t.Errorf("long search query overflowed the frame height: got %d > %d", gotH, h)
	}
}

func typeString(m tea.Model, s string) tea.Model {
	for _, r := range s {
		m = press(m, string(r))
	}
	return m
}

func pressSpace(m tea.Model) tea.Model {
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	return next
}

func mixedResult() skills.ScanResult {
	return skills.ScanResult{
		Scope: skills.Scope{Name: ".agents", Section: skills.SectionGlobal, Path: "/root"},
		Skills: []skills.Skill{
			{Name: "alpha", Path: "/root/alpha", Lock: &skills.LockEntry{Source: "owner/alpha"}},
			{Name: "avocado", Path: "/root/avocado"},
			{Name: "apex", Path: "/root/apex", Lock: &skills.LockEntry{Source: "owner/apex"}},
		},
	}
}

func TestSearchNarrowsList(t *testing.T) {
	var m tea.Model = newTestModel(browseResult()) // alpha (locked), bravo (local)
	m = press(m, "/")
	m = typeString(m, "br")

	out := plain(view(m))
	if !strings.Contains(out, "bravo") {
		t.Errorf("expected 'bravo' to match search 'br', got:\n%s", out)
	}
	if strings.Contains(out, "alpha") {
		t.Errorf("expected 'alpha' filtered out by search 'br', got:\n%s", out)
	}
}

func TestSearchEnterKeepsResults(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "/")
	m = typeString(m, "br")
	m = press(m, "enter")

	out := plain(view(m))
	if !strings.Contains(out, "bravo") || strings.Contains(out, "alpha") {
		t.Errorf("expected the narrowed list to remain after enter, got:\n%s", out)
	}
	// After enter the search box no longer takes typing.
	before := m.(Model).search.Value()
	m = typeString(m, "zzz")
	if m.(Model).search.Value() != before {
		t.Errorf("expected enter to leave the search box; it is still capturing keys")
	}
}

func TestSearchEscClearsAndRestores(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "/")
	m = typeString(m, "br")
	m = press(m, "esc")

	out := plain(view(m))
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "bravo") {
		t.Errorf("expected esc to clear search and restore the full list, got:\n%s", out)
	}
}

func TestFilterRemoteShowsOnlyLocked(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "f") // focus filter (cursor at All)
	m = press(m, "l") // cursor -> Remote
	m = pressSpace(m) // apply Remote

	out := plain(view(m))
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected locked skill 'alpha' under the Remote filter, got:\n%s", out)
	}
	if strings.Contains(out, "bravo") {
		t.Errorf("expected local skill 'bravo' hidden under the Remote filter, got:\n%s", out)
	}
}

func TestFilterLocalAndClear(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "f")
	m = press(m, "l") // Remote
	m = press(m, "l") // Local
	m = pressSpace(m) // apply Local

	out := plain(view(m))
	if !strings.Contains(out, "bravo") {
		t.Errorf("expected local skill 'bravo' under the Local filter, got:\n%s", out)
	}
	if strings.Contains(out, "owner/alpha") {
		t.Errorf("expected locked skill hidden under the Local filter, got:\n%s", out)
	}

	m = press(m, "c") // clear -> All
	out = plain(view(m))
	if !strings.Contains(out, "owner/alpha") || !strings.Contains(out, "bravo") {
		t.Errorf("expected 'c' to clear the filter back to All, got:\n%s", out)
	}
}

func TestSearchAndFilterCombine(t *testing.T) {
	var m tea.Model = newTestModel(mixedResult())
	m = press(m, "f")
	m = press(m, "l") // Remote
	m = pressSpace(m)
	m = press(m, "esc") // leave filter mode
	m = press(m, "/")
	m = typeString(m, "a") // matches all three names

	out := plain(view(m))
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "apex") {
		t.Errorf("expected locked skills matching 'a', got:\n%s", out)
	}
	if strings.Contains(out, "avocado") {
		t.Errorf("expected local 'avocado' excluded by the Remote filter, got:\n%s", out)
	}
}

func TestResetInSkillsPaneClearsSearchAndFilter(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "f")
	m = press(m, "l")
	m = pressSpace(m) // Remote
	m = press(m, "esc")
	m = press(m, "/")
	m = typeString(m, "al")
	m = press(m, "enter")

	m = press(m, "r") // Skills pane focused -> reset

	out := plain(view(m))
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "bravo") {
		t.Errorf("expected 'r' to reset search+filter and restore all skills, got:\n%s", out)
	}
	if got := m.(Model).search.Value(); got != "" {
		t.Errorf("expected 'r' to clear the search text, got %q", got)
	}
	if m.(Model).filter != originAll {
		t.Errorf("expected 'r' to reset the filter to All")
	}
}

func TestRInDetailPaneSelectsReferences(t *testing.T) {
	var m tea.Model = newTestModel(twoReferencesResult(t))
	m = sized(m, 120, 40)
	m = press(m, "3") // focus Details
	m = press(m, "r") // References tab, not reset

	if !strings.Contains(plain(view(m)), "a-guide.md") {
		t.Errorf("expected 'r' in the Details pane to open References, got:\n%s", plain(view(m)))
	}
}

func TestSelectionStaysWithinFilteredList(t *testing.T) {
	// mixedResult has one local skill (avocado); navigating the filtered list
	// must never select a skill outside it.
	var m tea.Model = newTestModel(mixedResult())
	m = press(m, "j") // move down the full list first
	m = press(m, "j")

	m = press(m, "f")
	m = press(m, "l") // cursor -> Remote
	m = press(m, "l") // cursor -> Local
	m = pressSpace(m) // apply Local (1 skill)
	m = press(m, "esc")

	for _, k := range []string{"j", "j", "k", "j"} {
		m = press(m, k)
		if _, ok := m.(Model).selectedSkill(); !ok {
			t.Fatalf("after %q the list is non-empty but nothing is selected", k)
		}
	}
	if !strings.Contains(plain(view(m)), "avocado") {
		t.Errorf("expected the one local skill to stay selected, got:\n%s", plain(view(m)))
	}
}
