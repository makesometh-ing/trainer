package skills

import (
	"os"
	"path/filepath"
	"sort"
)

func ScanGlobal(root string, lockPath string) ScanResult {
	result := ScanResult{
		Scope: Scope{Name: "Global", Path: root},
	}

	locks, err := ReadGlobalLock(lockPath)
	if err != nil {
		locks = map[string]LockEntry{}
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		skillPath := filepath.Join(dir, "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			continue
		}
		result.Skills = append(result.Skills, buildSkill(entry.Name(), dir, skillPath, locks))
	}

	sort.Slice(result.Skills, func(i, j int) bool {
		return result.Skills[i].Name < result.Skills[j].Name
	})

	return result
}

func buildSkill(name, dir, skillPath string, locks map[string]LockEntry) Skill {
	skill := Skill{
		Name:      name,
		Path:      dir,
		SkillPath: skillPath,
	}

	// A malformed or frontmatter-less SKILL.md is still listed: the read/parse
	// errors are ignored and the skill keeps its directory name and an empty body.
	if content, err := os.ReadFile(skillPath); err == nil {
		fm, raw, body, _ := ParseSkillMarkdown(content)
		if fm.Name != "" {
			skill.Name = fm.Name
		}
		skill.Description = fm.Description
		skill.Body = body
		skill.Frontmatter = raw
	}

	skill.References = collectFiles(dir, "references")
	skill.Scripts = collectFiles(dir, "scripts")
	skill.Assets = collectFiles(dir, "assets")

	if lock, ok := locks[name]; ok {
		locked := lock
		skill.Lock = &locked
	}

	return skill
}

func collectFiles(skillDir, sub string) []SkillFile {
	base := filepath.Join(skillDir, sub)
	info, err := os.Stat(base)
	if err != nil || !info.IsDir() {
		return nil
	}

	var files []SkillFile
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(base, path)
		if relErr != nil {
			rel = d.Name()
		}
		files = append(files, SkillFile{Name: rel, Path: path})
		return nil
	})

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files
}
