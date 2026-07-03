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

func mkfile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
