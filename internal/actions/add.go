package actions

import (
	"os"
	"os/exec"
)

// AddCommand builds the interactive `npx skills@latest add <source>` command,
// pinning @latest so the newest skills script always runs. No scope flag is
// passed, so npx prompts for the agents, skills, and Project/Global scope
// itself. When keyPath is non-empty, GIT_SSH_COMMAND is set so git uses that
// SSH key.
func AddCommand(source string, keyPath string) *exec.Cmd {
	cmd := exec.Command("npx", "skills@latest", "add", source)
	if keyPath != "" {
		cmd.Env = append(os.Environ(), `GIT_SSH_COMMAND=ssh -i "`+keyPath+`"`)
	}
	return cmd
}
