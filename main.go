package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Define command-line flags
	generatePR := flag.Bool("pr", false, "Generate a PR message and prepare for PR creation")
	targetBranch := flag.String("target", "master", "Target branch for PR (default: master)")
	skipCreate := flag.Bool("skip-create", false, "Skip PR creation on GitHub (only generate message)")
	configPath := flag.String("config", "", "Path to config file (default: search in standard locations)")
	dryRun := flag.Bool("dry-run", false, "Generate message but don't commit or create PR")
	flag.Parse()

	// Load config from appropriate location
	config, err := loadConfigFromPrioritizedLocations(*configPath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	var message string

	if *generatePR {
		// Generate PR message
		commits, err := getCommitMessages(*targetBranch)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		message, err = createPRMessage(commits, config.PRTemplate, config.LLM)
		if err != nil {
			fmt.Println("Error generating PR message:", err)
			os.Exit(1)
		}
	} else {
		// Generate commit message (existing functionality)
		diff, err := getStagedDiff()
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		message, err = createCommitMessage(diff, config.CommitTemplate, config.LLM)
		if err != nil {
			fmt.Println("Error generating commit message:", err)
			os.Exit(1)
		}
	}

	if *dryRun {
		fmt.Println("=== Generated Message (Dry Run) ===")
		fmt.Println(message)
		fmt.Println("==================================")
		return
	}

	// Create a temporary message file
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("git_message_%d.txt", time.Now().Unix()))
	file, err := os.Create(tempFile)
	if err != nil {
		fmt.Println("Error creating temp file:", err)
		os.Exit(1)
	}
	
	// Only remove the temp file if we're not creating a PR or if it's a commit message
	if !*generatePR || *skipCreate {
		defer os.Remove(tempFile)
	}

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

	if *generatePR {
		if !*skipCreate {
			// Create PR using GitHub CLI
			fmt.Println("Creating PR on GitHub...")
			prURL, err := createPullRequest(tempFile, *targetBranch)
			if err != nil {
				fmt.Println("Error creating PR:", err)
				os.Exit(1)
			}
			fmt.Println("PR created successfully!")
			fmt.Println("PR URL:", prURL)
		} else {
			// For PR messages without creation, just display the file path
			fmt.Printf("PR message saved to: %s\n", tempFile)
			fmt.Println("You can use this message when creating a PR on GitHub.")
			
			// Copy to clipboard if possible
			if err := copyToClipboard(tempFile); err != nil {
				fmt.Println("Note: Could not copy to clipboard:", err)
			} else {
				fmt.Println("PR message copied to clipboard!")
			}
		}
	} else {
		// For commit messages, proceed with commit
		if err := commitChanges(tempFile); err != nil {
			fmt.Println("Error committing changes:", err)
			os.Exit(1)
		}
		fmt.Println("Commit successful!")
	}
} 