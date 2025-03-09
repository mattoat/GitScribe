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
	"bufio"
	"regexp"
)

// LLMConfig holds configuration for the OpenAI API
type LLMConfig struct {
	APIKey          string  `json:"api_key"`
	Model           string  `json:"model"`
	Temperature     float64 `json:"temperature"`
	MaxTokens       int     `json:"max_tokens"`
	EnableQuestions bool    `json:"enable_questions"`
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

// QuestionResponse represents a question from the LLM and the user's answer
type QuestionResponse struct {
	Question string
	Answer   string
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
	systemPrompt := fmt.Sprintf(`You are a professional software engineer who has just finished writing code.
	You've staged your changes and are now tasked with writing a commit message. You will be given a git
	diff and a template. Use the git diff to determine what changes have been made in this commit. This is important
	for you to write an accurate and thoughtful commit message. Use the template to generate a commit message. 
	The commit message should be concise and informative. You should not use complicated words if there is a simpler 
	alternative. The people reveiwing your commit message are also professional software engineers, 
	so you can use technical language and do not need to spell out abbreviations such as PR, LLM, FF, etc. 
	The template is a markdown file, but don't include the comments in your response.
	The first line of the commit message should be structured as follows:
	<subdirectory of the repo> <common directory of the file changes>: <brief title of the changes>
	Example: go ingester_worker: Adds implementation for receiving LLM requests
	Example: client dashboard_settings: add LLM settings to UI
	Example: go gql_api: Defines GraphQL API for auth signin
	Example: database/migrations: Adds new migrations for new tables
	Example: client map: fixes bug with map view
	
	Do not include any markdown headers in your response.
	The rest of the commit message should be an informative description of the changes you made.
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

// GeneratePRMessage uses the OpenAI API to generate a PR message based on commit messages
func GeneratePRMessage(commits string, config LLMConfig, template string) (string, error) {
	if config.APIKey == "" {
		return "", fmt.Errorf("OpenAI API key not found. Set the OPENAI_KEY environment variable")
	}

	// Create the system prompt using the template
	systemPrompt := fmt.Sprintf(
	`You are a professional software engineer who has finished a feature branch and is creating a pull request. 
	You will be given a list of commit messages from the branch and a PR template. Use the template to generate a 
	comprehensive PR description. The PR description should clearly explain the changes, their purpose, and any 
	important implementation details.Do not include any other texts about testing, a human who will review 
	your PR message will fill that part out. IMPORTANT: You MUST include the ENTIRE template in your response, 
	including ALL sections at the end. %s Use the following template format for your response:
	%s`, getQuestionsPrompt(config.EnableQuestions), template)

	// Prepare the request
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Here are the commit messages from the branch:\n\n%s", commits)},
	}

	fmt.Println("Generating PR description based on commit messages...")
	
	// First API call to generate PR message or ask questions
	response, err := makeOpenAIRequest(messages, config)
	if err != nil {
		return "", err
	}

	// Check if questions are enabled and if the response contains questions
	questionResponses, hasQuestions := extractQuestions(response)
	if hasQuestions && config.EnableQuestions {
		fmt.Printf("The AI has %d questions to help create a better PR description.\n", len(questionResponses))
		
		// Get answers from the user
		questionResponses = askUserQuestions(questionResponses)
		
		// Check if any questions were answered
		anyAnswered := false
		for _, q := range questionResponses {
			if q.Answer != "" {
				anyAnswered = true
				break
			}
		}
		
		// Only make a second API call if at least one question was answered
		if anyAnswered {
			// Create a new messages array that includes all previous context
			// The OpenAI API doesn't maintain context between separate API calls
			// so we need to include all messages in the new request
			newMessages := []ChatMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: fmt.Sprintf("Here are the commit messages from the branch:\n\n%s", commits)},
				{Role: "assistant", Content: "I need some additional information to write a better PR description."},
			}
			
			// Add each question and its answer as separate messages to maintain the conversation flow
			for _, qa := range questionResponses {
				if qa.Answer != "" {
					newMessages = append(newMessages, 
						ChatMessage{Role: "assistant", Content: qa.Question},
						ChatMessage{Role: "user", Content: qa.Answer},
					)
				}
			}
			
			// Add a final prompt to generate the PR description
			newMessages = append(newMessages, ChatMessage{
				Role: "user", 
				Content: "Now that you have this additional information, please generate a comprehensive PR description using the template provided earlier.",
			})
			
			fmt.Println("Generating final PR description with your additional context...")
			
			// Make a second API call with the additional context
			response, err = makeOpenAIRequest(newMessages, config)
			if err != nil {
				return "", err
			}
		} else {
			fmt.Println("Proceeding with the initial PR description since no questions were answered.")
			// Try to extract a PR description from the initial response
			response = extractPRDescription(response)
		}
	}

	// Return the generated PR message
	return strings.TrimSpace(response), nil
}

// getQuestionsPrompt returns the prompt for questions based on whether the feature is enabled
func getQuestionsPrompt(enableQuestions bool) string {
	if enableQuestions {
		return `
	If you need additional information to write a more informative PR description, you can ask up to 3 questions.
	To ask questions, respond with a JSON object in the following format:
	{"questions": ["question 1", "question 2", "question 3"]}
	
	Only ask questions if you genuinely need more context to write a better PR description. Don't ask questions in most cases.
	`
	}
	return ""
}

// makeOpenAIRequest makes a request to the OpenAI API and returns the response content
func makeOpenAIRequest(messages []ChatMessage, config LLMConfig) (string, error) {
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

	return chatResponse.Choices[0].Message.Content, nil
}

// extractQuestions checks if the response contains questions and extracts them
func extractQuestions(response string) ([]QuestionResponse, bool) {
	// Try to parse the entire response as JSON first
	var questionsObj struct {
		Questions []string `json:"questions"`
	}
	
	// If the entire response is valid JSON with questions
	if err := json.Unmarshal([]byte(response), &questionsObj); err == nil && len(questionsObj.Questions) > 0 {
		Log(DEBUG, "Found questions in complete JSON response")
		return convertToQuestionResponses(questionsObj.Questions), true
	}
	
	// If not, try to find JSON object within text using regex
	re := regexp.MustCompile(`\{[\s\n]*"questions"[\s\n]*:[\s\n]*\[.*?\][\s\n]*\}`)
	match := re.FindString(response)
	
	if match == "" {
		Log(DEBUG, "No questions JSON found in response")
		return nil, false
	}
	
	Log(DEBUG, "Found potential questions JSON: %s", match)
	
	// Try to parse the extracted JSON
	if err := json.Unmarshal([]byte(match), &questionsObj); err != nil {
		Log(WARN, "Failed to parse questions JSON: %v", err)
		return nil, false
	}
	
	// Skip if no questions were found
	if len(questionsObj.Questions) == 0 {
		Log(DEBUG, "Questions array was empty")
		return nil, false
	}
	
	return convertToQuestionResponses(questionsObj.Questions), true
}

// Helper function to convert string questions to QuestionResponse objects
func convertToQuestionResponses(questions []string) []QuestionResponse {
	// Limit the number of questions to 3
	maxQuestions := 3
	if len(questions) > maxQuestions {
		Log(INFO, "Limiting questions to %d (received %d)", maxQuestions, len(questions))
		questions = questions[:maxQuestions]
	}
	
	// Convert to QuestionResponse objects
	questionResponses := make([]QuestionResponse, len(questions))
	for i, q := range questions {
		questionResponses[i] = QuestionResponse{
			Question: q,
			Answer:   "", // Will be filled in later
		}
	}
	
	return questionResponses
}

// askUserQuestions presents questions to the user and collects answers
func askUserQuestions(questions []QuestionResponse) []QuestionResponse {
	fmt.Println("\nThe AI needs some additional information to write a better PR description:")
	fmt.Println("(Press Enter with no text to skip a question)")
	
	reader := bufio.NewReader(os.Stdin)
	
	for i := range questions {
		fmt.Printf("\nQuestion %d: %s\n", i+1, questions[i].Question)
		fmt.Print("Your answer: ")
		
		answer, _ := reader.ReadString('\n')
		questions[i].Answer = strings.TrimSpace(answer)
		
		// If the user enters 'skip all' or 'skipall', skip remaining questions
		if strings.ToLower(questions[i].Answer) == "skip all" || strings.ToLower(questions[i].Answer) == "skipall" {
			fmt.Println("Skipping remaining questions...")
			// Set empty answers for remaining questions
			for j := i + 1; j < len(questions); j++ {
				questions[j].Answer = ""
			}
			break
		}
	}
	
	// Count how many questions were answered
	answeredCount := 0
	for _, q := range questions {
		if q.Answer != "" {
			answeredCount++
		}
	}
	
	if answeredCount == 0 {
		fmt.Println("\nNo questions were answered. Proceeding with original context only.")
	} else if answeredCount < len(questions) {
		fmt.Printf("\n%d out of %d questions answered. Proceeding with partial additional context.\n", answeredCount, len(questions))
	} else {
		fmt.Println("\nAll questions answered. Proceeding with full additional context.")
	}
	
	return questions
}

// formatQuestionsAndAnswers formats the questions and answers for the API request
func formatQuestionsAndAnswers(qas []QuestionResponse) string {
	var sb strings.Builder
	
	sb.WriteString("Here are my answers to your questions:\n\n")
	
	for i, qa := range qas {
		sb.WriteString(fmt.Sprintf("Question %d: %s\n", i+1, qa.Question))
		sb.WriteString(fmt.Sprintf("Answer: %s\n\n", qa.Answer))
	}
	
	return sb.String()
}

// extractPRDescription attempts to extract a PR description from a response that contains questions
func extractPRDescription(response string) string {
	// If the response only contains questions, return an empty string
	if strings.TrimSpace(response) == "" || strings.HasPrefix(strings.TrimSpace(response), "{\"questions\":") {
		return ""
	}
	
	// Check if the response contains a JSON object with questions
	startIdx := strings.Index(response, "{\"questions\":")
	if startIdx == -1 {
		// No questions found, return the entire response
		return response
	}
	
	// Find the end of the JSON object
	endIdx := -1
	braceCount := 0
	for i := startIdx; i < len(response); i++ {
		if response[i] == '{' {
			braceCount++
		} else if response[i] == '}' {
			braceCount--
			if braceCount == 0 {
				endIdx = i
				break
			}
		}
	}
	
	if endIdx == -1 {
		// Could not find the end of the JSON object, return the entire response
		return response
	}
	
	// Return everything before the questions and after the questions
	beforeQuestions := strings.TrimSpace(response[:startIdx])
	afterQuestions := strings.TrimSpace(response[endIdx+1:])
	
	if beforeQuestions != "" && afterQuestions != "" {
		return beforeQuestions + "\n\n" + afterQuestions
	} else if beforeQuestions != "" {
		return beforeQuestions
	} else if afterQuestions != "" {
		return afterQuestions
	}
	
	// If we couldn't extract anything, return an empty string
	return ""
} 