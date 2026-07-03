package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

// The confirm text names the scope the delete acts on rather than always
// claiming "global": a Project-scope skill must not be described as global.
func TestProjectDeleteConfirmNamesProjectScopeNotGlobal(t *testing.T) {
	result := skills.ScanResult{
		Scope: skills.Scope{Name: ".agents", Section: skills.SectionProject, Path: "/proj"},
		Skills: []skills.Skill{
			{Name: "alpha", Path: "/proj/alpha", Lock: &skills.LockEntry{Source: "owner/alpha"}},
		},
	}

	var m tea.Model = newTestModel(result)
	m = press(m, ":")
	m = press(m, "d")

	out := view(m)
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected confirmation to name the selected skill, got:\n%s", out)
	}
	if !strings.Contains(out, "Project") {
		t.Errorf("expected confirmation to name the Project scope, got:\n%s", out)
	}
	if strings.Contains(strings.ToLower(out), "global") {
		t.Errorf("expected a Project-scope delete not to claim global, got:\n%s", out)
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

// A lock-listed skill in a Global-section scope removes with --global; a
// lock-listed skill in a Project-section scope removes without it. The flag is
// derived from the selected skill's own scope so the removal is deterministic.
func lockSkillResult(section skills.Section) skills.ScanResult {
	return skills.ScanResult{
		Scope: skills.Scope{Name: ".agents", Section: section, Path: "/root"},
		Skills: []skills.Skill{
			{Name: "alpha", Path: "/root/alpha", Lock: &skills.LockEntry{Source: "owner/alpha"}},
		},
	}
}

func runLockDelete(t *testing.T, section skills.Section) ([]string, bool) {
	t.Helper()
	var ranArgs []string
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		ranArgs = cmd.Args
		return func() tea.Msg { return done(nil) }
	}
	rescanned := false
	rescan := func() []skills.ScanResult {
		rescanned = true
		return []skills.ScanResult{lockSkillResult(section)}
	}

	var m tea.Model = newTestModel(lockSkillResult(section), WithDeleteRunner(runner), WithRescan(rescan))
	m = press(m, ":")
	m = press(m, "d")
	next, cmd := m.Update(tea.KeyPressMsg{Text: "y"})
	if cmd != nil {
		next, _ = next.Update(cmd())
	}
	_ = next
	return ranArgs, rescanned
}

func TestGlobalLockDeleteAddsGlobalFlagAndRescans(t *testing.T) {
	ranArgs, rescanned := runLockDelete(t, skills.SectionGlobal)

	wantArgs := []string{"npx", "skills", "remove", "alpha", "--global"}
	if !slices.Equal(ranArgs, wantArgs) {
		t.Errorf("ran args = %v, want %v", ranArgs, wantArgs)
	}
	if !rescanned {
		t.Error("expected rescan after delete")
	}
}

func TestProjectLockDeleteOmitsGlobalFlagAndRescans(t *testing.T) {
	ranArgs, rescanned := runLockDelete(t, skills.SectionProject)

	wantArgs := []string{"npx", "skills", "remove", "alpha"}
	if !slices.Equal(ranArgs, wantArgs) {
		t.Errorf("ran args = %v, want %v", ranArgs, wantArgs)
	}
	if !rescanned {
		t.Error("expected rescan after delete")
	}
}

// A genuine on-disk delete failure (os.RemoveAll errors) shows no message: the
// skill stays on disk, so a rescan still lists it and the failure is visible by
// the skill not disappearing.
func TestOnDiskDeleteFailureLeavesSkillWithNoMessage(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "bravo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A read-only skill directory makes removing its SKILL.md fail, so the whole
	// os.RemoveAll errors and the skill survives intact.
	if err := os.Chmod(skillDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(skillDir, 0o755) })

	lockPath := filepath.Join(root, "no.lock")
	rescan := func() []skills.ScanResult {
		return []skills.ScanResult{skills.Scan(root, lockPath)}
	}
	result := skills.ScanResult{
		Scope:  skills.Scope{Name: "Global", Section: skills.SectionGlobal, Path: root},
		Skills: []skills.Skill{{Name: "bravo", Path: skillDir, SkillPath: filepath.Join(skillDir, "SKILL.md")}},
	}

	var m tea.Model = newTestModel(result, WithRescan(rescan))
	m = press(m, ":")
	m = press(m, "d")
	next, cmd := m.Update(tea.KeyPressMsg{Text: "y"})
	if cmd != nil {
		next, _ = next.Update(cmd())
	}
	m = next

	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Fatalf("expected the failed delete to leave SKILL.md, stat err = %v", err)
	}
	if !strings.Contains(view(m), "bravo") {
		t.Errorf("expected the skill to remain listed after a failed delete, got:\n%s", view(m))
	}
	if hasRedText(view(m)) {
		t.Errorf("expected no error message on a failed on-disk delete, got:\n%s", view(m))
	}
}

// With npx unavailable, delete is dimmed and inert only when the selected skill
// would delete via npx (a lock-tracked skill). An on-disk skill deletes without
// npx, so its delete stays enabled and starts the confirm.
func TestDeleteDimmedOnlyForNpxOnlySelection(t *testing.T) {
	// browseResult: alpha is lock-tracked (npx delete), bravo is on-disk.
	var m tea.Model = newTestModel(browseResult(), WithLockedDeleteEnabled(false))

	// alpha selected: delete dimmed with the tag, and d is inert.
	m = press(m, ":")
	if !strings.Contains(plain(view(m)), "disabled without npx") {
		t.Errorf("expected delete dimmed for a lock-tracked skill, got:\n%s", plain(view(m)))
	}
	m = press(m, "d")
	if strings.Contains(view(m), "Delete skill") {
		t.Errorf("expected d to be inert for a lock-tracked skill without npx, got:\n%s", view(m))
	}

	// bravo (on-disk) selected: delete enabled, d starts the confirm.
	m = press(m, "esc")
	m = press(m, "j")
	m = press(m, ":")
	if strings.Contains(plain(view(m)), "disabled without npx") {
		t.Errorf("expected an on-disk delete to stay enabled, got:\n%s", plain(view(m)))
	}
	m = press(m, "d")
	if !strings.Contains(view(m), "Delete skill") {
		t.Errorf("expected d to start the confirm for an on-disk skill, got:\n%s", view(m))
	}
}
