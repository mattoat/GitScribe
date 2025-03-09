package main

// import (
// 	"fmt"
// 	"os"
// )

// func main() {
// 	// Create a new LLM config which will attempt to load the API key
// 	config := NewLLMConfig()
	
// 	// Print the result
// 	fmt.Println("API Key Test Results:")
// 	fmt.Println("---------------------")
	
// 	// Check environment variable directly
// 	envKey := os.Getenv("OPENAI_KEY")
// 	if envKey != "" {
// 		fmt.Println("✅ OPENAI_KEY environment variable is set")
// 		fmt.Println("   Length:", len(envKey))
// 	} else {
// 		fmt.Println("❌ OPENAI_KEY environment variable is NOT set")
// 	}
	
// 	// Check the config object
// 	if config.APIKey != "" {
// 		fmt.Println("✅ LLMConfig successfully loaded API key")
// 		fmt.Println("   Length:", len(config.APIKey))
// 	} else {
// 		fmt.Println("❌ LLMConfig failed to load API key")
// 	}
// } 