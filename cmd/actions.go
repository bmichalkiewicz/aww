package cmd

import (
	"aww/internal/backend"
	"aww/internal/repository"
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// run is action for run command
func run(project *repository.Project, groupActions *repository.GroupActions) error {
	projectPath := project.GetPath()
	if project.Actions == nil && groupActions == nil {
		return nil
	}

	// Determine commit and push actions
	commitMsg := ""
	if project.Actions != nil {
		commitMsg = project.Actions.Commit
	}
	if commitMsg == "" && groupActions != nil {
		commitMsg = groupActions.Commit
	}

	performPush := false
	if project.Actions != nil && project.Actions.Push != nil {
		performPush = *project.Actions.Push
	} else if groupActions != nil && groupActions.Push != nil {
		performPush = *groupActions.Push
	}

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

// plan is action for plan command
func plan(project *repository.Project, groupActions *repository.GroupActions) error {
	projectPath := project.GetPath()
	if project.Actions == nil && groupActions == nil {
		return nil
	}

	// Determine commit and push actions
	commitMsg := ""
	if project.Actions != nil {
		commitMsg = project.Actions.Commit
	}
	if commitMsg == "" && groupActions != nil {
		commitMsg = groupActions.Commit
	}

	performPush := false
	if project.Actions != nil && project.Actions.Push != nil {
		performPush = *project.Actions.Push
	} else if groupActions != nil && groupActions.Push != nil {
		performPush = *groupActions.Push
	}

	performCommit := commitMsg != ""

	if !performCommit && !performPush {
		log.Debug().Str("path", projectPath).Msg("No commit or push actions specified. Skipping...")
		return nil
	}

	var commitPerformed, pushPerformed bool
	var outputBuffer string

	if performCommit {
		// Check for changes
		ok, err := ifUncomitted(projectPath)
		if err != nil {
			return fmt.Errorf("checking if uncommitted failed for %s: %w", projectPath, err)
		}
		if ok {
			commitPerformed = true
		}
	}

	if performPush {
		ok, err := ifUnpushed(projectPath)
		if err != nil {
			return fmt.Errorf("push failed for %s: %w", projectPath, err)
		}
		if ok {
			pushPerformed = true
		}
	}

	// Build output in a buffer with colors
	header := color.New(color.FgHiCyan, color.Bold).SprintFunc()
	success := color.New(color.FgGreen).SprintFunc()
	failure := color.New(color.FgRed).SprintFunc()
	outputBuffer += fmt.Sprintf("Project: %s\n", header(projectPath))
	outputBuffer += "└── Actions:\n"

	if performCommit {
		if commitPerformed {
			outputBuffer += fmt.Sprintf("    ├── Commit: %s\n", success(commitMsg))
		} else {
			outputBuffer += fmt.Sprintf("    ├── Commit: %s\n", failure("No changes to commit"))
		}
	}
	if performPush {
		if pushPerformed {
			outputBuffer += fmt.Sprintf("    └── Push: %s\n", success("true"))
		} else {
			outputBuffer += fmt.Sprintf("    └── Push: %s\n", failure("false"))
		}
	} else {
		outputBuffer += fmt.Sprintf("    └── Push: %s\n", failure("false"))
	}

	// Print the buffered output
	fmt.Print(outputBuffer)

	return nil
}

// reset is action for reset command
func reset(project *repository.Project, groupActions *repository.GroupActions) error {
	if project.Actions == nil && groupActions == nil {
		// No actions defined at both levels, nothing to reset
		return nil
	}

	if project.Actions != nil {
		// Reset project-specific actions
		project.Actions.Reset()
	}

	if groupActions != nil {
		// Reset group-level actions
		groupActions.Reset()
	}

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
				Name:  "apply",
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

					for _, group := range groups {
						if len(group.Projects) == 0 {
							log.Warn().Str("group", group.Name).Msg("Doesn't contain any projects")
							continue
						}
						// Execute the provided action
						err = processProjects(group.Projects, group.Actions, run)
						if err != nil {
							return err
						}
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

					for _, group := range groups {
						if len(group.Projects) == 0 {
							log.Warn().Str("group", group.Name).Msg("Doesn't contain any projects")
							continue
						}

						// Execute the provided action
						err = processProjects(group.Projects, group.Actions, reset)
						if err != nil {
							return err
						}
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
				Name:  "plan",
				Usage: "Check the outgoing changes",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := start()
					if err != nil {
						return err
					}

					err = overrideGroups(cmd)
					if err != nil {
						return err
					}

					for _, group := range groups {
						if len(group.Projects) == 0 {
							log.Warn().Str("group", group.Name).Msg("Doesn't contain any projects")
							continue
						}
						// Execute the provided action
						err = processProjects(group.Projects, group.Actions, plan)
						if err != nil {
							return err
						}
					}
					return nil
				},
			},
		},
	}
}
