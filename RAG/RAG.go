package RAG

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/google/generative-ai-go/genai"
	"github.com/pinecone-io/go-pinecone/pinecone"
)

const (
	DEFAULT_TOP_K   = 5
	PROMPT_TEMPLATE = `
You are a database system design expert. Your task is to analyze SQL schemas and user requests to suggest database modifications that follow best practices in system design.

Given the following context use the resources and the instructions to answer the user request:
resources:
{resources}

CURRENT DATABASE SCHEMA:
{schema}

USER REQUEST:
{request}

Please analyze the request and provide recommendations that follow these database system design principles:

1. **Normalization**: Ensure proper normalization (1NF, 2NF, 3NF) to eliminate redundancy
2. **Referential Integrity**: Maintain proper foreign key relationships and constraints
3. **Indexing Strategy**: Suggest appropriate indexes for performance optimization
4. **Data Types**: Use appropriate and efficient data types
5. **Naming Conventions**: Follow consistent and meaningful naming patterns
6. **Scalability**: Consider future growth and performance implications
7. **Security**: Include appropriate access controls and data protection measures
8. **ACID Properties**: Ensure atomicity, consistency, isolation, and durability

Your response should include:
- Analysis of the current schema structure
- Identification of any existing design issues
- Specific SQL DDL statements to implement the requested changes
- Explanation of how the changes improve the system design
- Any potential risks or considerations for the modification

Format your response with clear sections and provide executable SQL when applicable.
the final schema changes should be headed with the keyword "SCHEMA CHANGES"
and should be in the following format:

	`
)


type AgentResponse struct {
	Response string
	SchemaChanges string
}

type RAGmodel interface {
	Embed(text string) ([]float32, error)
	Query(query string, topK int) ([]string, error)
	Upsert(id string, vector []float32, metadata map[string]string) error
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

func (r *RAGConfig) QueryAgent(namespace string, schema string, query string, topK int) ([]*pinecone.ScoredVector, error) {
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
	prompt := fmt.Sprintf(PROMPT_TEMPLATE, resources, schema, query)

	// get the model
	model := r.GenerativeModel

	// get the response
	response, err := model.GenerateContent(context.Background(), genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	
}