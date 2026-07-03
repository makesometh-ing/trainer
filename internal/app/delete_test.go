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
	var m tea.Model = newTestModel(browseResult())

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
		Scope: skills.Scope{Name: ".agents", Section: skills.SectionGlobal, Path: root},
		Skills: []skills.Skill{
			{Name: "bravo", Path: skillDir},
		},
	}

	rescanned := false
	rescan := func() []skills.ScanResult {
		rescanned = true
		return []skills.ScanResult{{Scope: skills.Scope{Name: ".agents", Section: skills.SectionGlobal, Path: root}}}
	}

	var m tea.Model = newTestModel(result, WithRescan(rescan))
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

// A harness-scope skill is a symlink into the canonical .agents store.
// Confirming a delete must unlink the symlink and leave the canonical skill (and
// its files) intact — deleting one agent's mirror, not the shared skill.
func TestConfirmingSymlinkDeleteRemovesLinkNotTarget(t *testing.T) {
	base := t.TempDir()

	canonical := filepath.Join(base, ".agents", "skills", "foo")
	if err := os.MkdirAll(canonical, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonical, "SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	harnessScope := filepath.Join(base, ".claude", "skills")
	if err := os.MkdirAll(harnessScope, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(harnessScope, "foo")
	if err := os.Symlink(canonical, link); err != nil {
		t.Fatal(err)
	}

	// The claude harness scope has no lock, so foo is local and Path is the link.
	result := skills.ScanResult{
		Scope:  skills.Scope{Name: "claude", Section: skills.SectionGlobal, Path: harnessScope},
		Skills: []skills.Skill{{Name: "foo", Path: link}},
	}
	rescan := func() []skills.ScanResult {
		return skills.ScanAll(base, base)
	}

	var m tea.Model = newTestModel(result, WithRescan(rescan))
	m = press(m, ":")
	m = press(m, "d")
	_, cmd := m.Update(tea.KeyPressMsg{Text: "y"})
	if cmd != nil {
		// The filesystem removal happens synchronously inside Update; the returned
		// cmd (if any) carries only the rescan follow-up, so running it is enough.
		cmd()
	}

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Errorf("expected the symlink to be removed, lstat err = %v", err)
	}
	if _, err := os.Stat(canonical); err != nil {
		t.Errorf("expected the canonical skill directory to survive, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(canonical, "SKILL.md")); err != nil {
		t.Errorf("expected the canonical SKILL.md to survive, stat err = %v", err)
	}
}

func TestLockfileDeleteDisabledWithoutNPX(t *testing.T) {
	result := skills.ScanResult{
		Scope: skills.Scope{Name: ".agents", Section: skills.SectionGlobal, Path: "/root"},
		Skills: []skills.Skill{
			{Name: "alpha", Path: "/root/alpha", Lock: &skills.LockEntry{Source: "owner/alpha"}},
		},
	}

	var m tea.Model = newTestModel(result, WithLockedDeleteEnabled(false))
	m = press(m, ":")
	m = press(m, "d")

	m = press(m, "y")

	out := view(m)
	if !strings.Contains(strings.ToLower(out), "disabled") {
		t.Errorf("expected explanatory disabled message, got:\n%s", out)
	}
}
