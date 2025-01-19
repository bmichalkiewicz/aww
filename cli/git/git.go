package git

import (
	"aww/internal/backend"
	"aww/internal/repository"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chelnak/ysmrr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

var (
	Debug     bool
	groups    []*repository.Group
	groupsMap map[string]int
)

func start() error {
	var err error

	if Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	groups, err = repository.Load()
	if err != nil {
		return err
	}

	return nil
}

func overrideGroups(cmd *cli.Command) error {
	groupsMap = make(map[string]int, len(groups))

	for i, group := range groups {
		groupsMap[group.Name] = i
	}

	if cmd.String("repo") != "" {
		groupIndex, exists := groupsMap[cmd.String("repo")]
		if !exists {
			return fmt.Errorf("group '%s' not found", cmd.String("repo"))
		}
		group := groups[groupIndex]
		groups = []*repository.Group{group}
	}

	return nil
}

func isExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Command creates a CLI command for git operations.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "git",
		Usage: "Perform git-related operations on groups and repositories",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				Aliases: []string{"r"},
				Usage:   "Operate only on the specified group name",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "Display a list of all available groups",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := start()
					if err != nil {
						return err
					}

					var builder []string
					for _, group := range groups {
						builder = append(builder, group.Name)
					}

					fmt.Println(strings.Join(builder, "\n"))
					return nil
				},
			},
			{
				Name:  "switch-branch",
				Usage: "Switch to a specific branch for all repositories",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "branch",
						Usage: "The branch to switch to",
						Value: "default",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := start()
					if err != nil {
						return err
					}

					var parseRemote = true
					var branch = cmd.String("branch")

					if cmd.String("branch") != "default" {
						parseRemote = false
					}

					var combinedError []error

					for _, group := range groups {

						for _, project := range group.Projects {

							path := filepath.Join(repository.DestRepoPath, project.GetPath())

							ok, _ := isExist(path)
							if !ok {
								log.Warn().Str("path", path).Msg("Repository not found")
								continue
							}
							var repoBranch string

							if parseRemote {
								info, err := backend.Git.SymbolicRef(&backend.Options{
									Dir: path,
								})
								if err != nil {
									combinedError = append(combinedError, fmt.Errorf("failed to determine symbolic ref for repository %s: %w", project.Url, err))
									continue
								}

								parts := strings.Split(strings.TrimSpace(info), "/")
								if len(parts) == 0 {
									combinedError = append(combinedError, fmt.Errorf("unexpected symbolic ref format for repository %s: %s", project.Url, info))
									continue
								}
								repoBranch = parts[len(parts)-1]
							} else {
								repoBranch = branch
							}

							if repoBranch == "" {
								combinedError = append(combinedError, fmt.Errorf("branch name is empty for repository %s", project.Url))
								continue
							}

							log.Debug().Str("branch", repoBranch).Str("repo", project.Url).Msg("Switching branch")

							// Checkout branch
							err = backend.Git.Checkout(&backend.Options{
								Dir:    path,
								Branch: repoBranch,
							})
							if err != nil {
								combinedError = append(combinedError, fmt.Errorf("failed to checkout branch %s in repository %s: %w", repoBranch, project.Url, err))
								continue
							}
						}
					}

					if len(combinedError) > 0 {
						return errors.Join(combinedError...)
					}

					log.Info().Msg("Switching branches finished ✅")
					return nil
				},
			},
			{
				Name:  "find",
				Usage: "Find repositories based on specific conditions",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "empty",
						Usage: "Identify repositories that are empty",
					},
					&cli.BoolFlag{
						Name:  "uncommitted",
						Usage: "Locate repositories with uncommitted changes",
					},
					&cli.BoolFlag{
						Name:  "unpushed",
						Usage: "Find repositories with unpushed commits",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := start()
					if err != nil {
						return err
					}

					// Determine which conditional option is selected
					var condition conditionalOption
					if cmd.Bool("empty") {
						condition = Empty
					} else if cmd.Bool("uncommitted") {
						condition = Uncommitted
					} else if cmd.Bool("unpushed") {
						condition = Unpushed
					} else {
						return fmt.Errorf("please specify a condition: --empty, --uncommitted, or --unpushed")
					}

					for _, group := range groups {
						for _, project := range group.Projects {
							path := filepath.Join(repository.DestRepoPath, project.GetPath())

							ok, err := isExist(path)
							if err != nil {
								return err
							}
							if !ok {
								log.Warn().Str("path", path).Msg("Repository not found")
								continue
							}

							switch condition {
							case Empty:
								files, err := os.ReadDir(path)
								if err != nil {
									return err
								}
								ok, err := isExist(filepath.Join(path, ".git"))
								if err != nil {
									return err
								}
								if ok && len(files) == 1 {
									fmt.Println(path)
								}
							case Uncommitted:
								status, err := backend.Git.Status(true, &backend.Options{
									Dir: path,
								})
								if err != nil {
									return err
								}

								if status != "" {
									fmt.Println(path)
								}

							case Unpushed:
								unpushed, err := backend.Git.Cherry(&backend.Options{
									Dir: path,
								})
								if err != nil {
									return err
								}
								if unpushed != "" {
									fmt.Println(path)
								}
							}
						}
					}
					log.Info().Msg("Searching finished ✅")
					return nil
				},
			},
			{
				Name:  "clone",
				Usage: "Clone all repositories for the specified groups",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := start()
					if err != nil {
						return err
					}

					// Initialize spinner manager
					sm := ysmrr.NewSpinnerManager()

					var wg sync.WaitGroup
					var combinedError []error

					for _, group := range groups {
						if len(group.Projects) == 0 {
							log.Warn().Str("group", group.Name).Msg("Doesn't have any projects")
							continue
						}

						// Add a spinner for each group
						spinner := sm.AddSpinner(group.Name)
						wg.Add(1)

						go func(group *repository.Group, spinner *ysmrr.Spinner) {
							spinner.UpdateMessage(group.Name + " processing...")
							defer wg.Done()
							defer spinner.CompleteWithMessage(group.Name + " done!")

							for _, project := range group.Projects {
								path := filepath.Join(repository.DestRepoPath, project.GetPath())

								if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
									err = backend.Git.Clone(
										&backend.Options{
											Url: project.Url,
											Dir: path,
										},
									)
									if err != nil {
										combinedError = append(combinedError, fmt.Errorf("failed to clone repository %s: %w", project.Url, err))
										continue
									}
								}
							}
						}(group, spinner)
					}

					sm.Start()
					wg.Wait()
					sm.Stop()

					if len(combinedError) > 0 {
						return errors.Join(combinedError...)
					}

					log.Info().Msg("Cloning finished ✅")
					return nil
				},
			},
			actionsCmd(),
		},
	}
}
