package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, root, name, frontmatter string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(frontmatter), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return dir
}

func skillMd(name, desc string) string {
	return "---\nname: " + name + "\ndescription: " + desc + "\n---\n\n# " + name + "\n\nBody.\n"
}

const validSkillMd = `---
name: skill-a
description: Does the A thing.
---

# Skill A

Body content.
`

func TestScanGlobalDiscoversValidSkill(t *testing.T) {
	root := t.TempDir()
	dir := writeSkill(t, root, "skill-a", validSkillMd)

	result := ScanGlobal(root, filepath.Join(root, "missing-lock.json"))

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	got := result.Skills[0]
	if got.Name != "skill-a" {
		t.Errorf("name: got %q want %q", got.Name, "skill-a")
	}
	if got.Description != "Does the A thing." {
		t.Errorf("description: got %q want %q", got.Description, "Does the A thing.")
	}
	if got.Path != dir {
		t.Errorf("path: got %q want %q", got.Path, dir)
	}
}

func TestScanGlobalSortsAndIgnoresNested(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "skill-b", skillMd("skill-b", "B thing"))
	dirA := writeSkill(t, root, "skill-a", skillMd("skill-a", "A thing"))
	writeSkill(t, dirA, "sub", skillMd("nested", "should be ignored"))

	result := ScanGlobal(root, filepath.Join(root, "missing.json"))

	if len(result.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "skill-a" || result.Skills[1].Name != "skill-b" {
		t.Errorf("unexpected order: %q, %q", result.Skills[0].Name, result.Skills[1].Name)
	}
}

func TestScanGlobalCollectsBundledFiles(t *testing.T) {
	root := t.TempDir()
	dir := writeSkill(t, root, "skill-a", skillMd("skill-a", "A thing"))

	mkfile(t, filepath.Join(dir, "references", "guide.md"), "# Guide")
	mkfile(t, filepath.Join(dir, "references", "nested", "extra.md"), "# Extra")
	mkfile(t, filepath.Join(dir, "scripts", "run.sh"), "echo hi")
	mkfile(t, filepath.Join(dir, "assets", "logo.png"), "png")

	result := ScanGlobal(root, filepath.Join(root, "missing.json"))
	got := result.Skills[0]

	if len(got.References) != 2 {
		t.Fatalf("references: got %d want 2", len(got.References))
	}
	if got.References[0].Name != "guide.md" || got.References[1].Name != filepath.Join("nested", "extra.md") {
		t.Errorf("references not sorted by rel path: %+v", got.References)
	}
	if len(got.Scripts) != 1 || got.Scripts[0].Name != "run.sh" {
		t.Errorf("scripts: %+v", got.Scripts)
	}
	if len(got.Assets) != 1 || got.Assets[0].Name != "logo.png" {
		t.Errorf("assets: %+v", got.Assets)
	}
}

func TestScanGlobalMergesLockMetadata(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "in-lockfile", skillMd("in-lockfile", "Has a source"))
	writeSkill(t, root, "local-only", skillMd("local-only", "No source"))

	lockPath := filepath.Join(root, ".skill-lock.json")
	mkfile(t, lockPath, `{
  "version": 3,
  "skills": {
    "in-lockfile": {
      "source": "owner/repo",
      "sourceType": "github",
      "sourceUrl": "https://github.com/owner/repo.git",
      "skillPath": "skills/in-lockfile/SKILL.md"
    }
  }
}`)

	result := ScanGlobal(root, lockPath)

	bySkill := map[string]Skill{}
	for _, s := range result.Skills {
		bySkill[s.Name] = s
	}
	inLock := bySkill["in-lockfile"]
	if inLock.Lock == nil {
		t.Fatal("skill in the lockfile should have lock metadata")
	}
	if inLock.Lock.Source != "owner/repo" {
		t.Errorf("source: got %q", inLock.Lock.Source)
	}
	if bySkill["local-only"].Lock != nil {
		t.Error("skill absent from the lockfile should have nil lock")
	}
}

func TestScanGlobalMalformedSkillStillListed(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "broken", "no frontmatter here\njust text\n")

	result := ScanGlobal(root, filepath.Join(root, "missing.json"))

	if len(result.Skills) != 1 {
		t.Fatalf("expected malformed skill still listed, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "broken" {
		t.Errorf("expected malformed skill to keep its directory name, got %q", result.Skills[0].Name)
	}
}

func TestScanGlobalAbsentLockfileLeavesLockNil(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "skill-a", skillMd("skill-a", "A thing"))

	result := ScanGlobal(root, filepath.Join(root, "does-not-exist.json"))

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Lock != nil {
		t.Error("expected nil lock when lockfile absent")
	}
}

const frontmatterSkillMd = `---
name: fm-skill
description: A skill with extra frontmatter.
license: MIT
allowed-tools: Read, Grep
---

# FM Skill

Body content here.
`

func TestScanGlobalKeepsRawFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "fm-skill", frontmatterSkillMd)

	result := ScanGlobal(root, filepath.Join(root, "missing.json"))

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	fm := result.Skills[0].Frontmatter
	for _, want := range []string{"name: fm-skill", "license: MIT", "allowed-tools: Read, Grep"} {
		if !strings.Contains(fm, want) {
			t.Errorf("frontmatter missing %q; got:\n%s", want, fm)
		}
	}
	if strings.Count(fm, "---") < 2 {
		t.Errorf("expected both --- fence lines; got:\n%s", fm)
	}
}

func TestScanDiscoversSymlinkedSkillDir(t *testing.T) {
	base := t.TempDir()
	store := filepath.Join(base, "store")
	scope := filepath.Join(base, "scope")
	if err := os.MkdirAll(scope, 0o755); err != nil {
		t.Fatalf("mkdir scope: %v", err)
	}
	writeSkill(t, store, "foo", skillMd("foo", "Foo thing"))

	// npx skills installs to the canonical store and symlinks each skill into a
	// harness scope. os.ReadDir reports the symlink via lstat, so entry.IsDir()
	// is false for it; the scanner must follow the link to find SKILL.md.
	if err := os.Symlink(filepath.Join(store, "foo"), filepath.Join(scope, "foo")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	result := Scan(scope, filepath.Join(scope, "missing.json"))

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill via symlink, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "foo" {
		t.Errorf("name: got %q want %q", result.Skills[0].Name, "foo")
	}
}

func TestScanIgnoresNonSkillDirsAndFiles(t *testing.T) {
	scope := t.TempDir()
	writeSkill(t, scope, "bar", skillMd("bar", "Bar thing"))
	if err := os.MkdirAll(filepath.Join(scope, "notaskill"), 0o755); err != nil {
		t.Fatalf("mkdir notaskill: %v", err)
	}
	mkfile(t, filepath.Join(scope, "loose.txt"), "not a skill")

	result := Scan(scope, filepath.Join(scope, "missing.json"))

	if len(result.Skills) != 1 || result.Skills[0].Name != "bar" {
		t.Fatalf("expected only [bar], got %+v", result.Skills)
	}
}

// scopeKey identifies a scan result by section and label for assertions.
func scopeKey(r ScanResult) string {
	return string(r.Scope.Section) + "/" + r.Scope.Name
}

func TestScanAllReturnsNonEmptyScopesTagged(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()

	// Global .agents with one real skill and its lock location.
	writeSkill(t, filepath.Join(home, ".agents", "skills"), "alpha", skillMd("alpha", "A"))
	// Global claude with one symlinked skill.
	claudeSkills := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(claudeSkills, 0o755); err != nil {
		t.Fatalf("mkdir claude: %v", err)
	}
	if err := os.Symlink(filepath.Join(home, ".agents", "skills", "alpha"), filepath.Join(claudeSkills, "alpha")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	// Global codex is present but empty.
	if err := os.MkdirAll(filepath.Join(home, ".codex", "skills"), 0o755); err != nil {
		t.Fatalf("mkdir codex: %v", err)
	}
	// Project .agents with one skill.
	writeSkill(t, filepath.Join(cwd, ".agents", "skills"), "proj", skillMd("proj", "P"))

	results := ScanAll(home, cwd)

	got := map[string]bool{}
	for _, r := range results {
		got[scopeKey(r)] = true
	}
	for _, want := range []string{"Global/.agents", "Global/claude", "Project/.agents"} {
		if !got[want] {
			t.Errorf("expected scope %q in results; got %v", want, got)
		}
	}
	if got["Global/codex"] {
		t.Error("empty codex scope should be omitted")
	}
	if got["Global/opencode"] || got["Global/cursor"] || got["Global/pi"] {
		t.Error("absent scopes should be omitted")
	}
}

// findScope returns the scan result for a section/label, or fails.
func findScope(t *testing.T, results []ScanResult, key string) ScanResult {
	t.Helper()
	for _, r := range results {
		if scopeKey(r) == key {
			return r
		}
	}
	t.Fatalf("scope %q not found in results", key)
	return ScanResult{}
}

func skillByName(t *testing.T, r ScanResult, name string) Skill {
	t.Helper()
	for _, s := range r.Skills {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("skill %q not found in scope %q", name, r.Scope.Name)
	return Skill{}
}

func TestScanAllAgentsScopeReadsLockHarnessDoesNot(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()

	writeSkill(t, filepath.Join(home, ".agents", "skills"), "foo", skillMd("foo", "Foo"))
	mkfile(t, filepath.Join(home, ".agents", ".skill-lock.json"), `{
  "version": 3,
  "skills": {
    "foo": {"source": "owner/repo", "sourceType": "github"}
  }
}`)
	writeSkill(t, filepath.Join(home, ".claude", "skills"), "foo", skillMd("foo", "Foo"))

	results := ScanAll(home, cwd)

	agentsFoo := skillByName(t, findScope(t, results, "Global/.agents"), "foo")
	if agentsFoo.Lock == nil || agentsFoo.Lock.Source != "owner/repo" {
		t.Errorf(".agents foo should be remote with source owner/repo; got %+v", agentsFoo.Lock)
	}
	claudeFoo := skillByName(t, findScope(t, results, "Global/claude"), "foo")
	if claudeFoo.Lock != nil {
		t.Errorf("claude foo should be local (no lock); got %+v", claudeFoo.Lock)
	}
}

func TestScanAllProjectLockAtCwdRoot(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()

	writeSkill(t, filepath.Join(cwd, ".agents", "skills"), "projskill", skillMd("projskill", "P"))
	// The project lock is a v1 schema at the launch-directory root, not inside
	// .agents.
	mkfile(t, filepath.Join(cwd, "skills-lock.json"), `{
  "version": 1,
  "skills": {
    "projskill": {"source": "team/proj-skills", "sourceType": "github", "skillPath": "skills/projskill/SKILL.md"}
  }
}`)

	results := ScanAll(home, cwd)

	got := skillByName(t, findScope(t, results, "Project/.agents"), "projskill")
	if got.Lock == nil || got.Lock.Source != "team/proj-skills" {
		t.Errorf("project skill should read source from cwd skills-lock.json; got %+v", got.Lock)
	}
}

func mkfile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
