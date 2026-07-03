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

// Every scroll key the help modal lists must actually move the content. This
// guards against the help and the handlers listing different keys: g (not gg)
// jumps to the top, G to the bottom, ctrl+d/u half-page, ctrl+f/b full-page.
func TestHelpScrollKeysActuallyScroll(t *testing.T) {
	// From the top, these keys move the content down (the frame changes).
	for _, k := range []string{"ctrl+d", "ctrl+f", "G"} {
		m := longContentModel()
		before := view(m)
		m = press(m, k)
		if view(m) == before {
			t.Errorf("key %q is shown in help but did not scroll from the top", k)
		}
	}

	// From the bottom, these keys move the content up.
	for _, k := range []string{"ctrl+u", "ctrl+b", "g"} {
		m := longContentModel()
		m = press(m, "G") // go to the bottom first
		before := view(m)
		m = press(m, k)
		if view(m) == before {
			t.Errorf("key %q is shown in help but did not scroll from the bottom", k)
		}
	}
}
