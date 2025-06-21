package RAG

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/pinecone-io/go-pinecone/pinecone"
)

const (
	DEFAULT_TOP_K   = 5
)

type AgentResponse struct {
	Response      string `json:"response"`
	SchemaChanges string `json:"schema_changes"`
	SchemaDDL     string `json:"schema_ddl"`
}

type RAGmodel interface {
	Embed(text string) ([]float32, error)
	QueryAgent(namespace string, schema string, query string, topK int) (*AgentResponse, error)
	// Upsert(id string, vector []float32, metadata map[string]string) error
}

// implement the RAGmodel interface for the RAGConfig
func (r *RAGConfig) Embed(text string) ([]float32, error) {
	res, err := r.EmbeddingModel.EmbedContent(context.Background(), genai.Text(text))
	if err != nil {
		return nil, err
	}
	return res.Embedding.Values, nil
}

func (r *RAGConfig) Match(namespace string, query string, topK int) ([]*pinecone.ScoredVector, error) {
	// get the embedding of the query
	queryEmbedding, err := r.Embed(query)
	if err != nil {
		return nil, err
	}

	// get the topK results from the vector store
	indexConn, err := r.DbClient.Index(pinecone.NewIndexConnParams{
		Host:      r.IndexHost,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}
	defer indexConn.Close()

	// query the vector store
	results, err := indexConn.QueryByVectorValues(context.Background(), &pinecone.QueryByVectorValuesRequest{
		Vector:          queryEmbedding,
		TopK:            uint32(topK),
		IncludeMetadata: true,
	})
	if err != nil {
		return nil, err
	}

	// return the results
	return results.Matches, nil
}

func (r *RAGConfig) QueryAgent(namespace string, schema string, query string, topK int) (*AgentResponse, error) {
	if topK == 0 {
		topK = DEFAULT_TOP_K
	}

	// get the matches
	matches, err := r.Match(namespace, query, topK)
	if err != nil {
		return nil, err
	}

	// sort the matches by score
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// get the resources
	resources := ""
	for _, match := range matches {
		// fetch the resource by making a get request to the source_url in the metadata
		response, err := http.Get(match.Vector.Metadata.Fields["source_url"].GetStringValue())
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		resources += string(body) + "\n"
	}

	// get the prompt
	prompt := fmt.Sprintf(AGENT_PROMPT_TEMPLATE, resources, schema, query)

	// get the model
	model := r.GenerativeModel

	// get the response
	response, err := model.GenerateContent(context.Background(), genai.Text(prompt))
	if err != nil {
		return nil, err
	}

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
