package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/app"
	"github.com/makesometh-ing/trainer/internal/runtime"
	"github.com/makesometh-ing/trainer/internal/skills"
	"github.com/makesometh-ing/trainer/internal/ssh"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "trainer: cannot resolve home directory: %v\n", err)
		os.Exit(1)
	}

	// npx availability gates the add/update/delete commands inside the TUI (they
	// are dimmed when it is missing), so the app always launches.
	deps := runtime.CheckDefault()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "trainer: cannot resolve working directory: %v\n", err)
		os.Exit(1)
	}

	results := skills.ScanAll(home, cwd)

	sshDir := filepath.Join(home, ".ssh")
	keys, err := ssh.FindKeyPairs(sshDir)
	if err != nil {
		keys = nil
	}

	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		return tea.ExecProcess(cmd, done)
	}
	rescan := func() []skills.ScanResult {
		return skills.ScanAll(home, cwd)
	}

	model := app.NewModel(
		results,
		app.WithAddEnabled(deps.NPXAvailable),
		app.WithLockedDeleteEnabled(deps.NPXAvailable),
		app.WithSSHKeys(keys),
		app.WithAddRunner(runner),
		app.WithDeleteRunner(runner),
		app.WithRescan(rescan),
	)
	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "trainer: %v\n", err)
		os.Exit(1)
	}
}
