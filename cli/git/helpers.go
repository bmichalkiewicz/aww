package git

import (
	"aww/internal/backend"
	"aww/internal/repository"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

func ifUnpushed(projectPath string) (bool, error) {
	unpushed, err := backend.Git.Cherry(&backend.Options{
		Dir: projectPath,
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
	status, err := backend.Git.Status(true, &backend.Options{
		Dir: projectPath,
	})
	if err != nil {
		return false, err
	}
	if status != "" {
		return true, nil
	}

	return false, nil
}

// Utility function to process groups and projects
func processGroupsAndProjects(handler func(group *repository.Group, project *repository.Project, projectPath string) error) error {
	var combinedError []error

	for _, group := range groups {
		if len(group.Projects) == 0 {
			log.Warn().Str("group", group.Name).Msg("Doesn't contain any projects")
			continue
		}

		for _, project := range group.Projects {
			projectPath := filepath.Join(repository.DestRepoPath, project.GetPath())

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
			err = handler(group, project, projectPath)
			if err != nil {
				combinedError = append(combinedError, err)
			}
		}
	}

	if len(combinedError) > 0 {
		return errors.Join(combinedError...)
	}

	return nil
}
