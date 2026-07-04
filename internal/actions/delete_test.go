package actions

import (
	"slices"
	"testing"

	"github.com/makesometh-ing/trainer/internal/skills"
)

func TestDeleteStrategyLockfileSkill(t *testing.T) {
	skill := skills.Skill{
		Name: "alpha",
		Path: "/root/alpha",
		Lock: &skills.LockEntry{Source: "owner/alpha"},
	}

	if got := DeleteStrategy(skill); got != StrategyNPXRemove {
		t.Errorf("strategy = %v, want StrategyNPXRemove", got)
	}
}

func TestDeleteStrategyOnDiskSkill(t *testing.T) {
	skill := skills.Skill{
		Name: "bravo",
		Path: "/root/bravo",
	}

	if got := DeleteStrategy(skill); got != StrategyFilesystem {
		t.Errorf("strategy = %v, want StrategyFilesystem", got)
	}
}

func TestDeleteCommandGlobalArgv(t *testing.T) {
	cmd := DeleteCommand("alpha", true)

	wantArgs := []string{"npx", "skills@latest", "remove", "alpha", "--global"}
	if !slices.Equal(cmd.Args, wantArgs) {
		t.Errorf("args = %v, want %v", cmd.Args, wantArgs)
	}
}

func TestDeleteCommandProjectArgv(t *testing.T) {
	cmd := DeleteCommand("alpha", false)

	wantArgs := []string{"npx", "skills@latest", "remove", "alpha"}
	if !slices.Equal(cmd.Args, wantArgs) {
		t.Errorf("args = %v, want %v", cmd.Args, wantArgs)
	}
}
