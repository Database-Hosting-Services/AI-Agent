package RAG

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/pinecone-io/go-pinecone/pinecone"
)

const (
	DEFAULT_TOP_K   = 5
	PROMPT_TEMPLATE = `
You are a database system design expert. Your task is to analyze SQL schemas and user requests to suggest database modifications that follow best practices in system design.

Given the following context use the resources and the instructions to answer the user request:
resources:
%s

CURRENT DATABASE SCHEMA:
%s

The schema is provided in JSON format with the following structure:
{
  "TABLES": {
    "table_name": {
      "COLUMNS": {
        "column_name": {
          "TYPE": "data_type",
          "NULLABLE": true/false,
          "UNIQUE": true/false,
          "DEFAULT": "default_value",
          "CHECKS": [],
          "IS_PRIMARY": true/false,
          "IS_INDEX": true/false,
          "COMMENT": "column description"
        }
      },
      "PRIMARY_KEYS": ["column1", "column2"],
      "FOREIGN_KEYS": [
        {
          "COLUMNS": ["local_column"],
          "FOREIGN_TABLE": "referenced_table",
          "REFERRED_COLUMNS": ["referenced_column"],
          "ON_DELETE": "CASCADE/RESTRICT/SET NULL",
          "ON_UPDATE": "CASCADE/RESTRICT/SET NULL"
        }
      ],
      "CHECKS": [],
      "INDEXES": [["column1"], ["column1", "column2"]],
      "COMMENT": "table description"
    }
  }
}

USER REQUEST:
%s

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
The final schema changes should be headed with the keyword "SCHEMA CHANGES" in the following format:
# SCHEMA CHANGES
{schema_changes}
# END SCHEMA CHANGES
and should be in the json format provided above.
also add a section with headed with the keyword "Schema DDL" with the DDL statements to implement the changes based on old schema(provided above in json format) and new schema(your response).

The schema DDL should be in the following format:
# SCHEMA DDL
{schema_ddl}
# END SCHEMA DDL
with are the DDL statements written in sql.
	`
)

type AgentResponse struct {
	Response      string `json:"response"`
	SchemaChanges string `json:"schema_changes"`
	SchemaDDL     string `json:"schema_ddl"`
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
	prompt := fmt.Sprintf(PROMPT_TEMPLATE, resources, schema, query)

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
	schemaChanges := ""
	schemaDDL := ""

	// Split response into lines for parsing
	lines := strings.Split(responseText, "\n")

	// Extract schema changes between markers
	inSchemaChanges := false
	inSchemaDDL := false
	schemaChangesLines := []string{}
	schemaDDLLines := []string{}

	for _, line := range lines {
		if strings.Contains(line, "# SCHEMA CHANGES") {
			inSchemaChanges = true
			continue
		}
		if strings.Contains(line, "# END SCHEMA CHANGES") {
			inSchemaChanges = false
			continue
		}
		if strings.Contains(line, "# SCHEMA DDL") {
			inSchemaDDL = true
			continue
		}
		if strings.Contains(line, "# END SCHEMA DDL") {
			inSchemaDDL = false
			continue
		}

		if inSchemaChanges {
			schemaChangesLines = append(schemaChangesLines, line)
		}
		if inSchemaDDL {
			schemaDDLLines = append(schemaDDLLines, line)
		}
	}

	// Join the extracted lines
	schemaChanges = strings.TrimSpace(strings.Join(schemaChangesLines, "\n"))
	schemaDDL = strings.TrimSpace(strings.Join(schemaDDLLines, "\n"))

	// Return AgentResponse
	return &AgentResponse{
		Response:      responseText,
		SchemaChanges: schemaChanges,
		SchemaDDL:     schemaDDL,
	}, nil
}
