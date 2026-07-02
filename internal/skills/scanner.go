package skills

import (
	"fmt"
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
		result.Warnings = append(result.Warnings, fmt.Sprintf("lockfile: %v", err))
		locks = map[string]LockEntry{}
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("scan: %v", err))
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

	content, err := os.ReadFile(skillPath)
	if err != nil {
		skill.Warnings = append(skill.Warnings, fmt.Sprintf("read SKILL.md: %v", err))
	} else {
		fm, _, perr := ParseSkillMarkdown(content)
		if perr != nil {
			skill.Warnings = append(skill.Warnings, fmt.Sprintf("frontmatter: %v", perr))
		}
		if fm.Name != "" {
			skill.Name = fm.Name
		}
		skill.Description = fm.Description
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
