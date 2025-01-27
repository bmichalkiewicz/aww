package cmd

import (
	"aww/internal/backend"
	"aww/internal/repository"
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// run is action for run command
func run(project *repository.Project) error {
	projectPath := project.GetPath()
	commitMsg := project.Commit
	performPush := project.Push != nil && *project.Push
	performCommit := commitMsg != ""

	if !performCommit && !performPush {
		log.Debug().Str("path", projectPath).Msg("No commit or push actions specified. Skipping...")
		return nil
	}

	if performCommit {
		// Check for changes
		ok, err := ifUncomitted(projectPath)
		if err != nil {
			return fmt.Errorf("checking if uncommitted failed for %s: %w", projectPath, err)
		}

		if !ok {
			log.Warn().Str("path", projectPath).Msg("No changes found. Skipping commit...")
			return nil
		}

		// Perform commit
		log.Debug().Str("path", projectPath).Str("message", commitMsg).Msg("Performing commit...")
		err = backend.Git.Add(&backend.Options{
			Dir: projectPath,
		})
		if err != nil {
			return fmt.Errorf("add failed for %s: %w", projectPath, err)
		}

		err = backend.Git.Commit(&backend.Options{
			Dir:       projectPath,
			CommitMsg: commitMsg,
		})
		if err != nil {
			return fmt.Errorf("commit failed for %s: %w", projectPath, err)
		}
		log.Info().Str("path", projectPath).Str("commitMsg", commitMsg).Msg("Commit successful")
	}

	if performPush {
		// Perform push
		log.Debug().Str("path", projectPath).Msg("Performing push...")
		ok, err := ifUnpushed(projectPath)
		if err != nil {
			return fmt.Errorf("push failed for %s: %w", projectPath, err)
		}
		if !ok {
			log.Info().Str("path", projectPath).Msg("No commits to push found")
			return nil
		}
		err = backend.Git.Push(&backend.Options{Dir: projectPath})
		if err != nil {
			return fmt.Errorf("push failed for %s: %w", projectPath, err)
		}
		log.Info().Str("path", projectPath).Msg("Push successful")
	}

	return nil
}

// reset is action for reset command
func reset(project *repository.Project) error {
	project.Commit = ""
	project.Push = nil
	return nil
}

func Actions() *cli.Command {
	return &cli.Command{
		Name:  "actions",
		Usage: "Perform actions on groups and repositories",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				Aliases: []string{"r"},
				Usage:   "Operate only on the specified group name",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run actions specified in the configuration",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := start()
					if err != nil {
						return err
					}

					err = overrideGroups(cmd)
					if err != nil {
						return err
					}

					var combinedErrors []error

					for _, group := range groups {
						if len(group.Projects) == 0 {
							log.Warn().Str("group", group.Name).Msg("Doesn't contain any projects")
							continue
						}
						// Execute the provided action
						err = processProjects(group.Projects, run)
						if err != nil {
							combinedErrors = append(combinedErrors, err)
						}
					}

					if len(combinedErrors) > 0 {
						log.Error().Msgf("Errors encountered: %v", combinedErrors)
						return errors.Join(combinedErrors...)
					}

					err = repository.Save(groups)
					if err != nil {
						return err
					}
					log.Info().Msg("All actions completed successfully ✅")
					return nil
				},
			},
			{
				Name:  "reset",
				Usage: "Reset configuration",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := start()
					if err != nil {
						return err
					}

					err = overrideGroups(cmd)
					if err != nil {
						return err
					}

					var combinedErrors []error

					for _, group := range groups {
						if len(group.Projects) == 0 {
							log.Warn().Str("group", group.Name).Msg("Doesn't contain any projects")
							continue
						}
						// reset group settings
						group.Skip = false
						group.Commit = ""
						group.Push = nil
						// Execute the provided action
						err = processProjects(group.Projects, reset)
						if err != nil {
							combinedErrors = append(combinedErrors, err)
						}
					}

					if len(combinedErrors) > 0 {
						log.Error().Msgf("Errors encountered: %v", combinedErrors)
						return errors.Join(combinedErrors...)
					}

					err = repository.Save(groups)
					if err != nil {
						return err
					}
					log.Info().Msg("All actions completed successfully ✅")
					return nil
				},
			},
		},
	}
}
