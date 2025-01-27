package repository

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/rs/zerolog/log"
)

// Define patterns for extracting components from the SSH URL
var (
	patterns = map[string]string{
		"fqdn":    `git@([\w\.-]+):`, // Matches the FQDN
		"folders": `:(.+)\.git$`,     // Matches folders
	}
)

type Group struct {
	Name     string     `yaml:"name"`
	Skip     bool       `yaml:"skip"`
	Commit   string     `yaml:"commit,omitempty"`
	Push     *bool      `yaml:"push,omitempty"`
	Projects []*Project `yaml:"projects,omitempty"`
}

type Project struct {
	Url    string `yaml:"url"`
	Commit string `yaml:"commit,omitempty"`
	Push   *bool  `yaml:"push,omitempty"`

	FQDN    string `yaml:"-"`
	Folders string `yaml:"-"`
	Path    string `yaml:"-"`
}

func (p *Project) GetFQDN() string {
	return p.FQDN
}

func (p *Project) GetFolders() string {
	return p.Folders
}

func (p *Project) GetPath() string {
	return filepath.Join(p.FQDN, p.Folders)
}

// Decode parses an SSH URL into a Project struct
func (p *Project) Decode() error {
	// Extract FQDN
	fqdnRegexp := regexp.MustCompile(patterns["fqdn"])
	fqdnMatch := fqdnRegexp.FindStringSubmatch(p.Url)
	if len(fqdnMatch) < 2 {
		return ErrInvalidUrl(fmt.Sprintf("FQDN not found in URL: %s", p.Url))
	}
	p.FQDN = fqdnMatch[1]

	// Extract Folders (or root folder if no subfolders exist)
	foldersRegexp := regexp.MustCompile(patterns["folders"])
	foldersMatch := foldersRegexp.FindStringSubmatch(p.Url)
	if len(foldersMatch) < 2 {
		return ErrInvalidUrl(fmt.Sprintf("Folders not found in URL: %s", p.Url))
	}
	p.Folders = foldersMatch[1]

	log.Debug().Str("service", "helpers").Str("folders", p.Folders).Str("fqdn", p.FQDN).Send()
	return nil
}

// ErrInvalidUrl represents an error for invalid SSH URL parsing
type ErrInvalidUrl string

func (e ErrInvalidUrl) Error() string {
	return string(e)
}
