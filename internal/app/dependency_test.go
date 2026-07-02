package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

func TestAddDisabledShowsExplanatoryMessage(t *testing.T) {
	var m tea.Model = NewModel(browseResult(), WithAddEnabled(false))

	m = press(m, ":")
	m = press(m, "a")

	out := view(m)
	if !strings.Contains(out, "Adding skills is disabled") {
		t.Errorf("expected explanatory message when add is disabled, got:\n%s", out)
	}
}

func TestAddEnabledByDefault(t *testing.T) {
	m := NewModel(browseResult())
	if !m.AddEnabled() {
		t.Error("expected add to be enabled by default")
	}
}

func TestCapabilitiesReflectDependencyFlags(t *testing.T) {
	m := NewModel(skills.ScanResult{}, WithAddEnabled(false), WithLockedDeleteEnabled(false))
	if m.AddEnabled() {
		t.Error("expected add disabled")
	}
	if m.LockedDeleteEnabled() {
		t.Error("expected locked delete disabled")
	}
}
