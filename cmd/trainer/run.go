package main

import (
	"context"
	"io"

	"github.com/urfave/cli/v3"
)

// version is the build-time version string. GoReleaser overrides it with
// -ldflags "-X main.version=<tag>"; it stays "dev" for local builds.
var version = "dev"

// newCommand builds the trainer command-line interface. urfave/cli owns flag
// parsing, --version, and --help so the binary runs and exits cleanly without a
// terminal (which the Homebrew formula test needs) and so subcommands can be
// added later. launch runs the TUI and is the default action when no flag
// short-circuits; it is a parameter so tests can observe whether an invocation
// would launch or was handled by a flag.
func newCommand(launch func() error, stdout, stderr io.Writer) *cli.Command {
	return &cli.Command{
		Name:      "trainer",
		Usage:     "browse and manage installed agent skills",
		Version:   version,
		Writer:    stdout,
		ErrWriter: stderr,
		Action: func(context.Context, *cli.Command) error {
			return launch()
		},
	}
}
