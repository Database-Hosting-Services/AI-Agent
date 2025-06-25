package RAG

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/pinecone-io/go-pinecone/v4/pinecone"
	"google.golang.org/api/option"
)

const (
	DEFAULT_TOP_K = 5
)

type AgentResponse struct {
	Response      string `json:"response"`
	SchemaChanges string `json:"schema_changes"`
	SchemaDDL     string `json:"schema_ddl"`
}

type RAGmodel interface {
	Embed(text string) ([]float32, error)
	Match(namespace string, query string, topK int) ([]*pinecone.ScoredVector, error)
	QueryAgent(namespace string, schema string, query string, topK int) (*AgentResponse, error)
	Report(analytics string, schema string) (string, error)
	// Upsert(id string, vector []float32, metadata map[string]string) error
}

func GetRAGTest() RAGmodel { // this is for testing purposes only
	return rag
}

func GetRAG(config *RAGConfig) RAGmodel {
	// connect to gemini API
	if config.GeminiAPIKey == "" {
		log.Fatal("Gemini API key is required")
	}

	// Initialize Gemini client
	geminiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(config.GeminiAPIKey))
	if err != nil {
		log.Fatalf("Failed to initialize Gemini client: %v", err)
	}

	// get the embedding model
	embeddingModel := geminiClient.EmbeddingModel(config.GeminiEmbeddingModel)

	if res, err := embeddingModel.EmbedContent(context.Background(), genai.Text("Hi, Gemini")); err != nil || res == nil {
		if err == nil {
			err = errors.New("model returned a nil result")
		}
		geminiClient.Close()
		log.Fatalf("connection error while trying to connect to gemini: %s\n ERROR: %s", config.GeminiEmbeddingModel, err.Error())
	}

	// get the generative model
	generativeModel := geminiClient.GenerativeModel(config.GeminiModel)
	if res, err := generativeModel.GenerateContent(context.Background(), genai.Text("Hi, Gemini")); err != nil || res == nil {
		if err == nil {
			err = errors.New("model returned a nil result")
		}
		geminiClient.Close()
		log.Fatalf("connection error while trying to connect to gemini: %s\n ERROR: %s", config.GeminiModel, err.Error())
	}

	// connect to the Vector database
	if config.PineconeAPIKey == "" {
		geminiClient.Close()
		log.Fatal("Pinecone API key is required")
	}

	// Initialize Pinecone client
	pineconeClient, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: config.PineconeAPIKey,
	})

	if err != nil {
		geminiClient.Close()
		log.Fatalf("Failed to initialize Pinecone client: %v", err)
	}

	if _, err := pineconeClient.ListIndexes(context.Background()); err != nil {
		geminiClient.Close()
		log.Fatalf("connection error while trying to connect to pinecone ERROR: %s", err.Error())
	}

	// Get index name from environment variable or use default
	if config.PineconeIndexName == "" {
		config.PineconeIndexName = "knowledge-index" // default index name
		log.Printf("Using default index name: %s\n", config.PineconeIndexName)
	}

	// Get index host from environment variable
	if config.PineconeIndexHost == "" {
		// Alternatively, you can describe the index to get the host
		idx, err := pineconeClient.DescribeIndex(context.Background(), config.PineconeIndexName)
		if err != nil {
			geminiClient.Close()
			log.Fatalf("Failed to describe index or PINECONE_INDEX_HOST not set: %v", err)
		}
		config.PineconeIndexHost = idx.Host
		log.Printf("Retrieved index host: %s\n", config.PineconeIndexHost)
		log.Printf("Index dimension: %d\n", idx.Dimension)
	}

	// get the topK results from the vector store
	indexConn, err := pineconeClient.Index(pinecone.NewIndexConnParams{
		Host:      config.PineconeIndexHost,
		Namespace: "schemas-json",
	})
	if err != nil {
		log.Printf("ERROR: Failed to connect to index: %v", err)
		geminiClient.Close()
		log.Fatalf("Failed to connect to index: %v", err)
	}

	// check the index stats
	_, err = indexConn.DescribeIndexStats(context.Background()) // this would take some time to complete handshake and stuff XD
	if err != nil {
		geminiClient.Close()
		log.Fatalf("Failed to describe index stats: %v", err)
	}

	rag = &RAGPineconeGemini{
		GeminiClient:    geminiClient,
		DbClient:        pineconeClient,
		IndexConn:       indexConn,
		IndexHost:       config.PineconeIndexHost,
		EmbeddingModel:  embeddingModel,
		GenerativeModel: generativeModel,
	}

	return rag
}

// implement the RAGmodel interface for the RAGConfig
func (r *RAGPineconeGemini) Embed(text string) ([]float32, error) {
	// start a timer
	startTime := time.Now()
	res, err := r.EmbeddingModel.EmbedContent(context.Background(), genai.Text(text))
	if err != nil {
		return nil, err
	}
	log.Printf("INFO: embedding the query took ==> %f seconds", time.Since(startTime).Seconds())
	return res.Embedding.Values, nil
}

func (r *RAGPineconeGemini) Match(namespace string, query string, topK int) ([]*pinecone.ScoredVector, error) {
	// switch the namespace to the correct namespace
	conn := r.IndexConn.WithNamespace(namespace)
	topK += 5 // add 5 to the topK to get more results to replace the missing ones
	// get the embedding of the query
	queryEmbedding, err := r.Embed(query)
	if err != nil {
		log.Printf("ERROR: Failed to generate embedding: %v", err)
		return nil, err
	}

	// start a timer
	startTime := time.Now()
	// query the vector store
	results, err := conn.QueryByVectorValues(context.Background(), &pinecone.QueryByVectorValuesRequest{
		Vector:          queryEmbedding,
		TopK:            uint32(topK),
		IncludeMetadata: true,
		IncludeValues:   false,
	})
	if err != nil {
		log.Printf("ERROR: Pinecone query failed: %v", err)
		return nil, err
	}
	log.Printf("INFO: querying the vector store took ==> %f seconds", time.Since(startTime).Seconds())
	// return the results
	return results.Matches, nil
}

// QueryAgent queries the agent with the given namespace, schema, query, and topK
// this is the main function that will be used to query in agent mode and get the response
func (r *RAGPineconeGemini) QueryAgent(namespace string, schema string, query string, topK int) (*AgentResponse, error) {
	if topK == 0 {
		topK = DEFAULT_TOP_K
	}

	// Check index statistics for debugging
	// if err := r.checkIndexStats(namespace); err != nil {
	// 	log.Printf("Warning: Could not check index stats: %v", err)
	// }

	// get the matches
	matches, err := r.Match(namespace, query, topK)
	if err != nil {
		return nil, err
	}
	// start a timer
	startTime := time.Now()
	// get the resources
	resourcesChan := make(chan string, topK)
	r.fetchResourcesConcurrently(resourcesChan, matches, topK)
	resources := ""
	for resource := range resourcesChan {
		resources += "--------------------------------\n"
		resources += resource + "\n"
	}
	resources += "--------------------------------\n"
	log.Printf("INFO: fetching the resources took ==> %f seconds", time.Since(startTime).Seconds())
	// get the prompt
	prompt := fmt.Sprintf(AGENT_PROMPT_TEMPLATE, resources, schema, query)

	// get the model
	model := r.GenerativeModel
	// start a timer
	startTime = time.Now()
	// get the response
	response, err := model.GenerateContent(context.Background(), genai.Text(prompt))
	if err != nil {
		return nil, err
	}
	log.Printf("INFO: generating the response took ==> %f seconds", time.Since(startTime).Seconds())
	// concatenate the response
	responseText := ""
	for _, part := range response.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			responseText += string(textPart)
		}
	}

	// extract the schema changes by looking for the keyword "SCHEMA CHANGES" from the responseText
	// Regex patterns to match code blocks
	jsonRe := regexp.MustCompile("(?s)```json\\s*(.*?)```")
	sqlRe := regexp.MustCompile("(?s)```sql\\s*(.*?)```")

	// Extract code blocks
	jsonMatch := jsonRe.FindStringSubmatch(responseText)
	sqlMatch := sqlRe.FindStringSubmatch(responseText)
	// Join the extracted lines
	schemaChanges := strings.TrimSpace(jsonMatch[1])
	schemaDDL := strings.TrimSpace(sqlMatch[1])

	// Return AgentResponse
	return &AgentResponse{
		Response:      responseText,
		SchemaChanges: schemaChanges,
		SchemaDDL:     schemaDDL,
	}, nil
}

// generate a report to a project manager based on the analytics of there database
// the report should be in a markdown format
func (r *RAGPineconeGemini) Report(analytics string, schema string) (string, error) {
	// get the prompt
	prompt := fmt.Sprintf(REPORT_PROMPT_TEMPLATE, "resources: none", analytics, schema)

	// get the model
	model := r.GenerativeModel

	// start a timer
	startTime := time.Now()
	// get the response
	response, err := model.GenerateContent(context.Background(), genai.Text(prompt))
	if err != nil {
		return "", err
	}
	log.Printf("INFO: generating the report took ==> %f seconds", time.Since(startTime).Seconds())
	// concatenate the response
	responseText := ""
	for _, part := range response.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			responseText += string(textPart)
		}
	}
	return responseText, nil
}

func (r *RAGPineconeGemini) fetchResourcesConcurrently(resources chan string, matches []*pinecone.ScoredVector, topK int) {
	// sort the matches by score
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var fetchedCount int

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, match := range matches {
		// Check if we already have enough resources
		mu.Lock()
		if fetchedCount >= topK {
			mu.Unlock()
			break
		}
		mu.Unlock()

		wg.Add(1)
		go func(match *pinecone.ScoredVector) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
				// Check again if we still need more resources
				mu.Lock()
				if fetchedCount >= topK {
					mu.Unlock()
					return
				}
				mu.Unlock()

				// fetch the resource by making a get request to the source_url in the metadata
				url := strings.Trim(match.Vector.Metadata.Fields["source_url"].GetStringValue(), "\"\n \t")
				response, err := client.Get(url)
				if err != nil {
					log.Printf("Warning: Failed to fetch resource from %s: %v", url, err)
					return
				}
				defer response.Body.Close()

				if response.StatusCode != http.StatusOK {
					log.Printf("Warning: HTTP %d for resource %s", response.StatusCode, url)
					return
				}

				body, err := io.ReadAll(response.Body)
				if err != nil {
					log.Printf("Warning: Failed to read resource body: %v", err)
					return
				}

				// Try to send the resource, but check count first
				mu.Lock()
				if fetchedCount < topK {
					select {
					case resources <- string(body):
						fetchedCount++
						log.Printf("INFO: Successfully fetched resource %d/%d", fetchedCount, topK)
					case <-ctx.Done():
					}
				}
				mu.Unlock()
			}
		}(match)
	}

	// Wait for all goroutines to complete or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("INFO: All goroutines completed")
	case <-ctx.Done():
		log.Printf("INFO: Timeout reached, stopping resource fetching")
	}

	// Close the resources channel to signal completion
	close(resources)
}

// CheckIndexStats checks the statistics of the Pinecone index
func (r *RAGPineconeGemini) checkIndexStats(namespace string) error {
	log.Printf("=== CHECKING INDEX STATISTICS ===")

	// Get index description
	indexName := os.Getenv("PINECONE_INDEX_NAME")
	if indexName == "" {
		indexName = "knowledge-index"
	}

	idx, err := r.DbClient.DescribeIndex(context.Background(), indexName)
	if err != nil {
		log.Printf("ERROR: Failed to describe index: %v", err)
		return err
	}

	log.Printf("Index name: %s", idx.Name)
	log.Printf("Index dimension: %d", idx.Dimension)
	log.Printf("Index metric: %s", idx.Metric)
	log.Printf("Index host: %s", idx.Host)
	log.Printf("Index status: %s", idx.Status.State)

	// Connect to index to get stats
	indexConn, err := r.DbClient.Index(pinecone.NewIndexConnParams{
		Host:      r.IndexHost,
		Namespace: namespace,
	})
	if err != nil {
		log.Printf("ERROR: Failed to connect to index for stats: %v", err)
		return err
	}
	defer indexConn.Close()

	// Get index statistics
	stats, err := indexConn.DescribeIndexStats(context.Background())
	if err != nil {
		log.Printf("ERROR: Failed to get index stats: %v", err)
		return err
	}

	log.Printf("Total vector count: %d", stats.TotalVectorCount)
	log.Printf("Index fullness: %f", stats.IndexFullness)

	if stats.Namespaces != nil {
		log.Printf("Available namespaces:")
		for ns, nsStats := range stats.Namespaces {
			log.Printf("  Namespace '%s': %d vectors", ns, nsStats.VectorCount)
		}

		// Check if our specific namespace exists
		if nsStats, exists := stats.Namespaces[namespace]; exists {
			log.Printf("Target namespace '%s' has %d vectors", namespace, nsStats.VectorCount)
		} else {
			log.Printf("WARNING: Target namespace '%s' does not exist in index!", namespace)
		}
	}

	log.Printf("=== END INDEX STATISTICS ===")
	return nil
}