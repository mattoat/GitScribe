package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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

	// Generate commit message using template and LLM
	message, err := createCommitMessage(diff, config.CommitTemplate, config.LLM)
	if err != nil {
		fmt.Println("Error generating commit message:", err)
		os.Exit(1)
	}

	// Create a temporary commit message file
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("commit_message_%d.txt", time.Now().Unix()))
	file, err := os.Create(tempFile)
	if err != nil {
		fmt.Println("Error creating temp file:", err)
		os.Exit(1)
	}
	defer os.Remove(tempFile) // Cleanup temp file after commit

	if _, err := file.WriteString(message); err != nil {
		fmt.Println("Error writing to temp file:", err)
		os.Exit(1)
	}
	if err := file.Close(); err != nil {
		fmt.Println("Error closing temp file:", err)
		os.Exit(1)
	}

	// Open editor for the user to edit the message
	if err := openInVim(tempFile); err != nil {
		fmt.Println("Error opening editor:", err)
		os.Exit(1)
	}

	// Commit changes with the edited message
	if err := commitChanges(tempFile); err != nil {
		fmt.Println("Error committing changes:", err)
		os.Exit(1)
	}

	fmt.Println("Commit successful!")
} 