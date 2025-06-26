package RAG

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type ChatbotResponse struct {
	Response string   `json:"response"`
	Sources  []string `json:"sources,omitempty"`
}

type DbChatbot struct {
	RagSystem RAGmodel
}

func NewDbChatbot() *DbChatbot {
	return &DbChatbot{
		RagSystem: GetRAGTest(),
	}
}

func (c *DbChatbot) QueryChat(namespace string, userQuery string, topK int) (*ChatbotResponse, error) {
	if topK == 0 {
		topK = DEFAULT_TOP_K
	}

	log.Printf("Processing database chatbot query: %s", userQuery)

	// Use the RAG system to find relevant documents
	startTime := time.Now()
	matches, err := c.RagSystem.Match(namespace, userQuery, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to find relevant database information: %v", err)
	}
	log.Printf("INFO: Vector matching took %f seconds", time.Since(startTime).Seconds())

	// Extract source URLs and build a context from the matches
	resources := ""
	sources := []string{}
	sourceMap := make(map[string]bool)

	// Process each match to extract source URLs and content
	for _, match := range matches {
		resources += "--------------------------------\n"

		if url, ok := match.Vector.Metadata.Fields["source_url"]; ok {
			sourceURL := strings.Trim(url.GetStringValue(), "\"\n \t")
			if !sourceMap[sourceURL] {
				sourceMap[sourceURL] = true
				sources = append(sources, sourceURL)
				resources += fmt.Sprintf("Source: %s\n\n", sourceURL)
			}
		}

		if content, ok := match.Vector.Metadata.Fields["content"]; ok {
			resources += strings.Trim(content.GetStringValue(), "\"\n \t") + "\n"
		}
	}
	resources += "--------------------------------\n"

	responseStartTime := time.Now()
	contextualQuery := fmt.Sprintf("The user is asking about database concepts. Here is relevant context: %s\n\nUser query: %s", resources, userQuery)
	fmt.Println("Contextual Query: \n", contextualQuery) // keep this to track the data fetched from the vector database

	agentResponse, err := c.RagSystem.QueryAgent(namespace, "", contextualQuery, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %v", err)
	}
	log.Printf("INFO: Response generation took %f seconds", time.Since(responseStartTime).Seconds())
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %v", err)
	}
	log.Printf("INFO: Response generation took %f seconds", time.Since(responseStartTime).Seconds())

	responseText := agentResponse.Response

	// Clean up any markdown code blocks or special formatting if needed
	responseText = strings.ReplaceAll(responseText, "```json", "")
	responseText = strings.ReplaceAll(responseText, "```sql", "")
	responseText = strings.ReplaceAll(responseText, "```", "")

	return &ChatbotResponse{
		Response: responseText,
		Sources:  sources,
	}, nil
}
