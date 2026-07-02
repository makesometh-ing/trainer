package runtime

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type LookPathFunc func(name string) (string, bool)

type VersionFunc func(name string) string

type Tool struct {
	Name    string
	Path    string
	Version string
}

type DependencyStatus struct {
	Node         Tool
	NPM          Tool
	NPX          Tool
	Missing      []string
	NPXAvailable bool
}

func Check(look LookPathFunc, version VersionFunc) DependencyStatus {
	status := DependencyStatus{}
	status.Node = detect("node", look, version, &status.Missing)
	status.NPM = detect("npm", look, version, &status.Missing)
	status.NPX = detect("npx", look, version, &status.Missing)
	status.NPXAvailable = status.NPX.Path != ""
	return status
}

func detect(name string, look LookPathFunc, version VersionFunc, missing *[]string) Tool {
	path, ok := look(name)
	if !ok {
		*missing = append(*missing, name)
		return Tool{Name: name}
	}
	return Tool{
		Name:    name,
		Path:    path,
		Version: normalizeVersion(version(name)),
	}
}

func normalizeVersion(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "v")
}

// CheckDefault detects dependencies using the real PATH lookup and `--version`
// invocations. Tests use Check with injected lookups instead.
func CheckDefault() DependencyStatus {
	return Check(SystemLookPath, SystemVersion)
}

func SystemLookPath(name string) (string, bool) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return path, true
}

func SystemVersion(name string) string {
	out, err := exec.Command(name, "--version").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func ConfirmContinueWithoutNPX(in io.Reader, out io.Writer) bool {
	_, _ = fmt.Fprintln(out, "npx is not available. Adding skills will be disabled.")
	_, _ = fmt.Fprint(out, "Continue? [y/N] ")

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes"
}
