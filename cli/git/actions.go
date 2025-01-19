package git

import (
	"aww/internal/backend"
	"aww/internal/repository"
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func actionsCmd() *cli.Command {
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

					err = processGroups(runActions, false)
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

					err = processGroups(resetActions, true)
					if err != nil {
						return err
					}
					log.Info().Msg("All actions reset successfully ✅")
					return nil
				},
			},
		},
	}
}

// Helper to process all groups and their projects
func processGroups(action func(*repository.Group, *repository.Project, string) error, updateRepositoryFile bool) error {
	var combinedErrors []error

	for _, group := range groups {
		for _, project := range group.Projects {
			projectPath := filepath.Join(repository.DestRepoPath, project.GetPath())

			// Validate repository existence
			exists, err := isExist(projectPath)
			if err != nil {
				combinedErrors = append(combinedErrors, fmt.Errorf("failed to check existence for %s: %w", projectPath, err))
				continue
			}
			if !exists {
				log.Warn().Str("path", projectPath).Msg("Repository not found. Skipping...")
				continue
			}

			// Execute the provided action
			err = action(group, project, projectPath)
			if err != nil {
				combinedErrors = append(combinedErrors, err)
			}
		}
	}

	if updateRepositoryFile {
		repository.Save(groups)
	}

	if len(combinedErrors) > 0 {
		log.Error().Msgf("Errors encountered: %v", combinedErrors)
		return errors.Join(combinedErrors...)
	}
	return nil
}

// Action logic for "run"
func runActions(group *repository.Group, project *repository.Project, projectPath string) error {
	commitMsg := project.Commit
	if commitMsg == "" {
		commitMsg = group.Commit
	}

	performPush := false
	if project.Push != nil {
		performPush = *project.Push // Use the value of project.Push
	} else if group.Push != nil {
		performPush = *group.Push // Use the group.Push as fallback
	}

	performCommit := commitMsg != ""

	if !performCommit && !performPush {
		log.Debug().Str("path", projectPath).Msg("No commit or push actions specified. Skipping...")
		return nil
	}

	if performCommit {
		// Check for changes
		status, err := backend.Git.Status(true, &backend.Options{Dir: projectPath})
		if err != nil {
			return fmt.Errorf("failed to get status for %s: %w", projectPath, err)
		}

		if status == "" {
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
		fmt.Println("lol")
		err = backend.Git.Commit(&backend.Options{
			Dir:       projectPath,
			CommitMsg: commitMsg,
		})
		if err != nil {
			return fmt.Errorf("commit failed for %s: %w", projectPath, err)
		}
		log.Info().Str("path", projectPath).Msg("Commit successful")
	}

	if performPush {
		// Perform push
		log.Debug().Str("path", projectPath).Msg("Performing push...")
		err := backend.Git.Push(&backend.Options{Dir: projectPath})
		if err != nil {
			return fmt.Errorf("push failed for %s: %w", projectPath, err)
		}
		log.Info().Str("path", projectPath).Msg("Push successful")
	}

	return nil
}

// Action logic for "reset"
func resetActions(group *repository.Group, project *repository.Project, projectPath string) error {
	project.Commit = ""
	project.Push = nil
	log.Info().Str("path", projectPath).Msg("Commit and push actions reset")
	return nil
}
