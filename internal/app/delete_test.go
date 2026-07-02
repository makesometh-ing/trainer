package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

func TestDeleteConfirmShowsWarning(t *testing.T) {
	var m tea.Model = NewModel(browseResult())

	m = press(m, ":")
	m = press(m, "d")

	out := view(m)
	if !strings.Contains(out, "Delete skill") {
		t.Errorf("expected delete confirmation title, got:\n%s", out)
	}
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected confirmation to name the selected skill, got:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "symlink") {
		t.Errorf("expected warning about broken symlinks, got:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "global") {
		t.Errorf("expected warning that it affects the global directory, got:\n%s", out)
	}
}

func TestConfirmingOnDiskDeleteRemovesDirectoryAndRescans(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "bravo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := skills.ScanResult{
		Scope: skills.Scope{Name: "Global", Path: root},
		Skills: []skills.Skill{
			{Name: "bravo", Path: skillDir},
		},
	}

	rescanned := false
	rescan := func() skills.ScanResult {
		rescanned = true
		return skills.ScanResult{Scope: skills.Scope{Name: "Global", Path: root}}
	}

	var m tea.Model = NewModel(result, WithRescan(rescan))
	m = press(m, ":")
	m = press(m, "d")

	next, cmd := m.Update(tea.KeyPressMsg{Text: "y"})
	if cmd != nil {
		next, _ = next.Update(cmd())
	}
	m = next

	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Errorf("expected skill directory to be removed, stat err = %v", err)
	}
	if !rescanned {
		t.Error("expected rescan after delete")
	}
	if strings.Contains(view(m), "bravo") {
		t.Errorf("expected refreshed list not to show bravo, got:\n%s", view(m))
	}
}

func TestLockfileDeleteDisabledWithoutNPX(t *testing.T) {
	result := skills.ScanResult{
		Scope: skills.Scope{Name: "Global", Path: "/root"},
		Skills: []skills.Skill{
			{Name: "alpha", Path: "/root/alpha", Lock: &skills.LockEntry{Source: "owner/alpha"}},
		},
	}

	var m tea.Model = NewModel(result, WithLockedDeleteEnabled(false))
	m = press(m, ":")
	m = press(m, "d")

	m = press(m, "y")

	out := view(m)
	if !strings.Contains(strings.ToLower(out), "disabled") {
		t.Errorf("expected explanatory disabled message, got:\n%s", out)
	}
}
