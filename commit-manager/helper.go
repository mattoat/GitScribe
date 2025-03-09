package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
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