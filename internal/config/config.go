// Package config provides configuration management for a Go application.
package config

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
	// ConfigFilePath is the path to the configuration file.
	ConfigFilePath = filepath.Join(HomeDirectory, ".brr", "repositories.yaml")
	// Main root folder
	RootFolder = filepath.Join(HomeDirectory, "aww")
)

// LoadTemplate loads the template from the file.
func Load() ([]GroupTemplate, error) {
	template := []GroupTemplate{}

	templateFile, err := os.Open(ConfigFilePath)
	if errors.Is(err, os.ErrNotExist) {
		// Return error if the file doesn't exist
		return nil, fmt.Errorf("%s not found", ConfigFilePath)
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
