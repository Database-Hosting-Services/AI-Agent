package RAG_test

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/Database-Hosting-Services/AI-Agent/RAG"
)

func before() {
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

func after() {
	// delete the files schema.json and query.txt
	os.Remove("testIO/schema.json")
	os.Remove("testIO/query.txt")
}
func TestRAG(t *testing.T) {
	before()
	defer after()
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
