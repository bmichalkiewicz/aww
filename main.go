package main

import (
	"context"
	"dusa/cli"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
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
