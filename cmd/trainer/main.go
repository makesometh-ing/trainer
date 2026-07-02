package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/app"
	"github.com/makesometh-ing/trainer/internal/runtime"
	"github.com/makesometh-ing/trainer/internal/skills"
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

	model := app.NewModel(
		result,
		app.WithAddEnabled(deps.NPXAvailable),
		app.WithLockedDeleteEnabled(deps.NPXAvailable),
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
