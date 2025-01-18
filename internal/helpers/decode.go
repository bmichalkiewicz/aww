package helpers

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

// Project struct to store extracted data
type Project struct {
	fqdn    string
	folders string
	path    string
}

func (p *Project) GetFQDN() string {
	return p.fqdn
}

func (p *Project) GetFolders() string {
	return p.folders
}

func (p *Project) GetPath() string {
	return filepath.Join(p.fqdn, p.folders)
}

// DecodeSSHURL parses an SSH URL into a Project struct
func DecodeSSHURL(sshUrl string) (*Project, error) {
	// Extract FQDN
	fqdnRegexp := regexp.MustCompile(patterns["fqdn"])
	fqdnMatch := fqdnRegexp.FindStringSubmatch(sshUrl)
	if len(fqdnMatch) < 2 {
		return nil, ErrInvalidSSHURL(fmt.Sprintf("FQDN not found in URL: %s", sshUrl))
	}
	fqdn := fqdnMatch[1]

	// Extract Folders (or root folder if no subfolders exist)
	foldersRegexp := regexp.MustCompile(patterns["folders"])
	foldersMatch := foldersRegexp.FindStringSubmatch(sshUrl)
	if len(foldersMatch) < 2 {
		return nil, ErrInvalidSSHURL(fmt.Sprintf("Folders not found in URL: %s", sshUrl))
	}
	folders := foldersMatch[1]

	log.Debug().Str("service", "helpers").Str("folders", folders).Str("fqdn", fqdn).Send()

	return &Project{
		fqdn:    fqdn,
		folders: folders,
	}, nil
}

// ErrInvalidSSHURL represents an error for invalid SSH URL parsing
type ErrInvalidSSHURL string

func (e ErrInvalidSSHURL) Error() string {
	return string(e)
}
