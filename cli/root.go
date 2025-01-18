package cli

import (
	"aww/cli/git"

	"github.com/urfave/cli/v3"
)

// New creates a new CLI application instance.
func New() *cli.Command {
	// Create the main CLI application.
	app := &cli.Command{
		Name:                  "aww",
		EnableShellCompletion: true,
		Usage:                 "Git repositories and token management",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "sets log level to debug",
				Sources:     cli.EnvVars("DEBUG"),
				Value:       false,
				Destination: &git.Debug,
			},
		},
	}

	// Build the Commands
	app.Commands = append(app.Commands, git.Command())

	return app
}
