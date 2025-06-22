package RAG_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/Database-Hosting-Services/AI-Agent/RAG"
)

func beforeAgent() {
	// create a new file schema.json and query.txt in the current directory
	os.Create("testIO/schema.json")
	os.Create("testIO/query.txt")
	// write the schema to the file schema.json
	schema := RAG.Schema{
		Tables: map[string]RAG.TableInfo{
			"users": {
				Columns: map[string]RAG.ColumnInfo{
					"id":       {Type: "INTEGER", IsPrimary: true},
					"name":     {Type: "TEXT"},
					"email":    {Type: "TEXT", IsIndex: true},
					"password": {Type: "TEXT"},
				},
				PrimaryKeys: []string{"id"},
			},
		},
	}
	// write the schema to the file schema.json
	text, err := json.Marshal(schema)
	if err != nil {
		log.Fatalf("Failed to marshal schema: %v", err)
	}
	os.WriteFile("testIO/schema.json", text, 0644)

	// write the query to the file query.txt
	query := `
	add a new column to the users table for the username and
	add a new table called orders that contains the information about orderes made by the users
	you can add more tables if it's necessary to the relations between the users and the orders
	but don't add more tables than necessary
	`
	os.WriteFile("testIO/query.txt", []byte(query), 0644)
}

func afterAgent() {
	// delete the files schema.json and query.txt
	os.Remove("testIO/schema.json")
	os.Remove("testIO/query.txt")
}

func beforeReport() {
	beforeAgent()
	// create a new file analytics.json in the current directory
	os.Create("testIO/analytics.json")
	// write the analytics to the file analytics.json
	analytics := RAG.Analytics{
		MonthlyAnalytics: map[string]RAG.Analytic{
			"2025-01": {
				DiskUsage:    100.0,
				CPUUsage:     100.0,
				MemoryUsage:  100.0,
				NetworkUsage: 100.0,
				Costs:        100.0,
			},
			"2025-02": {
				DiskUsage:    100.0,
				CPUUsage:     100.0,
				MemoryUsage:  100.0,
				NetworkUsage: 100.0,
				Costs:        100.0,
			},
		},
	}
	// write the analytics to the file analytics.json
	text, err := json.Marshal(analytics)
	if err != nil {
		log.Fatalf("Failed to marshal analytics: %v", err)
	}
	os.WriteFile("testIO/analytics.json", text, 0644)
}

func afterReport() {
	afterAgent()
	// delete the file analytics.json
	os.Remove("testIO/analytics.json")
}

func TestRAG(t *testing.T) {
	beforeAgent()
	defer afterAgent()
	rag := RAG.Rag
	// read the schema from the file schema.json
	schema, err := os.ReadFile("testIO/schema.json")
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	// read the query from the file query.txt
	query, err := os.ReadFile("testIO/query.txt")
	if err != nil {
		log.Fatalf("Failed to read query file: %v", err)
	}

	// query the agent
	response, err := rag.QueryAgent("schemas-json", string(schema), string(query), 5)
	if err != nil {
		log.Fatalf("Failed to query agent: %v", err)
	}

	// save the full response to response.txt
	// save the schema changes to schema_changes.json
	// save the schema DDL to schema_ddl.sql

	// save the full response to response.txt
	os.WriteFile("testIO/response.md", []byte(response.Response), 0644)

	// save the schema changes to schema_changes.json
	os.WriteFile("testIO/schema_changes.json", []byte(response.SchemaChanges), 0644)

	// save the schema DDL to schema_ddl.sql
	os.WriteFile("testIO/schema_ddl.sql", []byte(response.SchemaDDL), 0644)
}

func TestEmbed(t *testing.T) {
	beforeAgent()
	defer afterAgent()
	rag := RAG.Rag
	// read the query from the file query.txt
	query, err := os.ReadFile("testIO/query.txt")
	if err != nil {
		log.Fatalf("Failed to read query file: %v", err)
	}
	// embed the query
	embedding, err := rag.Embed(string(query))
	if err != nil {
		log.Fatalf("Failed to embed query: %v", err)
	}
	// save the embedding to the file embedding.txt
	// output the embedding between each element with a comma
	str := ""
	for _, element := range embedding {
		str += fmt.Sprintf("%f,", element)
	}
	os.WriteFile("testIO/embedding.txt", []byte(str), 0644)
}

func TestMatch(t *testing.T) {
	beforeAgent()
	defer afterAgent()
	rag := RAG.Rag
	// read the query from the file query.txt
	query, err := os.ReadFile("testIO/query.txt")
	if err != nil {
		log.Fatalf("Failed to read query file: %v", err)
	}

	matches, err := rag.Match("schemas-json", string(query), 5)
	if err != nil {
		log.Fatalf("Failed to match query: %v", err)
	}
	str := ""
	for _, match := range matches {
		str += fmt.Sprintf("Match: %s, Score: %f\n", match.Vector.Id, match.Score)
	}
	os.WriteFile("testIO/matches.txt", []byte(str), 0644)

}

func TestMatchWithRest(t *testing.T) {
	beforeAgent()
	defer afterAgent()
	rag := RAG.Rag
	// read the query from the file query.txt
	query, err := os.ReadFile("testIO/query.txt")
	if err != nil {
		log.Fatalf("Failed to read query file: %v", err)
	}
	// embed the query
	queryEmbedding, err := rag.Embed(string(query))
	if err != nil {
		log.Fatalf("Failed to embed query: %v", err)
	}
	// use the rest api to match the query
	body := map[string]interface{}{
		"vector": queryEmbedding,
		"topK":   10,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Fatalf("Failed to marshal body: %v", err)
	}

	requesrURL := "https://" + os.Getenv("PINECONE_INDEX_HOST") + "/query"
	request, err := http.NewRequest("POST", requesrURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	request.Header.Set("Api-Key", os.Getenv("PINECONE_API_KEY"))
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Fatalf("Failed to make request: %v", err)
	}
	defer response.Body.Close()

	// save the response to the file response.txt
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}
	os.WriteFile("testIO/response.txt", responseBody, 0644)
}

func TestReport(t *testing.T) {
	beforeReport()
	defer afterReport()
	rag := RAG.Rag
	// read the analytics from the file analytics.json
	analytics, err := os.ReadFile("testIO/analytics.json")
	if err != nil {
		log.Fatalf("Failed to read analytics file: %v", err)
	}
	// read the schema from the file schema.json
	schema, err := os.ReadFile("testIO/schema.json")
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}
	// generate the report
	report, err := rag.Report(string(analytics), string(schema))
	if err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}
	// save the report to the file report.md
	os.WriteFile("testIO/report.md", []byte(report), 0644)
}
