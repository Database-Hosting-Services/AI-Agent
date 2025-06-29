package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Database-Hosting-Services/AI-Agent/RAG"
)

func main() {
	fmt.Println("Database Chatbot CLI")
	fmt.Println("====================")
	fmt.Println("Type your database-related questions or 'exit' to quit.")
	fmt.Println("Using the fixed namespace: database-articles")
	fmt.Println()

	// Create RAG configuration from environment variables
	config := &RAG.RAGConfig{
		GeminiAPIKey:         os.Getenv("GEMINI_API_KEY"),
		GeminiModel:          os.Getenv("GEMINI_MODEL"),
		GeminiEmbeddingModel: os.Getenv("GEMINI_EMBEDDING_MODEL"),
		PineconeAPIKey:       os.Getenv("PINECONE_API_KEY"),
		PineconeIndexName:    os.Getenv("PINECONE_INDEX_NAME"),
		PineconeIndexHost:    os.Getenv("PINECONE_INDEX_HOST"),
	}

	// Use the production RAG model, not the test version
	ragModel := RAG.GetRAG(config)
	if ragModel == nil {
		log.Fatal("Failed to initialize RAG model")
	}

	scanner := bufio.NewScanner(os.Stdin)
	// Hard-code the namespace to "database-articles" only
	const namespace = "database-articles"
	const topK = 3

	fmt.Printf("Using fixed namespace: %s\n\n", namespace)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		userInput := scanner.Text()
		if strings.ToLower(userInput) == "exit" {
			fmt.Println("Goodbye!")
			break
		}


		var response RAG.ChatbotResponse
		var err error

		response, err = ragModel.QueryChat(userInput)

		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
			continue
		}

		// Display the response
		fmt.Println("\nResponse:")
		fmt.Println("---------")
		fmt.Println(response.ResponseText)
		fmt.Println()

		// Display sources if available
		if len(response.Sources) > 0 {
			fmt.Println("Sources:")
			fmt.Println("--------")
			for i, source := range response.Sources {
				fmt.Printf("%d. %s\n", i+1, source)
			}
			fmt.Println()
		}
	}
}
