package actions

import "os/exec"

// UpdateCommand builds the interactive `npx skills@latest update` command, which
// updates all installed skills to their latest versions.
func UpdateCommand() *exec.Cmd {
	return exec.Command("npx", "skills@latest", "update")
}
