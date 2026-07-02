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

func TestDeleteCommandArgv(t *testing.T) {
	cmd := DeleteCommand("alpha")

	wantArgs := []string{"npx", "skills", "remove", "-g", "alpha"}
	if !slices.Equal(cmd.Args, wantArgs) {
		t.Errorf("args = %v, want %v", cmd.Args, wantArgs)
	}
}
