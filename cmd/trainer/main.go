package main

import (
	"context"
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
	cmd := newCommand(launchTUI, os.Stdout, os.Stderr)
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "trainer: %v\n", err)
		os.Exit(1)
	}
}

// launchTUI scans every skill scope and runs the interactive program. It is the
// command's default action when no flag short-circuits.
func launchTUI() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot resolve home directory: %w", err)
	}

	// npx availability gates the add/update/delete commands inside the TUI (they
	// are dimmed when it is missing), so the app always launches.
	deps := runtime.CheckDefault()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot resolve working directory: %w", err)
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
	if _, err := tea.NewProgram(model).Run(); err != nil {
		return err
	}
	return nil
}
