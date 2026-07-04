package actions

import (
	"slices"
	"testing"
)

func TestAddCommandWithoutKey(t *testing.T) {
	cmd := AddCommand("owner/repo", "")

	wantArgs := []string{"npx", "skills@latest", "add", "owner/repo"}
	if !slices.Equal(cmd.Args, wantArgs) {
		t.Errorf("args = %v, want %v", cmd.Args, wantArgs)
	}
	for _, e := range cmd.Env {
		if len(e) >= len("GIT_SSH_COMMAND=") && e[:len("GIT_SSH_COMMAND=")] == "GIT_SSH_COMMAND=" {
			t.Errorf("did not expect GIT_SSH_COMMAND in env, got %q", e)
		}
	}
}

func TestAddCommandWithKeySetsGitSSHCommand(t *testing.T) {
	cmd := AddCommand("git@github.com:owner/repo.git", "/home/me/.ssh/id_ed25519")

	wantArgs := []string{"npx", "skills@latest", "add", "git@github.com:owner/repo.git"}
	if !slices.Equal(cmd.Args, wantArgs) {
		t.Errorf("args = %v, want %v", cmd.Args, wantArgs)
	}

	want := `GIT_SSH_COMMAND=ssh -i "/home/me/.ssh/id_ed25519"`
	if !slices.Contains(cmd.Env, want) {
		t.Errorf("expected env to contain %q, got %v", want, cmd.Env)
	}
}
