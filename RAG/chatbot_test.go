package RAG_test

// import (
// 	"encoding/json"
// 	"log"
// 	"os"
// 	"testing"

// 	"github.com/Database-Hosting-Services/AI-Agent/RAG"
// )

// func TestChatbot(t *testing.T) {
// 	// Create test directory if it doesn't exist
// 	os.MkdirAll("testIO", 0755)

// 	// Create a chatbot instance
// 	chatbot := RAG.NewDbChatbot()
// 	if chatbot == nil {
// 		t.Fatal("Failed to create chatbot")
// 	}

// 	// Sample query about databases
// 	query := "What are the best practices for indexing in PostgreSQL?"

// 	// Query the chatbot
// 	response, err := chatbot.QueryChat("database-docs", query, 3)
// 	if err != nil {
// 		t.Fatalf("Failed to query chatbot: %v", err)
// 	}

// 	// Validate response
// 	if response.Response == "" {
// 		t.Error("Empty response received")
// 	}

// 	// Save the response to file for manual inspection
// 	responseJSON, err := json.MarshalIndent(response, "", "  ")
// 	if err != nil {
// 		t.Fatalf("Failed to marshal response: %v", err)
// 	}

// 	err = os.WriteFile("testIO/chatbot_response.json", responseJSON, 0644)
// 	if err != nil {
// 		t.Fatalf("Failed to write response to file: %v", err)
// 	}

// 	// Also save the plain text response
// 	err = os.WriteFile("testIO/chatbot_response.txt", []byte(response.Response), 0644)
// 	if err != nil {
// 		t.Fatalf("Failed to write response text to file: %v", err)
// 	}

// 	log.Printf("Chatbot response saved to testIO/chatbot_response.json and testIO/chatbot_response.txt")
// }
