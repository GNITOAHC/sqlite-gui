// Package database defines a minimal interface for relational database access.
package database

import (
	"context"
	"database/sql"
	"errors"
)

var ErrNotConnected = errors.New("database not connected")

type (
	ForeignKeyAction string
	Row              map[string]any
	Key              map[string]any
)

type ForeignKey struct {
	RefTable string
	FromCol  string
	ToCol    string
	OnDelete ForeignKeyAction
	OnUpdate ForeignKeyAction
}

const (
	ForeignKeyActionNoAction   ForeignKeyAction = "NO ACTION"
	ForeignKeyActionSetNull    ForeignKeyAction = "SET NULL"
	ForeignKeyActionSetDefault ForeignKeyAction = "SET DEFAULT"
	ForeignKeyActionRestrict   ForeignKeyAction = "RESTRICT"
	ForeignKeyActionCascade    ForeignKeyAction = "CASCADE"
)

type Column struct {
	Name    string
	Type    string
	NotNull bool // Whether the column can be null
	Default sql.NullString

	PrimaryKey      bool // Column is part of the primary key
	PrimaryKeyIndex int  // 1-based position within a composite primary key (0 if not part of the PK)

	ForeignKeys []ForeignKey
}

type ColumnDef struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	NotNull    bool   `json:"notNull"`
	Default    *string
	PrimaryKey bool
}

type Database interface {
	// Connect establishes a connection to the database with the given connection string.
	Connect(ctx context.Context, conn string) error

	// Close closes the connection to the database.
	Close() error

	// Ping verifies the connection is still alive.
	Ping(ctx context.Context) error

	// GetTables retrieves all table names from the database.
	Tables(ctx context.Context) ([]string, error)

	// GetColumns retrieves the column names for a specific table from the database.
	Columns(ctx context.Context, table string) ([]Column, error)

	// CreateTable creates a table with the provided columns. Supports composite primary keys via ColumnDef.PrimaryKey.
	CreateTable(ctx context.Context, name string, columns []ColumnDef, ifNotExists bool) error

	// AddColumn adds a new column to an existing table.
	AddColumn(ctx context.Context, table string, column ColumnDef) error

	// DropColumn removes a existing column.
	DropColumn(ctx context.Context, table, column string) error

	// DropTable removes an existing table.
	DropTable(ctx context.Context, table string, ifExists bool) error

	// InsertRow inserts a new row into the specified table with the provided data.
	Insert(ctx context.Context, table string, data Row) error

	// GetRows retrieves rows from the specified table with optional limit and offset for pagination.
	Rows(ctx context.Context, table string, limit, offset int) ([]Row, error)

	// UpdateRow updates rows in the specified table that match the given conditions.
	// The key must contain all primary-key columns and their values (supports composite PKs).
	Update(ctx context.Context, table string, key Key, data Row) error

	// DeleteRow deletes rows from the specified table that match the given conditions.
	// The key must contain all primary-key columns and their values (supports composite PKs).
	Delete(ctx context.Context, table string, key Key) error

	// ExecuteQuery executes a raw SQL query and returns the results.
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(ctx context.Context, query string, args ...any) ([]Row, error)
}
