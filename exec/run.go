package exec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// New returns Runner with default settings
func New() *Runner {
	return &Runner{
		silent: false,
		output: false,
	}
}

type Runner struct {
	dir    string
	silent bool
	output bool
}

// CommandRunner is for running the command
var CommandRunner = func(cmd *exec.Cmd) error {
	return cmd.Run()
}

// Silent enables silent mode for the runner and returns the runner.
func (r *Runner) Silent() *Runner {
	r.silent = true
	return r
}

// Output enables output capturing for the runner and returns the runner.
func (r *Runner) Output() *Runner {
	r.output = true
	return r
}

// Dir sets the working directory for the runner and returns the runner.
func (r *Runner) Dir(path string) *Runner {
	r.dir = path
	return r
}

// Go executes a command with behavior determined by Runner's fields.
// - If `output` is true, captures and returns the command's stdout.
// - If `silent` is true, suppresses logs and command output.
// - If `dir` is set, runs the command in the specified directory.
func (r *Runner) Go(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)

	// Set working directory if specified
	if r.dir != "" {
		cmd.Dir = r.dir
	}

	// Set output streams
	var outputBuffer bytes.Buffer
	if r.silent {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard

		if r.output {
			cmd.Stdout = &outputBuffer
		}
	} else {
		cmd.Stderr = os.Stderr
		if r.output {
			cmd.Stdout = &outputBuffer
		} else {
			cmd.Stdout = os.Stderr
		}
	}

	// Log command if not silent
	if !r.silent {
		log.Info().Msgf("Running command: %s %s", cmd.Args[0], strings.Join(cmd.Args[1:], " "))
	}

	// Run the command
	err := CommandRunner(cmd)
	if err != nil {
		return outputBuffer.String(), &RunError{cmd, err}
	}

	// Return captured output if `output` is enabled
	if r.output {
		return outputBuffer.String(), nil
	}

	// Return an empty string if no output is captured
	return "", nil
}

// RunError represents an error that occurred while running a command.
type RunError struct {
	Command   *exec.Cmd
	ExecError error
}

// Error implements the error interface for RunError.
func (e *RunError) Error() string {
	return fmt.Sprintf("%s: %s", e.Command.Path, e.ExecError)
}
