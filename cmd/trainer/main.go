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

	deps := runtime.CheckDefault()
	printDependencies(deps)

	if !deps.NPXAvailable {
		if !runtime.ConfirmContinueWithoutNPX(os.Stdin, os.Stdout) {
			os.Exit(0)
		}
	}

	root := skills.DefaultSkillsRoot(home)
	lockPath := skills.DefaultGlobalLockPath(home)
	result := skills.ScanGlobal(root, lockPath)

	sshDir := filepath.Join(home, ".ssh")
	keys, err := ssh.FindKeyPairs(sshDir)
	if err != nil {
		keys = nil
	}

	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		return tea.ExecProcess(cmd, done)
	}
	rescan := func() skills.ScanResult {
		return skills.ScanGlobal(root, lockPath)
	}

	model := app.NewModel(
		result,
		app.WithAddEnabled(deps.NPXAvailable),
		app.WithLockedDeleteEnabled(deps.NPXAvailable),
		app.WithSSHKeys(keys),
		app.WithAddRunner(runner),
		app.WithRescan(rescan),
	)
	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "trainer: %v\n", err)
		os.Exit(1)
	}
}

func printDependencies(deps runtime.DependencyStatus) {
	for _, tool := range []runtime.Tool{deps.Node, deps.NPM, deps.NPX} {
		if tool.Path == "" {
			fmt.Printf("%s not found\n", tool.Name)
			continue
		}
		fmt.Printf("%s %s\n", tool.Name, tool.Version)
	}
}
