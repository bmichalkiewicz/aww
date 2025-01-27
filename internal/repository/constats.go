package repository

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/rs/zerolog/log"
)

// Define pattern for extracting components from the SSH URL
var pattern = `^git@([a-zA-Z0-9.-]+):([a-zA-Z0-9_./-]+)\.git$`

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
	return filepath.Join(DestRepoPath, p.FQDN, p.Folders)
}

func (p *Project) Validate(url string) error {
	matched, err := regexp.MatchString(pattern, url)
	if err != nil {
		return fmt.Errorf("error validating SSH URL: %v", err)
	}
	if !matched {
		return fmt.Errorf("invalid SSH URL provided (example: git@gitlab.com:goodgroup/goodrepo.git)")
	}

	return nil
}

// Decode parses an SSH URL into a Project struct
func (p *Project) Decode() error {
	// Validate the SSH URL
	err := p.Validate(p.Url)
	if err != nil {
		return err
	}

	regex := regexp.MustCompile(pattern)
	submatches := regex.FindStringSubmatch(p.Url)

	// Ensure the submatches contain the expected groups
	if len(submatches) < 2 {
		return fmt.Errorf("failed to extract groups from SSH URL")
	}

	p.FQDN = submatches[1]
	p.Folders = submatches[2]
	log.Debug().Str("service", "helpers").Str("folders", p.Folders).Str("fqdn", p.FQDN).Send()
	return nil
}
