package main

import (
	"context"
	"strings"
	"testing"
)

// setVersion swaps the build-time version string for the duration of a test.
// newCommand reads version at construction, so set it before building.
func setVersion(t *testing.T, v string) {
	t.Helper()
	old := version
	version = v
	t.Cleanup(func() { version = old })
}

// runArgs builds the command with a launch spy and runs it with the given
// args (args[0] is the command name). It reports whether launch was called and
// what the command wrote to stdout/stderr.
func runArgs(t *testing.T, args ...string) (launched bool, stdout, stderr string, err error) {
	t.Helper()
	var out, errOut strings.Builder
	cmd := newCommand(func() error { launched = true; return nil }, &out, &errOut)
	err = cmd.Run(context.Background(), args)
	return launched, out.String(), errOut.String(), err
}

func TestVersionFlagPrintsVersionAndDoesNotLaunch(t *testing.T) {
	setVersion(t, "9.9.9-test")

	launched, stdout, _, err := runArgs(t, "trainer", "--version")

	if err != nil {
		t.Fatalf("--version returned error: %v", err)
	}
	if launched {
		t.Fatal("--version should not launch the TUI")
	}
	if !strings.Contains(stdout, "9.9.9-test") {
		t.Fatalf("--version stdout = %q, want it to contain %q", stdout, "9.9.9-test")
	}
}

func TestHelpFlagPrintsUsageAndDoesNotLaunch(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		launched, stdout, _, err := runArgs(t, "trainer", flag)

		if err != nil {
			t.Fatalf("%s returned error: %v", flag, err)
		}
		if launched {
			t.Fatalf("%s should not launch the TUI", flag)
		}
		for _, want := range []string{"trainer", "browse and manage", "--version", "--help"} {
			if !strings.Contains(stdout, want) {
				t.Fatalf("%s usage %q missing %q", flag, stdout, want)
			}
		}
	}
}

func TestNoArgsLaunches(t *testing.T) {
	launched, stdout, stderr, err := runArgs(t, "trainer")

	if err != nil {
		t.Fatalf("no args returned error: %v", err)
	}
	if !launched {
		t.Fatal("no args should launch the TUI")
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("launching should print nothing; stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestUnknownFlagErrorsWithoutLaunch(t *testing.T) {
	launched, _, _, err := runArgs(t, "trainer", "--nope")

	if err == nil {
		t.Fatal("an unknown flag should return an error")
	}
	if launched {
		t.Fatal("an unknown flag should not launch the TUI")
	}
}
