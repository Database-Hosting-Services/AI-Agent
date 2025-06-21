package RAG

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/pinecone-io/go-pinecone/pinecone"
	"google.golang.org/api/option"
)

type RAGConfig struct {
	DbClient       *pinecone.Client
	GeminiClient   *genai.Client
	IndexHost      string
	EmbeddingModel *genai.EmbeddingModel
	GenerativeModel *genai.GenerativeModel
}

var Rag RAGmodel

func init() {
	// read the dotenv file
	// Get the current working directory
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("Failed to get file path")
	}
	dir := filepath.Dir(filename)
	// get the parent directory
	dir = filepath.Dir(dir)
	// get the parent directory
	targetFiledir := filepath.Join(dir, ".env")

	if err := godotenv.Load(targetFiledir); err != nil {
		log.Fatal(err)
	}

	// connect to gemini API
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	// Initialize Gemini client
	geminiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(geminiAPIKey))
	if err != nil {
		log.Fatalf("Failed to initialize Gemini client: %v", err)
	}

	// get the embedding model
	embeddingModel := geminiClient.EmbeddingModel(os.Getenv("GEMINI_EMBEDDING_MODEL"))

	if res, err := embeddingModel.EmbedContent(context.Background(), genai.Text("Hi, Gemini")); err != nil || res == nil {
		if err == nil {
			err = errors.New("model returned a nil result")
		}
		geminiClient.Close()
		log.Fatalf("connection error while trying to connect to gemini: %s\n ERROR: %s", os.Getenv("GEMINI_EMBEDDING_MODEL"), err.Error())
	}

	// get the generative model
	generativeModel := geminiClient.GenerativeModel(os.Getenv("GEMINI_MODEL"))
	if res, err := generativeModel.GenerateContent(context.Background(), genai.Text("Hi, Gemini")); err != nil || res == nil {
		if err == nil {
			err = errors.New("model returned a nil result")
		}
		geminiClient.Close()
		log.Fatalf("connection error while trying to connect to gemini: %s\n ERROR: %s", os.Getenv("GEMINI_MODEL"), err.Error())
	}

	// connect to the Vector database
	pineconeAPIKey := os.Getenv("PINECONE_API_KEY")
	if pineconeAPIKey == "" {
		geminiClient.Close()
		log.Fatal("PINECONE_API_KEY environment variable is required")
	}

	// Initialize Pinecone client
	pineconeClient, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: pineconeAPIKey,
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
	indexName := os.Getenv("PINECONE_INDEX_NAME")
	if indexName == "" {
		indexName = "knowledge-index" // default index name
		log.Printf("Using default index name: %s\n", indexName)
	}

	// Get index host from environment variable
	indexHost := os.Getenv("PINECONE_INDEX_HOST")
	if indexHost == "" {
		// Alternatively, you can describe the index to get the host
		idx, err := pineconeClient.DescribeIndex(context.Background(), indexName)
		if err != nil {
			geminiClient.Close()
			log.Fatalf("Failed to describe index or PINECONE_INDEX_HOST not set: %v", err)
		}
		indexHost = idx.Host
		log.Printf("Retrieved index host: %s\n", indexHost)
		log.Printf("Index dimension: %d\n", idx.Dimension)
	}

	Rag = &RAGConfig{
		GeminiClient: geminiClient,
		DbClient:     pineconeClient,
		IndexHost:    indexHost,
		EmbeddingModel: embeddingModel,
		GenerativeModel: generativeModel,
	}
}
