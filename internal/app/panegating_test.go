package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Search and filter are Skills-pane keys, like the tab keys are Details-pane
// keys: they act only while the Skills pane is focused.

func TestSearchKeyScopedToSkillsPane(t *testing.T) {
	// From the Details pane, `/` is inert: typing after it does not narrow.
	var fromDetails tea.Model = NewModel(browseResult())
	fromDetails = press(fromDetails, "3") // focus Details
	for _, k := range []string{"/", "z", "z", "z"} {
		fromDetails = press(fromDetails, k)
	}
	if !strings.Contains(view(fromDetails), "alpha") {
		t.Errorf("expected `/` to be inert outside the Skills pane, got:\n%s", view(fromDetails))
	}

	// From the Skills pane (default focus), `/` opens search and narrows.
	var fromSkills tea.Model = NewModel(browseResult())
	for _, k := range []string{"/", "z", "z", "z"} {
		fromSkills = press(fromSkills, k)
	}
	if strings.Contains(view(fromSkills), "alpha") {
		t.Errorf("expected `/` in the Skills pane to open search and narrow the list, got:\n%s", view(fromSkills))
	}
}

// j/k move the skills selection only from the Skills pane. From the Scope pane
// they are inert (that key belongs to scope navigation once there is more than
// one scope).
func TestSelectionKeysInertInScopePane(t *testing.T) {
	var m tea.Model = NewModel(browseResult()) // alpha selected
	m = press(m, "1")                          // focus Scope
	m = press(m, "j")

	out := plain(view(m))
	if !strings.Contains(out, "/root/alpha") {
		t.Errorf("expected the selection to stay on alpha, got:\n%s", out)
	}
	if strings.Contains(out, "/root/bravo") {
		t.Errorf("j in the Scope pane must not move the skills selection, got:\n%s", out)
	}
}

func TestFilterKeyScopedToSkillsPane(t *testing.T) {
	// From the Details pane, `f` is inert.
	var fromDetails tea.Model = NewModel(browseResult())
	fromDetails = press(fromDetails, "3") // focus Details
	fromDetails = press(fromDetails, "f")
	fromDetails = press(fromDetails, "l")     // would move the filter cursor if filtering
	fromDetails = press(fromDetails, "space") // would apply the filter if filtering
	if !strings.Contains(view(fromDetails), "bravo") {
		t.Errorf("expected `f` to be inert outside the Skills pane (bravo still shown), got:\n%s", view(fromDetails))
	}

	// From the Skills pane, `f` opens the filter; Remote hides the local-only skill.
	var fromSkills tea.Model = NewModel(browseResult())
	fromSkills = press(fromSkills, "f")
	fromSkills = press(fromSkills, "l")     // cursor -> Remote
	fromSkills = press(fromSkills, "space") // apply Remote
	out := view(fromSkills)
	if !strings.Contains(out, "alpha") || strings.Contains(out, "bravo") {
		t.Errorf("expected Remote filter to show only the locked skill, got:\n%s", out)
	}
}
