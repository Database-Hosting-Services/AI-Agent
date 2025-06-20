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