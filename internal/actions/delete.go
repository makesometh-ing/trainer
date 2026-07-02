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
	// `npx skills remove -g <name>`.
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

// DeleteCommand builds the `npx skills remove -g <name>` command.
func DeleteCommand(skillName string) *exec.Cmd {
	return exec.Command("npx", "skills", "remove", "-g", skillName)
}

// RemoveDirectory deletes a skill directory from disk.
func RemoveDirectory(path string) error {
	return os.RemoveAll(path)
}
