package cli

import (
	"context"
	"dusa/internal/backend"
	config "dusa/internal/config"
	"dusa/internal/helpers"
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
	debug     bool
	groups    []config.GroupTemplate
	groupsMap map[string]int
)

func initialize(cmd *cli.Command) error {
	var err error

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	groups, err = config.Load()
	if err != nil {
		return err
	}

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
		groups = []config.GroupTemplate{group}
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

// addGitCmd creates a CLI command for git operations.
func addGitCmd() *cli.Command {
	return &cli.Command{
		Name:  "git",
		Usage: "Manage and interact with git repositories",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				Aliases: []string{"r"},
				Usage:   "Specify a single group name to operate on",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List available groups",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := initialize(cmd)
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
				Usage: "Switch branch",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "branch",
						Usage: "Specify a branch to switch",
						Value: "default",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := initialize(cmd)
					if err != nil {
						return err
					}

					var parseRemote = true
					var branch = cmd.String("branch")

					// Initialize spinner manager
					sm := ysmrr.NewSpinnerManager()

					if cmd.String("branch") != "default" {
						parseRemote = false
					}

					var combinedError error

					sm.Start()

					for _, group := range groups {

						spinner := sm.AddSpinner(group.Name)
						spinner.UpdateMessagef("%s: switching branches...", group.Name)

						for _, project := range group.Projects {
							projectDecoded, err := helpers.DecodeSSHURL(project.Url)
							if err != nil {
								spinner.ErrorWithMessagef("%s: failed", group.Name)
								combinedError = fmt.Errorf("%v; failed to decode SSH URL %s: %w", combinedError, project.Url, err)
								continue
							}

							path := filepath.Join(config.RootFolder, projectDecoded.GetPath())
							var repoBranch string

							if parseRemote {
								info, err := backend.Git.SymbolicRef(&backend.Options{
									Dir: path,
								})
								if err != nil {
									spinner.ErrorWithMessagef("%s: failed", group.Name)
									combinedError = fmt.Errorf("%v; failed to determine symbolic ref for repository %s: %w", combinedError, project.Url, err)
									continue
								}

								parts := strings.Split(strings.TrimSpace(info), "/")
								if len(parts) == 0 {
									spinner.ErrorWithMessagef("%s: failed", group.Name)
									combinedError = fmt.Errorf("%v; unexpected symbolic ref format for repository %s: %s", combinedError, project.Url, info)
									continue
								}
								repoBranch = parts[len(parts)-1]
							} else {
								repoBranch = branch
							}

							if repoBranch == "" {
								spinner.ErrorWithMessagef("%s: failed", group.Name)
								combinedError = fmt.Errorf("%v; branch name is empty for repository %s", combinedError, project.Url)
								continue
							}

							log.Debug().Msgf("Switching to branch '%s' in repository '%s'", repoBranch, project.Url)

							// Checkout branch
							err = backend.Git.Checkout(&backend.Options{
								Dir:    path,
								Branch: repoBranch,
							})
							if err != nil {
								spinner.ErrorWithMessagef("%s: failed", group.Name)
								combinedError = fmt.Errorf("%v; failed to checkout branch %s in repository %s: %w", combinedError, repoBranch, project.Url, err)
								continue
							}
						}

						spinner.CompleteWithMessagef("%s: done", group.Name)
					}

					sm.Stop()

					if combinedError != nil {
						return combinedError
					}

					log.Info().Msg("Switching branches finished ✅")
					return nil
				},
			},
			{
				Name:  "find",
				Usage: "Find repositories that meet a given condition",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "empty",
						Usage: "Find empty repositories",
					},
					&cli.BoolFlag{
						Name:  "uncommitted",
						Usage: "Find repositories with uncommitted changes",
					},
					&cli.BoolFlag{
						Name:  "unpushed",
						Usage: "Find repositories with unpushed commits",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := initialize(cmd)
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
							projectDecoded, err := helpers.DecodeSSHURL(project.Url)
							if err != nil {
								return err
							}

							path := filepath.Join(config.RootFolder, projectDecoded.GetPath())

							ok, err := isExist(path)
							if err != nil {
								return err
							}
							if !ok {
								return fmt.Errorf("repository %q doesn't exists or empty, please repeat cloning", path)
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
				Usage: "Clone repositories based on group configurations",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "repo",
						Aliases: []string{"r"},
						Usage:   "Specify a single group name to operate on",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := initialize(cmd)
					if err != nil {
						return err
					}

					// Initialize spinner manager
					sm := ysmrr.NewSpinnerManager()

					var wg sync.WaitGroup
					errChan := make(chan error, len(groups))

					for _, group := range groups {
						if len(group.Projects) == 0 {
							continue
						}

						// Add a spinner for each group
						spinner := sm.AddSpinner(group.Name)
						wg.Add(1)

						go func(group config.GroupTemplate, spinner *ysmrr.Spinner) {
							spinner.UpdateMessage(group.Name + " processing...")
							defer wg.Done()
							defer spinner.CompleteWithMessage(group.Name + " done!")

							for _, project := range group.Projects {
								projectDecoded, err := helpers.DecodeSSHURL(project.Url)
								if err != nil {
									errChan <- fmt.Errorf("failed to decode SSH URL %s: %w", project.Url, err)
									return
								}

								path := filepath.Join(config.RootFolder, projectDecoded.GetPath())

								if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
									err = backend.Git.Clone(
										&backend.Options{
											Url: project.Url,
											Dir: path,
										},
									)
									if err != nil {
										errChan <- fmt.Errorf("failed to clone repository %s: %w", project.Url, err)
										return
									}
								}
							}
						}(group, spinner)
					}

					sm.Start()
					wg.Wait()
					sm.Stop()
					close(errChan)

					// Collect and handle errors
					var combinedError error
					for err := range errChan {
						if combinedError == nil {
							combinedError = err
						} else {
							combinedError = fmt.Errorf("%v; %w", combinedError, err)
						}
					}

					if combinedError != nil {
						return combinedError
					}

					log.Info().Msg("Cloning finished ✅")
					return nil
				},
			},
		},
	}
}
