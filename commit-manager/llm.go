package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"github.com/joho/godotenv"
	"strings"
	"os"
)

// LLMConfig holds configuration for the OpenAI API
type LLMConfig struct {
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// ChatMessage represents a message in the OpenAI chat format
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the request body for OpenAI chat completions API
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

// ChatResponse represents the response from OpenAI chat completions API
type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewLLMConfig creates a new LLM configuration
func NewLLMConfig() LLMConfig {
	// Default values
	config := LLMConfig{
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
	}
	// First try to get API key directly from environment
	config.APIKey = os.Getenv("OPENAI_KEY")
	
	// If not found, try loading from .env file as fallback
	if config.APIKey == "" {
		if err := godotenv.Load(); err == nil {
			// Successfully loaded .env file, try again
			config.APIKey = os.Getenv("OPENAI_KEY")
		} else {
			// Print a helpful message about the missing API key
			fmt.Println("Note: Could not load .env file:", err)
		}
	}
	
	// Debug output to verify the API key status
	if config.APIKey == "" {
		fmt.Println("Warning: OPENAI_KEY environment variable not found")
		fmt.Println("Make sure it's set in your environment or .env file")
	} else {
		fmt.Println("OPENAI_KEY found with length:", len(config.APIKey))
	}
	
	return config
}

// GenerateCommitMessage uses the OpenAI API to generate a commit message based on the diff
func GenerateCommitMessage(diff string, config LLMConfig, template string) (string, error) {
	if config.APIKey == "" {
		return "", fmt.Errorf("OpenAI API key not found. Set the OPENAI_KEY environment variable")
	}

	// Create the system prompt using the template
	systemPrompt := fmt.Sprintf(`You are a professional software engineer who has just finished writing code. You've staged your changes and
	are now tasked with writing a commit message. You will be given a git diff and a template. Use the template to generate a commit message. The commit message should be concise and informative.
	Use the following template format for your response:
	%s`, template)

	// Prepare the request
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Here is the git diff:\n\n%s", diff)},
	}

	requestBody := ChatRequest{
		Model:       config.Model,
		Messages:    messages,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make the API request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var chatResponse ChatResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Check for API errors
	if chatResponse.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResponse.Error.Message)
	}

	if len(chatResponse.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	// Return the generated commit message
	return strings.TrimSpace(chatResponse.Choices[0].Message.Content), nil
} 