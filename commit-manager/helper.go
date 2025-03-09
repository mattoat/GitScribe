package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
	"path/filepath"
	"encoding/json"
)

// Config structure to hold file paths
type Config struct {
	CommitTemplate string `json:"commit_template"`
	PRTemplate     string `json:"pr_template"`
}

// loadConfig reads the configuration file.
func loadConfig(configPath string) (Config, error) {
	var config Config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %v", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("failed to parse config file: %v", err)
	}
	return config, nil
}

// getStagedDiff retrieves the diff of staged changes.
func getStagedDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %v", err)
	}
	return string(output), nil
}

// createCommitMessage generates a commit message using the template file.
func createCommitMessage(diff string, templatePath string) (string, error) {
	if diff == "" {
		return "No changes staged. Please stage changes before committing.", nil
	}

	template, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read commit template: %v", err)
	}

	return fmt.Sprintf(string(template), diff), nil
}

// openInVim allows the user to edit the commit message.
func openInVim(filename string) error {
	cmd := exec.Command("vim", filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// commitChanges commits using the edited message.
func commitChanges(messageFile string) error {
	cmd := exec.Command("git", "commit", "-F", messageFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	// Load config
	configPath := "config.json"
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	diff, err := getStagedDiff()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Generate commit message using template
	message, err := createCommitMessage(diff, config.CommitTemplate)
	if err != nil {
		fmt.Println("Error generating commit message:", err)
		os.Exit(1)
	}

	// Create a temporary commit message file
	tempFile := fmt.Sprintf("commit_message_%d.txt", time.Now().Unix())
	file, err := os.Create(tempFile)
	if err != nil {
		fmt.Println("Error creating temp file:", err)
		os.Exit(1)
	}
	defer os.Remove(tempFile) // Cleanup temp file after commit

	file.WriteString(message)
	file.Close()

	// Open Vim for the user to edit the message
	if err := openInVim(tempFile); err != nil {
		fmt.Println("Error opening Vim:", err)
		os.Exit(1)
	}

	// Commit changes with the edited message
	if err := commitChanges(tempFile); err != nil {
		fmt.Println("Error committing changes:", err)
		os.Exit(1)
	}

	fmt.Println("Commit successful!")
}
