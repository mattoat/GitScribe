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
	"runtime"
)

// Config structure to hold file paths and settings
type Config struct {
	CommitTemplate string    `json:"commit_template"`
	PRTemplate     string    `json:"pr_template"`
	LLM            LLMConfig `json:"llm"`
}

// expandPath expands the tilde in file paths to the user's home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Return original if we can't get home dir
		}
		return filepath.Join(home, path[2:])
	}
	return path
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
	
	// Expand paths
	config.CommitTemplate = expandPath(config.CommitTemplate)
	config.PRTemplate = expandPath(config.PRTemplate)
	
	// Set default LLM values if not provided
	if config.LLM.Model == "" {
		config.LLM.Model = "gpt-4"
	}
	if config.LLM.Temperature == 0 {
		config.LLM.Temperature = 0.7
	}
	if config.LLM.MaxTokens == 0 {
		config.LLM.MaxTokens = 1000
	}
	
	// Try to get API key from environment if not in config
	if config.LLM.APIKey == "" {
		config.LLM.APIKey = os.Getenv("OPENAI_KEY")
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

// createCommitMessage generates a commit message using the template file and LLM.
func createCommitMessage(diff string, templatePath string, llmConfig LLMConfig) (string, error) {
	if diff == "" {
		return "", fmt.Errorf("no changes staged. Please stage changes before committing.")
	}

	template, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read commit template: %v", err)
	}

	// Generate commit message using LLM
	message, err := GenerateCommitMessage(diff, llmConfig, string(template))
	if err != nil {
		return "", fmt.Errorf("LLM generation failed: %v", err)
	}
	
	return message, nil
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

// getCommitMessages retrieves all commit messages between the current branch and the target branch
func getCommitMessages(targetBranch string) (string, error) {
	// Get current branch name
	cmdBranch := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	currentBranch, err := cmdBranch.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %v", err)
	}
	
	// Get commit messages between target branch and current branch
	cmd := exec.Command("git", "log", "--pretty=format:%s", fmt.Sprintf("%s..%s", targetBranch, strings.TrimSpace(string(currentBranch))))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit messages: %v", err)
	}
	
	return string(output), nil
}

// createPRMessage generates a PR message using the template file, commit messages, and LLM
func createPRMessage(commits string, templatePath string, llmConfig LLMConfig) (string, error) {
	if commits == "" {
		return "", fmt.Errorf("no commits found between branches. Please make some commits first.")
	}

	template, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read PR template: %v", err)
	}

	// Generate PR message using LLM
	message, err := GeneratePRMessage(commits, llmConfig, string(template))
	if err != nil {
		return "", fmt.Errorf("LLM generation failed: %v", err)
	}
	
	return message, nil
}

// copyToClipboard attempts to copy the contents of a file to the system clipboard
func copyToClipboard(filePath string) error {
	// Read the file content
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	// Try different clipboard commands based on OS
	var cmd *exec.Cmd
	
	// Check if xclip is available (Linux)
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd = exec.Command("xclip", "-selection", "clipboard")
	} else if _, err := exec.LookPath("pbcopy"); err == nil {
		// macOS
		cmd = exec.Command("pbcopy")
	} else if _, err := exec.LookPath("clip"); err == nil {
		// Windows
		cmd = exec.Command("clip")
	} else {
		return fmt.Errorf("no clipboard command found")
	}
	
	cmd.Stdin = bytes.NewReader(content)
	return cmd.Run()
}

// createPullRequest creates a PR on GitHub using the gh CLI
func createPullRequest(prMessageFile string, targetBranch string) (string, error) {
	// Check if gh CLI is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("GitHub CLI (gh) not found. Please install it from https://cli.github.com/")
	}
	
	// Get current branch name
	cmdBranch := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	currentBranch, err := cmdBranch.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %v", err)
	}
	
	// Push the current branch to remote
	fmt.Println("Pushing commits to remote...")
	pushCmd := exec.Command("git", "push", "-u", "origin", strings.TrimSpace(string(currentBranch)))
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to push to remote: %v", err)
	}
	
	// Create PR using gh CLI
	fmt.Println("Creating PR on GitHub...")
	cmd := exec.Command("gh", "pr", "create", "--base", targetBranch, "--fill", "--body-file", prMessageFile)
	
	// Capture the output to get the PR URL
	output, err := cmd.CombinedOutput()
	if err != nil {
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
		return "", fmt.Errorf("PR created but couldn't extract URL from output")
	}
	
	return prURL, nil
}

// openBrowser attempts to open the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // Linux and others
		cmd = exec.Command("xdg-open", url)
	}
	
	return cmd.Start() // Use Start() instead of Run() to not wait for browser to close
}