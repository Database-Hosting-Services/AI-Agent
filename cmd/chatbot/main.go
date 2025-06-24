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
	fmt.Println()

	chatbot := RAG.NewDbChatbot()
	if chatbot == nil {
		log.Fatal("Failed to create chatbot")
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		userQuery := scanner.Text()
		if strings.ToLower(userQuery) == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		// Process the query
		response, err := chatbot.QueryChat("database-articles", userQuery, 3) // try to replace the namespace with any invalid namespace and see the out put 
		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
			continue
		}

		// Display the response
		fmt.Println("\nResponse:")
		fmt.Println("---------")
		fmt.Println(response.Response)
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
