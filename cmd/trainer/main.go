package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/app"
	"github.com/makesometh-ing/trainer/internal/skills"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "trainer: cannot resolve home directory: %v\n", err)
		os.Exit(1)
	}

	root := skills.DefaultSkillsRoot(home)
	lockPath := skills.DefaultGlobalLockPath(home)
	result := skills.ScanGlobal(root, lockPath)

	model := app.NewModel(result)
	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "trainer: %v\n", err)
		os.Exit(1)
	}
}
