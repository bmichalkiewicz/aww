package repository

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Variables for default file paths and GitLab URL
var (
	// HomeDirectory is the user's home directory.
	HomeDirectory, _ = os.UserHomeDir()
	// RepositoryPath is the path to the configuration folder.
	RepositoryPath = filepath.Join(HomeDirectory, ".aww")
	// RepositoryFilePath is the path to the configuration file.
	RepositoryFilePath = filepath.Join(RepositoryPath, "repositories.yaml")
	// Main root folder
	DestRepoPath = filepath.Join(HomeDirectory, "aww")
)

// LoadTemplate loads the template from the file.
func Load() ([]*Group, error) {
	template := []*Group{}

	templateFile, err := os.Open(RepositoryFilePath)
	if errors.Is(err, os.ErrNotExist) {
		// Return error if the file doesn't exist
		return nil, fmt.Errorf("%s not found", RepositoryFilePath)
	} else if err != nil {
		return nil, fmt.Errorf("error opening template file: %w", err)
	}
	defer templateFile.Close()

	byteValue, err := io.ReadAll(templateFile)
	if err != nil {
		return nil, fmt.Errorf("error reading template file: %w", err)
	}

	if err := yaml.Unmarshal(byteValue, &template); err != nil {
		return nil, fmt.Errorf("error parsing template file: %w", err)
	}

	if len(template) == 0 {
		return nil, fmt.Errorf("no groups found")
	}

	return template, nil
}

// Save updates the repository file.
func Save(repositories []*Group) error {
	// Open the file for writing
	file, err := os.OpenFile(RepositoryFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening template file for writing: %w", err)
	}
	defer file.Close()

	// Write the template to the file
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)

	if err := encoder.Encode(repositories); err != nil {
		return fmt.Errorf("error encoding template content: %w", err)
	}

	return nil
}

// Init checking if repository path is exists
func Init() error {
	if _, err := os.Stat(RepositoryPath); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(RepositoryPath, 0755); err != nil {
			return fmt.Errorf("error creating config directory: %w", err)
		}
		return nil
	}
	return nil
}
