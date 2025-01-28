package cmd

import (
	"aww/internal/backend"
	"aww/internal/repository"
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

var (
	Debug     bool
	groups    []*repository.Group
	groupsMap map[string]int
)

// Utility function to process group projects
func processProjects(projects []*repository.Project, groupActions *repository.GroupActions, action projectAction) error {
	var combinedError []error

	for _, project := range projects {
		err := project.Decode()
		if err != nil {
			return fmt.Errorf("problem with decoding project %s: %v", project.Url, err)
		}
		projectPath := project.GetPath()

		// Check if the project path exists
		ok, err := isExist(projectPath)
		if err != nil {
			combinedError = append(combinedError, fmt.Errorf("error checking path for repository %s: %w", project.Url, err))
			continue
		}
		if !ok {
			log.Warn().Str("path", projectPath).Msg("Repository not found")
			continue
		}

		// Execute the custom handler
		err = action(project, groupActions)
		if err != nil {
			combinedError = append(combinedError, err)
		}
	}

	if len(combinedError) > 0 {
		return errors.Join(combinedError...)
	}

	return nil
}

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

func ifUnpushed(projectPath string) (bool, error) {
	unpushed, err := backend.Git.Cherry(&backend.Options{
		Dir:            projectPath,
		AdditionalArgs: []string{"-v"},
	})
	if err != nil {
		return false, err
	}
	if unpushed != "" {
		return true, nil
	}

	return false, nil
}

func ifUncomitted(projectPath string) (bool, error) {
	status, err := backend.Git.Status(&backend.Options{
		Dir:            projectPath,
		AdditionalArgs: []string{"-s"},
	})
	if err != nil {
		return false, err
	}
	if status != "" {
		return true, nil
	}

	return false, nil
}
