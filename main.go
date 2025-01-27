package main

import (
	"aww/cmd"
	"aww/internal/repository"
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func main() {
	repository.Init()

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out: os.Stdout,
	})

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

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
				Destination: &cmd.Debug,
			},
		},
		Commands: []*cli.Command{
			cmd.Git(),
		},
	}

	// Run the application
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Application encountered an error")
	}

}
