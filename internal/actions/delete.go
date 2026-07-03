package actions

import (
	"os"
	"os/exec"

	"github.com/makesometh-ing/trainer/internal/skills"
)

// Strategy selects how a skill is removed.
type Strategy int

const (
	// StrategyNPXRemove removes a skill tracked in the lockfile via
	// `npx skills remove <name>`.
	StrategyNPXRemove Strategy = iota
	// StrategyFilesystem removes a skill that exists only on disk by
	// deleting its directory directly.
	StrategyFilesystem
)

// DeleteStrategy chooses the removal strategy for a skill: lockfile-tracked
// skills are removed through npx, skills present only on disk are removed
// directly.
func DeleteStrategy(skill skills.Skill) Strategy {
	if skill.Lock != nil {
		return StrategyNPXRemove
	}
	return StrategyFilesystem
}

// DeleteCommand builds the `npx skills remove <name>` command, adding --global
// when the skill lives in a Global-section scope so the removal targets the
// right scope deterministically.
func DeleteCommand(skillName string, global bool) *exec.Cmd {
	args := []string{"skills", "remove", skillName}
	if global {
		args = append(args, "--global")
	}
	return exec.Command("npx", args...)
}

// RemoveDirectory deletes a skill directory from disk.
func RemoveDirectory(path string) error {
	return os.RemoveAll(path)
}
