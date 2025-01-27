package main

import (
	"aww/cli"
	"aww/internal/repository"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// CreateTemplate creates or updates the template file.
func repositoryPathExists(path string) error {

	// Ensure path directory exists
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("error creating config directory: %w", err)
		}

		return nil
	}
	return nil
}

func main() {
	repositoryPathExists(repository.RepositoryPath)

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out: os.Stdout,
	})

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	app := cli.New()

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Error().Msgf("error: %s", err.Error())
		os.Exit(1)
	}
}
