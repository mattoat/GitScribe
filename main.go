package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Define command-line flags
	generatePR := flag.Bool("pr", false, "Generate a PR message and prepare for PR creation")
	targetBranch := flag.String("target", "master", "Target branch for PR (default: master)")
	skipCreate := flag.Bool("skip-create", false, "Skip PR creation on GitHub (only generate message)")
	configPath := flag.String("config", "", "Path to config file (default: search in standard locations)")
	dryRun := flag.Bool("dry-run", false, "Generate message but don't commit or create PR")
	logLevelFlag := flag.String("log-level", "none", "Set logging level (debug, info, warn, error, none)")
	amendCommit := flag.Bool("amend", false, "Amend the last commit with a new message (includes both last commit and any staged changes)")
	flag.Parse()

	// Set log level based on flag
	switch strings.ToLower(*logLevelFlag) {
	case "debug":
		SetLogLevel(DEBUG)
	case "info":
		SetLogLevel(INFO)
	case "warn", "warning":
		SetLogLevel(WARN)
	case "error":
		SetLogLevel(ERROR)
	case "none", "":
		// Set to a level higher than any defined log level to suppress all logs
		SetLogLevel(ERROR + 1)
	default:
		// Default to no logging
		SetLogLevel(ERROR + 1)
	}

	Log(INFO, "Starting application")
	Log(DEBUG, "Command-line flags: pr=%v, target=%s, skip-create=%v, config=%s, dry-run=%v, log-level=%s, amend=%v",
		*generatePR, *targetBranch, *skipCreate, *configPath, *dryRun, *logLevelFlag, *amendCommit)

	// Load config from appropriate location
	Log(INFO, "Loading configuration")
	config, err := loadConfigFromPrioritizedLocations(*configPath)
	if err != nil {
		Log(ERROR, "Failed to load config: %v", err)
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	var message string

	if *generatePR {
		Log(INFO, "Generating PR message")
		// Generate PR message
		commits, err := getCommitMessages(*targetBranch)
		if err != nil {
			Log(ERROR, "Failed to get commit messages: %v", err)
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		message, err = createPRMessage(commits, config.PRTemplate, config.LLM)
		if err != nil {
			Log(ERROR, "Failed to create PR message: %v", err)
			fmt.Println("Error generating PR message:", err)
			os.Exit(1)
		}
	} else if *amendCommit {
		Log(INFO, "Generating message for amending commit")
		// Get the last commit diff and any staged changes
		diff, err := getLastCommitDiff()
		if err != nil {
			Log(ERROR, "Failed to get last commit diff: %v", err)
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		message, err = createCommitMessage(diff, config.CommitTemplate, config.LLM)
		if err != nil {
			Log(ERROR, "Failed to create commit message for amend: %v", err)
			fmt.Println("Error generating commit message:", err)
			os.Exit(1)
		}
	} else {
		Log(INFO, "Generating commit message")
		// Generate commit message (existing functionality)
		diff, err := getStagedDiff()
		if err != nil {
			Log(ERROR, "Failed to get staged diff: %v", err)
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		message, err = createCommitMessage(diff, config.CommitTemplate, config.LLM)
		if err != nil {
			Log(ERROR, "Failed to create commit message: %v", err)
			fmt.Println("Error generating commit message:", err)
			os.Exit(1)
		}
	}

	if *dryRun {
		Log(INFO, "Dry run mode - displaying message and exiting")
		fmt.Println("=== Generated Message (Dry Run) ===")
		fmt.Println(message)
		fmt.Println("==================================")
		return
	}

	// Create a temporary message file
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("git_message_%d.txt", time.Now().Unix()))
	Log(DEBUG, "Creating temporary message file: %s", tempFile)
	file, err := os.Create(tempFile)
	if err != nil {
		Log(ERROR, "Failed to create temporary file: %v", err)
		fmt.Println("Error creating temp file:", err)
		os.Exit(1)
	}

	// Only remove the temp file if we're not creating a PR or if it's a commit message
	if !*generatePR || *skipCreate {
		Log(DEBUG, "Setting up deferred removal of temporary file")
		defer os.Remove(tempFile)
	}

	Log(DEBUG, "Writing message to temporary file (%d bytes)", len(message))
	if _, err := file.WriteString(message); err != nil {
		Log(ERROR, "Failed to write to temporary file: %v", err)
		fmt.Println("Error writing to temp file:", err)
		os.Exit(1)
	}
	if err := file.Close(); err != nil {
		Log(ERROR, "Failed to close temporary file: %v", err)
		fmt.Println("Error closing temp file:", err)
		os.Exit(1)
	}

	// Open editor for the user to edit the message
	Log(INFO, "Opening editor for user to edit message")
	if err := openInVim(tempFile); err != nil {
		Log(ERROR, "Failed to open editor: %v", err)
		fmt.Println("Error opening editor:", err)
		os.Exit(1)
	}

	if *generatePR {
		if !*skipCreate {
			// Create PR using GitHub CLI
			Log(INFO, "Creating PR on GitHub")
			fmt.Println("Creating PR on GitHub...")
			prURL, err := createPullRequest(tempFile, *targetBranch)
			if err != nil {
				Log(ERROR, "Failed to create PR: %v", err)
				fmt.Println("Error creating PR:", err)
				os.Exit(1)
			}
			Log(INFO, "PR created successfully: %s", prURL)
			fmt.Println("PR created successfully!")
			fmt.Println("PR URL:", prURL)
		} else {
			// For PR messages without creation, just display the file path
			Log(INFO, "Skipping PR creation, message saved to file")
			fmt.Printf("PR message saved to: %s\n", tempFile)
			fmt.Println("You can use this message when creating a PR on GitHub.")
		}
	} else if *amendCommit {
		// For amending commits
		Log(INFO, "Amending commit")
		fmt.Println("Amending commit with both the last commit changes and any staged changes...")
		if err := amendCommitWithMessage(tempFile); err != nil {
			Log(ERROR, "Failed to amend commit: %v", err)
			fmt.Println("Error amending commit:", err)
			os.Exit(1)
		}
		Log(INFO, "Commit amended successfully")
		fmt.Println("Commit amended successfully!")
	} else {
		// For commit messages, proceed with commit
		Log(INFO, "Committing changes")
		if err := commitChanges(tempFile); err != nil {
			Log(ERROR, "Failed to commit changes: %v", err)
			fmt.Println("Error committing changes:", err)
			os.Exit(1)
		}
		Log(INFO, "Commit completed successfully")
		fmt.Println("Commit successful!")
	}

	Log(INFO, "Application completed successfully")
}
