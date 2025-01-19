package backend

import (
	"fmt"
	"os"
	"path/filepath"

	"aww/exec"
)

type Options struct {
	Url       string
	Dir       string
	Branch    string
	CommitMsg string // For commit message
	Remote    string // For push/pull remote
}

// GitBackend defines common git operations
type GitBackend struct {
	Clone       func(*Options) error
	Status      func(short bool, options *Options) (output string, err error)
	Cherry      func(options *Options) (output string, err error)
	Push        func(options *Options) error
	Commit      func(options *Options) error
	Add         func(options *Options) error
	Pull        func(options *Options) error
	Checkout    func(options *Options) error
	SymbolicRef func(options *Options) (output string, err error)
}

// Git provides a GitBackend instance
var Git = &GitBackend{
	// Clone performs git clone operation
	Clone: func(options *Options) error {
		dir, _ := filepath.Split(options.Dir)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}

		args := []string{"clone"}

		if options.Branch != "" {
			args = append(args, "--branch", options.Branch, "--single-branch")
		}

		args = append(args, options.Url, options.Dir)

		_, err = exec.New().Silent().Go("git", args...)
		return err
	},

	// Status retrieves git status with optional short format
	Status: func(short bool, options *Options) (output string, err error) {
		args := []string{"status"}
		if short {
			args = append(args, "-s")
		}
		return exec.New().Dir(options.Dir).Silent().Output().Go("git", args...)
	},

	// Cherry verify if repository has a unpushed commits
	Cherry: func(options *Options) (output string, err error) {
		args := []string{"cherry", "-v"}

		output, err = exec.New().Dir(options.Dir).Silent().Output().Go("git", args...)
		return output, err
	},

	// Push pushes the local branch to the remote
	Push: func(options *Options) error {
		args := []string{"push", options.Remote}
		if options.Branch != "" {
			args = append(args, options.Branch)
		}
		_, err := exec.New().Dir(options.Dir).Silent().Go("git", args...)
		return err
	},

	// Commit commits changes with the provided message
	Commit: func(options *Options) error {
		if options.CommitMsg == "" {
			return fmt.Errorf("commit message cannot be empty")
		}

		args := []string{"commit", "-m", options.CommitMsg}
		_, err := exec.New().Dir(options.Dir).Silent().Go("git", args...)
		return err
	},

	// Add add changes
	Add: func(options *Options) error {
		args := []string{"add", "."}
		_, err := exec.New().Dir(options.Dir).Silent().Go("git", args...)
		return err
	},

	// Pull pulls the latest changes from the remote
	Pull: func(options *Options) error {
		args := []string{"pull", options.Remote}
		if options.Branch != "" {
			args = append(args, options.Branch)
		}
		_, err := exec.New().Dir(options.Dir).Silent().Go("git", args...)
		return err
	},
	Checkout: func(options *Options) error {
		args := []string{"checkout", options.Branch}

		_, err := exec.New().Dir(options.Dir).Silent().Go("git", args...)
		return err
	},

	// SymbolicRef shows information about remote repository (default branch etc.)
	SymbolicRef: func(options *Options) (output string, err error) {
		args := []string{"symbolic-ref", "refs/remotes/origin/HEAD"}

		output, err = exec.New().Dir(options.Dir).Silent().Output().Go("git", args...)
		return output, err
	},
}
