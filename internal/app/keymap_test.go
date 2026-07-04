package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

// longContentModel is a Details-focused model whose selected skill has a body
// tall enough to overflow the content viewport, so scroll keys have an
// observable effect.
func longContentModel() tea.Model {
	skill := skills.Skill{
		Name:      "big",
		Path:      "/root/big",
		SkillPath: "/root/big/SKILL.md",
		Body:      strings.Repeat("A line of body text for the scroll fixture.\n\n", 80),
	}
	result := skills.ScanResult{
		Scope:  skills.Scope{Name: ".agents", Section: skills.SectionGlobal, Path: "/root"},
		Skills: []skills.Skill{skill},
	}
	var m tea.Model = newTestModel(result)
	m = resize(m, 100, 40)
	m = press(m, "3") // focus Details; SKILL.md tab is the default
	return m
}

// ctrl+d/u scroll the displayed content from any pane, unlike the other keys
// which act only on the focused pane. This is the one deliberate global override.
func TestHalfPageScrollsContentFromAnyPane(t *testing.T) {
	for _, focusKey := range []string{"1", "2"} { // Scope, then Skills
		m := longContentModel() // starts focused on Details
		m = press(m, focusKey)  // move focus away from Details
		before := view(m)
		m = press(m, "ctrl+d")
		if view(m) == before {
			t.Errorf("ctrl+d did not scroll content while pane %q was focused", focusKey)
		}
		up := view(m)
		m = press(m, "ctrl+u")
		if view(m) == up {
			t.Errorf("ctrl+u did not scroll content while pane %q was focused", focusKey)
		}
	}
}

// Every scroll key the help modal lists must actually move the content. This
// guards against the help and the handlers listing different keys: g (not gg)
// jumps to the top, G to the bottom, ctrl+d/u half-page.
func TestHelpScrollKeysActuallyScroll(t *testing.T) {
	// From the top, these keys move the content down (the frame changes).
	for _, k := range []string{"ctrl+d", "G"} {
		m := longContentModel()
		before := view(m)
		m = press(m, k)
		if view(m) == before {
			t.Errorf("key %q is shown in help but did not scroll from the top", k)
		}
	}

	// From the bottom, these keys move the content up.
	for _, k := range []string{"ctrl+u", "g"} {
		m := longContentModel()
		m = press(m, "G") // go to the bottom first
		before := view(m)
		m = press(m, k)
		if view(m) == before {
			t.Errorf("key %q is shown in help but did not scroll from the bottom", k)
		}
	}
}
