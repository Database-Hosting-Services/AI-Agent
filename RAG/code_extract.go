package RAG

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// CodeBlock represents a code block extracted from markdown
type CodeBlock struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

// JSONBlock represents a JSON code block with parsed data
type JSONBlock struct {
	RawCode    string      `json:"raw_code"`
	ParsedJSON interface{} `json:"parsed_json,omitempty"`
	Language   string      `json:"language"`
	Error      string      `json:"error,omitempty"`
}

// SQLBlock represents a SQL code block with query type
type SQLBlock struct {
	Code      string `json:"code"`
	Language  string `json:"language"`
	QueryType string `json:"query_type"`
}

// ExtractedSegments contains all extracted code segments
type ExtractedSegments struct {
	JSONBlocks     []JSONBlock `json:"json_blocks"`
	SQLBlocks      []SQLBlock  `json:"sql_blocks"`
	AllCodeBlocks  []CodeBlock `json:"all_code_blocks"`
}

// CodeExtractor handles extraction of code segments from markdown
type CodeExtractor struct {
	codeBlockPattern *regexp.Regexp
	inlineCodePattern *regexp.Regexp
}

// NewCodeExtractor creates a new CodeExtractor instance
func NewCodeExtractor() *CodeExtractor {
	return &CodeExtractor{
		codeBlockPattern:  regexp.MustCompile(`(?s)` + "```(\\w+)?\\n(.*?)\\n```"),
		inlineCodePattern: regexp.MustCompile("`([^`]+)`"),
	}
}

// ExtractCodeBlocks extracts all code blocks from markdown text
func (ce *CodeExtractor) ExtractCodeBlocks(markdownText string) []CodeBlock {
	matches := ce.codeBlockPattern.FindAllStringSubmatch(markdownText, -1)
	var codeBlocks []CodeBlock

	for _, match := range matches {
		language := "unknown"
		if len(match[1]) > 0 {
			language = strings.ToLower(match[1])
		}
		
		codeBlocks = append(codeBlocks, CodeBlock{
			Language: language,
			Code:     strings.TrimSpace(match[2]),
		})
	}

	return codeBlocks
}

// ExtractJSONBlocks extracts and parses JSON code blocks
func (ce *CodeExtractor) ExtractJSONBlocks(markdownText string) []JSONBlock {
	codeBlocks := ce.ExtractCodeBlocks(markdownText)
	var jsonBlocks []JSONBlock

	jsonLanguages := map[string]bool{
		"json":       true,
		"javascript": true,
		"js":         true,
	}

	for _, block := range codeBlocks {
		if jsonLanguages[block.Language] {
			jsonBlock := JSONBlock{
				RawCode:  block.Code,
				Language: block.Language,
			}

			var parsedJSON interface{}
			if err := json.Unmarshal([]byte(block.Code), &parsedJSON); err != nil {
				if ce.looksLikeJSON(block.Code) {
					jsonBlock.Error = fmt.Sprintf("Invalid JSON syntax: %v", err)
				} else {
					continue // Skip if it doesn't look like JSON
				}
			} else {
				jsonBlock.ParsedJSON = parsedJSON
			}

			jsonBlocks = append(jsonBlocks, jsonBlock)
		}
	}

	return jsonBlocks
}

// ExtractSQLBlocks extracts SQL code blocks
func (ce *CodeExtractor) ExtractSQLBlocks(markdownText string) []SQLBlock {
	codeBlocks := ce.ExtractCodeBlocks(markdownText)
	var sqlBlocks []SQLBlock

	sqlLanguages := map[string]bool{
		"sql":        true,
		"mysql":      true,
		"postgresql": true,
		"sqlite":     true,
		"plsql":      true,
	}

	for _, block := range codeBlocks {
		if sqlLanguages[block.Language] {
			sqlBlocks = append(sqlBlocks, SQLBlock{
				Code:      block.Code,
				Language:  block.Language,
				QueryType: ce.identifySQLType(block.Code),
			})
		}
	}

	return sqlBlocks
}

// ExtractAllSegments extracts both JSON and SQL segments
func (ce *CodeExtractor) ExtractAllSegments(markdownText string) ExtractedSegments {
	return ExtractedSegments{
		JSONBlocks:    ce.ExtractJSONBlocks(markdownText),
		SQLBlocks:     ce.ExtractSQLBlocks(markdownText),
		AllCodeBlocks: ce.ExtractCodeBlocks(markdownText),
	}
}

// looksLikeJSON checks if code looks like JSON based on basic patterns
func (ce *CodeExtractor) looksLikeJSON(code string) bool {
	code = strings.TrimSpace(code)
	return (strings.HasPrefix(code, "{") && strings.HasSuffix(code, "}")) ||
		   (strings.HasPrefix(code, "[") && strings.HasSuffix(code, "]"))
}

// identifySQLType identifies the type of SQL query
func (ce *CodeExtractor) identifySQLType(sqlCode string) string {
	sqlUpper := strings.ToUpper(strings.TrimSpace(sqlCode))
	
	switch {
	case strings.HasPrefix(sqlUpper, "SELECT"):
		return "SELECT"
	case strings.HasPrefix(sqlUpper, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(sqlUpper, "UPDATE"):
		return "UPDATE"
	case strings.HasPrefix(sqlUpper, "DELETE"):
		return "DELETE"
	case strings.HasPrefix(sqlUpper, "CREATE"):
		return "CREATE"
	case strings.HasPrefix(sqlUpper, "ALTER"):
		return "ALTER"
	case strings.HasPrefix(sqlUpper, "DROP"):
		return "DROP"
	default:
		return "OTHER"
	}
}

// Example usage and testing
func main() {
	// Example markdown text with JSON and SQL blocks
	sampleMarkdown := `
Here's an example of how to use the API:

` + "```json" + `
{
    "name": "John Doe",
    "age": 30,
    "email": "john@example.com",
    "preferences": {
        "theme": "dark",
        "notifications": true
    }
}
` + "```" + `

And here's a SQL query to fetch user data:

` + "```sql" + `
SELECT u.id, u.name, u.email, p.theme, p.notifications
FROM users u
LEFT JOIN preferences p ON u.id = p.user_id
WHERE u.age > 25
ORDER BY u.name;
` + "```" + `

You can also use this configuration:

` + "```javascript" + `
{
    "database": {
        "host": "localhost",
        "port": 5432,
        "name": "myapp"
    }
}
` + "```" + `

Another SQL example:

` + "```mysql" + `
INSERT INTO users (name, email, age) 
VALUES ('Jane Smith', 'jane@example.com', 28);
` + "```" + `
	`

	extractor := NewCodeExtractor()
	results := extractor.ExtractAllSegments(sampleMarkdown)

	fmt.Println("=== JSON Blocks ===")
	for i, jsonBlock := range results.JSONBlocks {
		fmt.Printf("\nJSON Block %d:\n", i+1)
		fmt.Printf("Language: %s\n", jsonBlock.Language)
		fmt.Printf("Raw Code:\n%s\n", jsonBlock.RawCode)
		
		if jsonBlock.ParsedJSON != nil {
			if jsonBytes, err := json.MarshalIndent(jsonBlock.ParsedJSON, "", "  "); err == nil {
				fmt.Printf("Parsed JSON:\n%s\n", string(jsonBytes))
			}
		}
		
		if jsonBlock.Error != "" {
			fmt.Printf("Error: %s\n", jsonBlock.Error)
		}
	}

	fmt.Println("\n\n=== SQL Blocks ===")
	for i, sqlBlock := range results.SQLBlocks {
		fmt.Printf("\nSQL Block %d:\n", i+1)
		fmt.Printf("Language: %s\n", sqlBlock.Language)
		fmt.Printf("Query Type: %s\n", sqlBlock.QueryType)
		fmt.Printf("Code:\n%s\n", sqlBlock.Code)
	}

	fmt.Println("\n\n=== All Code Blocks ===")
	for i, block := range results.AllCodeBlocks {
		fmt.Printf("\nCode Block %d:\n", i+1)
		fmt.Printf("Language: %s\n", block.Language)
		
		code := block.Code
		if len(code) > 100 {
			code = code[:100] + "..."
		}
		fmt.Printf("Code: %s\n", code)
	}
}

// Additional utility functions for advanced usage

// ExtractInlineCode extracts inline code segments (text wrapped in backticks)
func (ce *CodeExtractor) ExtractInlineCode(markdownText string) []string {
	matches := ce.inlineCodePattern.FindAllStringSubmatch(markdownText, -1)
	var inlineCode []string

	for _, match := range matches {
		inlineCode = append(inlineCode, match[1])
	}

	return inlineCode
}

// FilterCodeBlocksByLanguage filters code blocks by specific language
func (ce *CodeExtractor) FilterCodeBlocksByLanguage(codeBlocks []CodeBlock, languages []string) []CodeBlock {
	languageMap := make(map[string]bool)
	for _, lang := range languages {
		languageMap[strings.ToLower(lang)] = true
	}

	var filtered []CodeBlock
	for _, block := range codeBlocks {
		if languageMap[block.Language] {
			filtered = append(filtered, block)
		}
	}

	return filtered
}

// ValidateJSON validates if a JSON string is valid
func (ce *CodeExtractor) ValidateJSON(jsonStr string) error {
	var result interface{}
	return json.Unmarshal([]byte(jsonStr), &result)
}

// PrettyPrintJSON formats JSON string with indentation
func (ce *CodeExtractor) PrettyPrintJSON(jsonStr string) (string, error) {
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", err
	}
	
	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}
	
	return string(formatted), nil
}