package utils

import (
	"bufio"
	"bytes"
	"context"
	"dusa/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/input"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// getPassword prompts the user for a password securely.
func getPassword() (string, error) {
	password, err := prompt.New().Ask("Input your password:").
		Input("", input.WithEchoMode(input.EchoPassword))

	if err != nil {
		return "", fmt.Errorf("problem with getting password: %w", err)
	}

	return strings.TrimSpace(password), nil
}

// getUsername prompts the user for a username if not provided.
func getUsername(username string) (string, error) {
	if username != "" {
		return username, nil
	}

	username, err := prompt.New().Ask("Input your username:").Input("")
	if err != nil {
		return "", fmt.Errorf("can't read the username: %w", err)
	}

	return strings.TrimSpace(username), nil
}

// updateBashrc updates the .bashrc file with the new token and creates a backup.
func updateBashrc(token string) error {
	// Define the path to .bashrc and the backup file
	bashrcPath := filepath.Join(config.HomeDirectory, ".bashrc")
	backupPath := filepath.Join(config.HomeDirectory, ".bashrc.backup")

	// Create a backup of the .bashrc file
	err := backupBashrc(bashrcPath, backupPath)
	if err != nil {
		return fmt.Errorf("failed to create .bashrc backup: %w", err)
	}

	// Open the .bashrc file for reading and writing
	bashrcFile, err := os.OpenFile(bashrcPath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("unable to open .bashrc file: %w", err)
	}
	defer bashrcFile.Close()

	// Read the existing content of .bashrc
	var content []string
	scanner := bufio.NewScanner(bashrcFile)
	for scanner.Scan() {
		content = append(content, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .bashrc file: %w", err)
	}

	// Prepare the new export line with a timestamp comment
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	exportLine := fmt.Sprintf("export TOKEN_SIGMA=\"%s\"", token)
	comment := fmt.Sprintf("# Created by `aww` on %s", currentTime)

	// Update or add the export line
	var found bool
	for i, line := range content {
		if strings.HasPrefix(line, "export TOKEN_SIGMA=") {
			content[i] = exportLine // Replace existing export line
			found = true
			break
		}
	}
	if !found {
		content = append(content, comment, exportLine) // Append new export line if not found
	}

	// Write the updated content back to the .bashrc file
	err = bashrcFile.Truncate(0)
	if err != nil {
		return fmt.Errorf("unable to truncate .bashrc file: %w", err)
	}

	_, err = bashrcFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("unable to seek .bashrc file: %w", err)
	}

	for _, line := range content {
		_, err = bashrcFile.WriteString(line + "\n")
		if err != nil {
			return fmt.Errorf("unable to write to .bashrc file: %w", err)
		}
	}

	log.Info().Msgf("Successfully updated .bashrc with new TOKEN_SIGMA, please use: source %s", bashrcPath)
	return nil
}

// backupBashrc creates a backup of the .bashrc file
func backupBashrc(originalPath, backupPath string) error {
	// Open the original .bashrc file
	originalFile, err := os.Open(originalPath)
	if err != nil {
		return fmt.Errorf("unable to open .bashrc for backup: %w", err)
	}
	defer originalFile.Close()

	// Create the backup file
	backupFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("unable to create backup of .bashrc: %w", err)
	}
	defer backupFile.Close()

	// Copy the contents of .bashrc to the backup file
	_, err = io.Copy(backupFile, originalFile)
	if err != nil {
		return fmt.Errorf("error copying .bashrc content to backup: %w", err)
	}

	log.Info().Msgf("Backup of .bashrc created at: %s", backupPath)
	return nil
}

// generateToken sends a request to generate a token based on the provided username and URL.
func generateToken(username, URL string) error {
	var err error
	type response struct {
		Token string `json:"token"`
	}

	username, err = getUsername(username)
	if err != nil {
		return err
	}
	password, err := getPassword()
	if err != nil {
		return err
	}

	values := map[string]string{
		"username": username,
		"password": password,
	}

	jsonData, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("problem with marshal values into json: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("issue with creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("issue with sending request: %w", err)
	}
	defer resp.Body.Close()

	responseBodyByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("issue with reading response: %w", err)
	}

	responseBody := string(responseBodyByte)

	if resp.StatusCode == 200 {
		var result response
		if err := json.Unmarshal(responseBodyByte, &result); err != nil {
			return fmt.Errorf("cannot unmarshal response: %w", err)
		}
		log.Info().
			Str("httpCode", resp.Status).
			Str("URL", URL).
			Str("response", responseBody).Send()

		log.Info().
			Msgf("export TOKEN_SIGMA=\"%s\"", result.Token)

		// Update the .bashrc file with the new token
		err := updateBashrc(result.Token)
		if err != nil {
			return fmt.Errorf("error updating .bashrc: %w", err)
		}

	} else {
		log.Error().
			Str("httpCode", resp.Status).
			Str("response", responseBody).Send()
		return err
	}
	return nil
}

// AddToken creates a CLI command for utility operations.
func AddToken() *cli.Command {
	return &cli.Command{
		Name:  "get-token",
		Usage: "Generates an authentication token for API access for the specified environment.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "username",
				Aliases: []string{"u"},
				Sources: cli.EnvVars("USER"),
				Usage:   "Specify the username for authentication to generate the token.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			err := generateToken(cmd.String("username"), config.SigmaAdminWebToken)
			if err != nil {
				return err
			}
			return nil
		},
	}
}
