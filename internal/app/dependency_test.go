package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

func TestAddDisabledTaggedInPalette(t *testing.T) {
	var m tea.Model = newTestModel(browseResult(), WithAddEnabled(false))

	m = press(m, ":")

	out := plain(view(m))
	if !strings.Contains(out, "disabled without npx") {
		t.Errorf("expected the 'disabled without npx' tag for add, got:\n%s", out)
	}
}

func TestAddEnabledByDefault(t *testing.T) {
	m := newTestModel(browseResult())
	if !m.AddEnabled() {
		t.Error("expected add to be enabled by default")
	}
}

func TestCapabilitiesReflectDependencyFlags(t *testing.T) {
	m := newTestModel(skills.ScanResult{}, WithAddEnabled(false), WithLockedDeleteEnabled(false))
	if m.AddEnabled() {
		t.Error("expected add disabled")
	}
	if m.LockedDeleteEnabled() {
		t.Error("expected locked delete disabled")
	}
}
