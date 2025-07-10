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

// TableColumn represents a database column with its properties
type TableColumn struct {
	TableName              string  `db:"table_name" json:"TableName"`
	ColumnName             string  `db:"column_name" json:"ColumnName"`
	DataType               string  `db:"data_type" json:"DataType"`
	IsNullable             bool    `db:"is_nullable" json:"IsNullable"`
	ColumnDefault          *string `db:"column_default" json:"ColumnDefault"`
	CharacterMaximumLength *int    `db:"character_maximum_length" json:"CharacterMaximumLength"`
	NumericPrecision       *int    `db:"numeric_precision" json:"NumericPrecision"`
	NumericScale           *int    `db:"numeric_scale" json:"NumericScale"`
	OrdinalPosition        int     `db:"ordinal_position" json:"OrdinalPosition"`
}

// ConstraintInfo represents database constraints
type ConstraintInfo struct {
	TableName         string  `db:"table_name" json:"TableName"`
	ConstraintName    string  `db:"constraint_name" json:"ConstraintName"`
	ConstraintType    string  `db:"constraint_type" json:"ConstraintType"`
	ColumnName        *string `db:"column_name" json:"ColumnName"`
	ForeignTableName  *string `db:"foreign_table_name" json:"ForeignTableName"`
	ForeignColumnName *string `db:"foreign_column_name" json:"ForeignColumnName"`
	CheckClause       *string `db:"check_clause" json:"CheckClause"`
	OrdinalPosition   *int    `db:"ordinal_position" json:"OrdinalPosition"`
}

// IndexInfo represents database indexes
type IndexInfo struct {
	TableName  string `db:"table_name" json:"TableName"`
	IndexName  string `db:"index_name" json:"IndexName"`
	ColumnName string `db:"column_name" json:"ColumnName"`
	IsUnique   bool   `db:"is_unique" json:"IsUnique"`
	IndexType  string `db:"index_type" json:"IndexType"`
	IsPrimary  bool   `db:"is_primary" json:"IsPrimary"`
}

type Table struct {
	TableName   string           `db:"table_name" json:"TableName"`
	Columns     []TableColumn    `db:"columns" json:"Columns"`
	Constraints []ConstraintInfo `db:"constraints" json:"Constraints"`
	Indexes     []IndexInfo      `db:"indexes" json:"Indexes"`
}