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
	Name       string
	Type       string
	NotNull    bool // Whether the column can be null
	Default    sql.NullString
	PrimaryKey bool

	ForeignKeys []ForeignKey
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

	// InsertRow inserts a new row into the specified table with the provided data.
	Insert(ctx context.Context, table string, data Row) error

	// GetRows retrieves rows from the specified table with optional limit and offset for pagination.
	Rows(ctx context.Context, table string, limit, offset int) ([]Row, error)

	// UpdateRow updates rows in the specified table that match the given conditions.
	Update(ctx context.Context, table, pkColumn string, pkValue any, data Row) error

	// DeleteRow deletes rows from the specified table that match the given conditions.
	Delete(ctx context.Context, table, pkColumn string, pkValue any) error

	// ExecuteQuery executes a raw SQL query and returns the results.
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(ctx context.Context, query string, args ...any) ([]Row, error)
}
