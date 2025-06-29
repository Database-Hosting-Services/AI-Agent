package RAG

import (

)

type Schema struct {
	Tables map[string]TableInfo `json:"TABLES"`
}

type TableInfo struct {
	Columns     map[string]ColumnInfo `json:"COLUMNS"`
	PrimaryKeys []string              `json:"PRIMARY_KEYS"`
	ForeignKeys []ForeignKeyInfo      `json:"FOREIGN_KEYS"`
	Checks      []interface{}         `json:"CHECKS"`
	Indexes     [][]string            `json:"INDEXES"`
	Comment     *string               `json:"COMMENT"`
}

type ColumnInfo struct {
	Type      string        `json:"TYPE"`
	Nullable  *bool         `json:"NULLABLE"`
	Unique    *bool         `json:"UNIQUE"`
	Default   interface{}   `json:"DEFAULT"`
	Checks    []interface{} `json:"CHECKS"`
	IsPrimary bool          `json:"IS_PRIMARY"`
	IsIndex   bool          `json:"IS_INDEX"`
	Comment   *string       `json:"COMMENT"`
}

type ForeignKeyInfo struct {
	Columns         []string `json:"COLUMNS"`
	ForeignTable    string   `json:"FOREIGN_TABLE"`
	ReferredColumns []string `json:"REFERRED_COLUMNS"`
	OnDelete        *string  `json:"ON_DELETE"`
	OnUpdate        *string  `json:"ON_UPDATE"`
}

type Analytics struct {
	MonthlyAnalytics map[string]Analytic `json:"MONTHLY_ANALYTICS"`
}

type Analytic struct {
	DiskUsage    float64 `json:"DISK_USAGE"`
	CPUUsage     float64 `json:"CPU_USAGE"`
	MemoryUsage  float64 `json:"MEMORY_USAGE"`
	NetworkUsage float64 `json:"NETWORK_USAGE"`
	Costs        float64 `json:"COSTS"`
}

type ChatbotResponse struct {
	ResponseText string   `json:"response_text"`
	Sources      []string `json:"sources"`
}