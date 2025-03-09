package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"bytes"
	"path/filepath"
	"encoding/json"
)

// Config structure to hold file paths and settings
type Config struct {
	CommitTemplate string    `json:"commit_template"`
	PRTemplate     string    `json:"pr_template"`
	LLM            LLMConfig `json:"llm"`
}

// expandPath expands the tilde in file paths to the user's home directory
func expandPath(path string) string {
	Log(DEBUG, "Expanding path: %s", path)
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			Log(WARN, "Could not get user home directory: %v", err)
			return path // Return original if we can't get home dir
		}
		expanded := filepath.Join(home, path[2:])
		Log(DEBUG, "Expanded path to: %s", expanded)
		return expanded
	}
	return path
}

// loadConfig reads the configuration file.
func loadConfig(configPath string) (Config, error) {
	Log(INFO, "Loading config from: %s", configPath)
	var config Config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		Log(ERROR, "Failed to read config file: %v", err)
		return config, fmt.Errorf("failed to read config file: %v", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		Log(ERROR, "Failed to parse config file: %v", err)
		return config, fmt.Errorf("failed to parse config file: %v", err)
	}
	
	// Expand paths
	Log(DEBUG, "Expanding template paths")
	config.CommitTemplate = expandPath(config.CommitTemplate)
	config.PRTemplate = expandPath(config.PRTemplate)
	
	// Set default LLM values if not provided
	if config.LLM.Model == "" {
		Log(DEBUG, "Setting default LLM model: gpt-4")
		config.LLM.Model = "gpt-4"
	}
	if config.LLM.Temperature == 0 {
		Log(DEBUG, "Setting default LLM temperature: 0.7")
		config.LLM.Temperature = 0.7
	}
	if config.LLM.MaxTokens == 0 {
		Log(DEBUG, "Setting default LLM max tokens: 1000")
		config.LLM.MaxTokens = 1000
	}
	
	// Try to get API key from environment if not in config
	if config.LLM.APIKey == "" {
		Log(DEBUG, "API key not found in config, checking environment")
		config.LLM.APIKey = os.Getenv("OPENAI_KEY")
		if config.LLM.APIKey == "" {
			Log(WARN, "OPENAI_KEY not found in environment")
		} else {
			Log(DEBUG, "OPENAI_KEY found in environment with length: %d", len(config.LLM.APIKey))
		}
	}
	
	Log(INFO, "Config loaded successfully")
	return config, nil
}

// getStagedDiff retrieves the diff of staged changes.
func getStagedDiff() (string, error) {
	Log(INFO, "Getting staged diff from git")
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		Log(ERROR, "Failed to get staged diff: %v", err)
		return "", fmt.Errorf("failed to get staged diff: %v", err)
	}
	diffSize := len(output)
	Log(DEBUG, "Retrieved staged diff (%d bytes)", diffSize)
	return string(output), nil
}

// createCommitMessage generates a commit message using the template file and LLM.
func createCommitMessage(diff string, templatePath string, llmConfig LLMConfig) (string, error) {
	Log(INFO, "Creating commit message using template: %s", templatePath)
	if diff == "" {
		Log(ERROR, "No changes staged for commit")
		return "", fmt.Errorf("no changes staged. Please stage changes before committing.")
	}

	Log(DEBUG, "Reading commit template file")
	template, err := ioutil.ReadFile(templatePath)
	if err != nil {
		Log(ERROR, "Failed to read commit template: %v", err)
		return "", fmt.Errorf("failed to read commit template: %v", err)
	}

	// Generate commit message using LLM
	Log(INFO, "Generating commit message using LLM model: %s", llmConfig.Model)
	message, err := GenerateCommitMessage(diff, llmConfig, string(template))
	if err != nil {
		Log(ERROR, "LLM generation failed: %v", err)
		return "", fmt.Errorf("LLM generation failed: %v", err)
	}
	
	Log(DEBUG, "Commit message generated successfully (%d chars)", len(message))
	return message, nil
}

// openInVim allows the user to edit the commit message.
func openInVim(filename string) error {
	Log(INFO, "Opening message in vim: %s", filename)
	cmd := exec.Command("vim", filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		Log(ERROR, "Error while editing with vim: %v", err)
	} else {
		Log(DEBUG, "Vim editor closed successfully")
	}
	return err
}

// commitChanges commits using the edited message.
func commitChanges(messageFile string) error {
	Log(INFO, "Committing changes with message file: %s", messageFile)
	cmd := exec.Command("git", "commit", "-F", messageFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		Log(ERROR, "Failed to commit changes: %v", err)
	} else {
		Log(INFO, "Changes committed successfully")
	}
	return err
}

// getCommitMessages retrieves all commit messages between the current branch and the target branch
func getCommitMessages(targetBranch string) (string, error) {
	Log(INFO, "Getting commit messages between current branch and %s", targetBranch)
	// Get current branch name
	cmdBranch := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	currentBranch, err := cmdBranch.Output()
	if err != nil {
		Log(ERROR, "Failed to get current branch: %v", err)
		return "", fmt.Errorf("failed to get current branch: %v", err)
	}
	currentBranchStr := strings.TrimSpace(string(currentBranch))
	Log(DEBUG, "Current branch: %s", currentBranchStr)
	
	// Get commit messages between target branch and current branch
	Log(DEBUG, "Fetching commit messages between %s and %s", targetBranch, currentBranchStr)
	cmd := exec.Command("git", "log", "--pretty=format:%s", fmt.Sprintf("%s..%s", targetBranch, currentBranchStr))
	output, err := cmd.Output()
	if err != nil {
		Log(ERROR, "Failed to get commit messages: %v", err)
		return "", fmt.Errorf("failed to get commit messages: %v", err)
	}
	
	commitCount := strings.Count(string(output), "\n") + 1
	if string(output) == "" {
		commitCount = 0
	}
	Log(INFO, "Retrieved %d commit messages", commitCount)
	return string(output), nil
}

// createPRMessage generates a PR message using the template file, commit messages, and LLM
func createPRMessage(commits string, templatePath string, llmConfig LLMConfig) (string, error) {
	Log(INFO, "Creating PR message using template: %s", templatePath)
	if commits == "" {
		Log(ERROR, "No commits found between branches")
		return "", fmt.Errorf("no commits found between branches. Please make some commits first.")
	}

	Log(DEBUG, "Reading PR template file")
	template, err := ioutil.ReadFile(templatePath)
	if err != nil {
		Log(ERROR, "Failed to read PR template: %v", err)
		return "", fmt.Errorf("failed to read PR template: %v", err)
	}

	// Generate PR message using LLM
	Log(INFO, "Generating PR message using LLM model: %s", llmConfig.Model)
	message, err := GeneratePRMessage(commits, llmConfig, string(template))
	if err != nil {
		Log(ERROR, "LLM generation failed: %v", err)
		return "", fmt.Errorf("LLM generation failed: %v", err)
	}
	
	Log(DEBUG, "PR message generated successfully (%d chars)", len(message))
	return message, nil
}

// createPullRequest creates a PR on GitHub using the gh CLI
func createPullRequest(prMessageFile string, targetBranch string) (string, error) {
	Log(INFO, "Creating pull request to target branch: %s", targetBranch)
	// Check if gh CLI is installed
	if _, err := exec.LookPath("gh"); err != nil {
		Log(ERROR, "GitHub CLI (gh) not found")
		return "", fmt.Errorf("GitHub CLI (gh) not found. Please install it from https://cli.github.com/")
	}
	
	// Get current branch name
	cmdBranch := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	currentBranch, err := cmdBranch.Output()
	if err != nil {
		Log(ERROR, "Failed to get current branch: %v", err)
		return "", fmt.Errorf("failed to get current branch: %v", err)
	}
	currentBranchStr := strings.TrimSpace(string(currentBranch))
	Log(DEBUG, "Current branch: %s", currentBranchStr)
	
	// Push the current branch to remote
	Log(INFO, "Pushing commits to remote...")
	pushCmd := exec.Command("git", "push", "-u", "origin", currentBranchStr)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		Log(ERROR, "Failed to push to remote: %v", err)
		return "", fmt.Errorf("failed to push to remote: %v", err)
	}
	
	// Create PR using gh CLI
	Log(INFO, "Creating PR on GitHub...")
	cmd := exec.Command("gh", "pr", "create", "--base", targetBranch, "--fill", "--body-file", prMessageFile)
	
	// Capture the output to get the PR URL
	output, err := cmd.CombinedOutput()
	if err != nil {
		Log(ERROR, "Failed to create PR: %v\n%s", err, string(output))
		return "", fmt.Errorf("failed to create PR: %v\n%s", err, string(output))
	}
	
	// Extract PR URL from output
	outputStr := string(output)
	
	// Find the URL in the output (usually the last line)
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	var prURL string
	for _, line := range lines {
		if strings.HasPrefix(line, "https://") {
			prURL = strings.TrimSpace(line)
			break
		}
	}
	
	if prURL == "" {
		Log(WARN, "PR created but couldn't extract URL from output")
		return "", fmt.Errorf("PR created but couldn't extract URL from output")
	}
	
	Log(INFO, "PR created successfully: %s", prURL)
	return prURL, nil
}

// loadConfigFromPrioritizedLocations tries to load config from multiple locations in order of priority
func loadConfigFromPrioritizedLocations(customPath string) (Config, error) {
	Log(INFO, "Loading config from prioritized locations")
	// If a custom path is provided, try that first
	if customPath != "" {
		Log(DEBUG, "Custom config path provided: %s", customPath)
		expandedPath := expandPath(customPath)
		config, err := loadConfig(expandedPath)
		if err == nil {
			Log(INFO, "Successfully loaded config from custom path")
			return config, nil
		}
		// If custom path fails, don't fall back - return the error
		Log(ERROR, "Failed to load config from specified path %s: %v", customPath, err)
		return Config{}, fmt.Errorf("failed to load config from specified path %s: %v", customPath, err)
	}

	// List of potential config locations in order of priority
	configLocations := []string{
		".gitscribe_config.json", // Current working directory
	}

	// Add user's home directory location
	home, err := os.UserHomeDir()
	if err == nil {
		homePath := filepath.Join(home, ".gitscribe", ".gitscribe_config.json")
		Log(DEBUG, "Adding home directory config path: %s", homePath)
		configLocations = append(configLocations, homePath)
	} else {
		Log(WARN, "Could not get user home directory: %v", err)
	}

	// Add executable directory location
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		execConfigPath := filepath.Join(execDir, ".gitscribe_config.json")
		Log(DEBUG, "Adding executable directory config path: %s", execConfigPath)
		configLocations = append(configLocations, execConfigPath)
	} else {
		Log(WARN, "Could not get executable path: %v", err)
	}

	// Try each location in order
	Log(DEBUG, "Trying %d potential config locations", len(configLocations))
	var lastErr error
	for _, location := range configLocations {
		Log(DEBUG, "Trying config location: %s", location)
		config, err := loadConfig(location)
		if err == nil {
			Log(INFO, "Successfully loaded config from: %s", location)
			return config, nil
		}
		lastErr = err
		Log(DEBUG, "Failed to load from %s: %v", location, err)
	}

	// If we get here, we couldn't find a config file
	Log(ERROR, "Could not find config file in any standard location")
	return Config{}, fmt.Errorf("could not find config file in any standard location: %v", lastErr)
}