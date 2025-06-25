package RAG_test

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Database-Hosting-Services/AI-Agent/RAG"
)

// ChatSession represents a simulated conversation with the chatbot
type ChatSession struct {
	UserID    string             `json:"user_id"`
	Timestamp string             `json:"timestamp"`
	Queries   []string           `json:"queries"`
	Responses []string           `json:"responses"`
	Sources   [][]string         `json:"sources"`
	Metrics   map[string]float64 `json:"metrics"`
}

func TestChatbotConversation(t *testing.T) {
	// Create test directory if it doesn't exist
	os.MkdirAll("testIO", 0755)

	// Create a chatbot instance
	chatbot := RAG.NewDbChatbot()
	if chatbot == nil {
		t.Fatal("Failed to create chatbot")
	}

	// Sample conversation about database design and optimization
	queries := []string{
		"What are the best practices for designing a PostgreSQL database schema?",
		"How should I optimize queries for a heavily-used e-commerce database?",
		"What indexing strategies would you recommend for tables with millions of rows?",
		"How do I handle database migrations with zero downtime?",
	}

	// Start a new chat session
	session := ChatSession{
		UserID:    "test-user-123",
		Timestamp: time.Now().Format(time.RFC3339),
		Queries:   queries,
		Responses: make([]string, 0, len(queries)),
		Sources:   make([][]string, 0, len(queries)),
		Metrics:   make(map[string]float64),
	}

	// Process each query in the conversation
	totalTime := time.Duration(0)
	for i, query := range queries {
		fmt.Printf("[Test] Processing query %d: %s\n", i+1, query)

		// Time the query processing
		start := time.Now()

		// Query the chatbot
		response, err := chatbot.QueryChat("database-articles", query, 3)
		if err != nil {
			t.Fatalf("Failed to query chatbot for query %d: %v", i+1, err)
		}

		// Record metrics
		queryTime := time.Since(start)
		totalTime += queryTime

		// Validate response
		if response.Response == "" {
			t.Errorf("Empty response received for query %d", i+1)
		}

		// Add to session
		session.Responses = append(session.Responses, response.Response)
		session.Sources = append(session.Sources, response.Sources)

		fmt.Printf("[Test] Response received in %v\n", queryTime)
	}

	// Record overall metrics
	session.Metrics["avg_response_time_ms"] = float64(totalTime.Milliseconds()) / float64(len(queries))
	session.Metrics["total_conversation_time_ms"] = float64(totalTime.Milliseconds())

	// Save the conversation to file for manual inspection
	sessionJSON, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal session: %v", err)
	}

	err = os.WriteFile("testIO/chatbot_conversation.json", sessionJSON, 0644)
	if err != nil {
		t.Fatalf("Failed to write session to file: %v", err)
	}

	log.Printf("Chatbot conversation saved to testIO/chatbot_conversation.json")
	log.Printf("Average response time: %.2f ms", session.Metrics["avg_response_time_ms"])
}
