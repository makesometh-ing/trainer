package app

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
	"github.com/makesometh-ing/trainer/internal/ssh"
)

func TestAddWizardOpensSourcePrompt(t *testing.T) {
	var m tea.Model = NewModel(browseResult())

	m = press(m, ":")
	m = press(m, "a")

	out := view(m)
	if !strings.Contains(out, "Add skill") {
		t.Errorf("expected add wizard title, got:\n%s", out)
	}
	if !strings.Contains(out, "source") {
		t.Errorf("expected source prompt, got:\n%s", out)
	}
}

func typeSource(m tea.Model, source string) tea.Model {
	for _, r := range source {
		m = press(m, string(r))
	}
	return m
}

func TestSSHStepShownForSSHSourceWithMultipleKeys(t *testing.T) {
	keys := []ssh.KeyPair{
		{Name: "id_ed25519", PrivatePath: "/ssh/id_ed25519"},
		{Name: "id_rsa", PrivatePath: "/ssh/id_rsa"},
	}
	var m tea.Model = NewModel(browseResult(), WithSSHKeys(keys))

	m = press(m, ":")
	m = press(m, "a")
	m = typeSource(m, "git@github.com:owner/repo.git")
	m = press(m, "enter")

	out := view(m)
	if !strings.Contains(out, "id_ed25519") || !strings.Contains(out, "id_rsa") {
		t.Errorf("expected SSH key choices, got:\n%s", out)
	}
}

func TestSSHStepSkippedForNonSSHSource(t *testing.T) {
	keys := []ssh.KeyPair{
		{Name: "id_ed25519", PrivatePath: "/ssh/id_ed25519"},
		{Name: "id_rsa", PrivatePath: "/ssh/id_rsa"},
	}
	var m tea.Model = NewModel(browseResult(), WithSSHKeys(keys))

	m = press(m, ":")
	m = press(m, "a")
	m = typeSource(m, "owner/repo")
	m = press(m, "enter")

	out := view(m)
	if strings.Contains(out, "id_ed25519") {
		t.Errorf("did not expect SSH key choices for non-SSH source, got:\n%s", out)
	}
}

func TestAddDisabledDoesNotOpenWizard(t *testing.T) {
	var m tea.Model = NewModel(browseResult(), WithAddEnabled(false))

	m = press(m, ":")
	m = press(m, "a")

	out := view(m)
	if strings.Contains(out, "Add skill") {
		t.Errorf("expected no add wizard when add is disabled, got:\n%s", out)
	}
}

func TestConfirmingAddRunsCommandThenRescans(t *testing.T) {
	var ranArgs []string
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		ranArgs = cmd.Args
		return func() tea.Msg { return done(nil) }
	}

	rescanned := false
	rescan := func() skills.ScanResult {
		rescanned = true
		return skills.ScanResult{
			Scope:  skills.Scope{Name: "Global"},
			Skills: []skills.Skill{{Name: "charlie", Path: "/root/charlie"}},
		}
	}

	var m tea.Model = NewModel(browseResult(),
		WithAddRunner(runner),
		WithRescan(rescan),
	)

	m = press(m, ":")
	m = press(m, "a")
	m = typeSource(m, "owner/repo")

	next, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Fatal("expected a command after confirming add")
	}
	msg := cmd()
	m, _ = next.Update(msg)

	wantArgs := []string{"npx", "skills", "add", "owner/repo", "-g"}
	if !slices.Equal(ranArgs, wantArgs) {
		t.Errorf("ran args = %v, want %v", ranArgs, wantArgs)
	}
	if !rescanned {
		t.Error("expected rescan after add command exits")
	}
	if !strings.Contains(view(m), "charlie") {
		t.Errorf("expected refreshed skill list to show charlie, got:\n%s", view(m))
	}
}
