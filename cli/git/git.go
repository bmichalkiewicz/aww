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

					log.Info().Msgf("List of all available groups:\n%s", strings.Join(builder, "\n"))
					return nil
				},
			},
			{
				Name:  "switch-branch",
				Usage: "Switch to a specific branch for all repositories, 'defaults' corresponding to git strategy (main for trunk and develop for gitflow)",
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

					branch := cmd.String("branch")
					parseRemote := branch == "default"

					err = processGroupsAndProjects(func(group *repository.Group, project *repository.Project, projectPath string) error {
						var repoBranch string

						if parseRemote {
							info, err := backend.Git.SymbolicRef(&backend.Options{Dir: projectPath})
							if err != nil {
								return fmt.Errorf("failed to determine symbolic ref for repository %s: %w", project.Url, err)
							}

							parts := strings.Split(strings.TrimSpace(info), "/")
							if len(parts) == 0 {
								return fmt.Errorf("unexpected symbolic ref format for repository %s: %s", project.Url, info)
							}
							repoBranch = parts[len(parts)-1]
						} else {
							repoBranch = branch
						}

						if repoBranch == "" {
							return fmt.Errorf("branch name is empty for repository %s", project.Url)
						}

						log.Debug().Str("branch", repoBranch).Str("repo", project.Url).Msg("Switching branch")

						// Checkout branch
						err := backend.Git.Checkout(&backend.Options{
							Dir:    projectPath,
							Branch: repoBranch,
						})
						if err != nil {
							return fmt.Errorf("failed to checkout branch %s in repository %s: %w", repoBranch, project.Url, err)
						}

						return nil
					})

					if err != nil {
						return err
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

					// Determine the condition
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

					err = processGroupsAndProjects(func(group *repository.Group, project *repository.Project, projectPath string) error {
						switch condition {
						case Empty:
							// Check if the repository is empty
							files, err := os.ReadDir(projectPath)
							if err != nil {
								return fmt.Errorf("failed to read directory %s: %w", projectPath, err)
							}
							ok, err := isExist(filepath.Join(projectPath, ".git"))
							if err != nil {
								return fmt.Errorf("failed to check .git folder for %s: %w", projectPath, err)
							}
							if ok && len(files) == 1 {
								fmt.Println(projectPath)
							}

						case Uncommitted:
							// Check for uncommitted changes
							ok, err := ifUncomitted(projectPath)
							if err != nil {
								return fmt.Errorf("failed to check uncommitted changes for %s: %w", projectPath, err)
							}
							if ok {
								fmt.Println(projectPath)
							}

						case Unpushed:
							// Check for unpushed commits
							ok, err := ifUnpushed(projectPath)
							if err != nil {
								return fmt.Errorf("failed to check unpushed commits for %s: %w", projectPath, err)
							}
							if ok {
								fmt.Println(projectPath)
							}
						}
						return nil
					})

					if err != nil {
						return err
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
					sm.Start()
					defer sm.Stop()

					var wg sync.WaitGroup
					var mu sync.Mutex
					var combinedError []error

					for _, group := range groups {
						wg.Add(1)

						go func(group *repository.Group) {
							defer wg.Done()

							// Add a spinner for the group
							spinner := sm.AddSpinner(group.Name)
							spinner.UpdateMessagef("[%s] processing...", group.Name)

							if len(group.Projects) == 0 {
								spinner.ErrorWithMessagef("[%s] no projects found", group.Name)
								log.Warn().Str("group", group.Name).Msg("Doesn't contain any projects")

								return
							}

							for _, project := range group.Projects {
								projectPath := filepath.Join(repository.DestRepoPath, project.GetPath())

								// Check if repository already exists
								if _, err := os.Stat(projectPath); !errors.Is(err, os.ErrNotExist) {
									continue
								}

								// Clone the repository
								err := backend.Git.Clone(&backend.Options{
									Url: project.Url,
									Dir: projectPath,
								})
								if err != nil {
									mu.Lock()
									combinedError = append(combinedError, fmt.Errorf("failed to clone repository %s: %w", project.Url, err))
									mu.Unlock()
									continue
								}
							}

							spinner.CompleteWithMessagef("[%s] done!", group.Name)
						}(group)
					}

					// Wait for all goroutines to complete
					wg.Wait()

					// Handle errors
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
