// Package sqlite provides SQLite database connectivity and operations.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"sqlite-gui/pkg/database"

	_ "modernc.org/sqlite"
)

// SQLite implements the database.Database interface using the modernc SQLite driver.
type SQLite struct {
	db *sql.DB
}

func New() *SQLite {
	return &SQLite{}
}

func (s *SQLite) Connect(ctx context.Context, conn string) error {
	db, err := sql.Open("sqlite", conn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1) // SQLite is single writer, keep it conservative.
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return err
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return err
	}
	s.db = db
	return nil
}

func (s *SQLite) Close() error {
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

func (s *SQLite) Ping(ctx context.Context) error {
	if err := s.ensureConnected(); err != nil {
		return err
	}
	return s.db.PingContext(ctx)
}

func (s *SQLite) Tables(ctx context.Context) ([]string, error) {
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

func (s *SQLite) Columns(ctx context.Context, table string) ([]database.Column, error) {
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	fkMap, err := s.foreignKeys(ctx, table)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("PRAGMA table_info(%s)", quoteIdent(table))
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []database.Column
	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, database.Column{
			Name:            name,
			Type:            colType,
			NotNull:         notNull == 1,
			Default:         defaultVal,
			PrimaryKey:      pk > 0,
			PrimaryKeyIndex: pk,
			ForeignKeys: func() []database.ForeignKey {
				if fks, ok := fkMap[name]; ok {
					return fks
				}
				return nil
			}(),
		})
	}
	return columns, rows.Err()
}

func (s *SQLite) CreateTable(ctx context.Context, name string, columns []database.ColumnDef, ifNotExists bool) error {
	if err := s.ensureConnected(); err != nil {
		return err
	}
	stmt, err := buildCreateTableSQL(name, columns, ifNotExists)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, stmt)
	return err
}

func (s *SQLite) AddColumn(ctx context.Context, table string, column database.ColumnDef) error {
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if strings.TrimSpace(table) == "" {
		return fmt.Errorf("table name is required")
	}
	if column.PrimaryKey {
		return fmt.Errorf("adding primary key columns via ALTER TABLE is not supported")
	}
	definition, err := buildColumnDefinition(column, false)
	if err != nil {
		return err
	}
	stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", quoteIdent(table), definition)
	_, err = s.db.ExecContext(ctx, stmt)
	return err
}

func (s *SQLite) DropColumn(ctx context.Context, table, column string) error {
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if strings.TrimSpace(table) == "" || strings.TrimSpace(column) == "" {
		return fmt.Errorf("table and column are required")
	}
	stmt := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", quoteIdent(table), quoteIdent(column))
	_, err := s.db.ExecContext(ctx, stmt)
	return err
}

func (s *SQLite) DropTable(ctx context.Context, table string, ifExists bool) error {
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if strings.TrimSpace(table) == "" {
		return fmt.Errorf("table name is required")
	}
	stmt := "DROP TABLE "
	if ifExists {
		stmt += "IF EXISTS "
	}
	stmt += quoteIdent(table)
	_, err := s.db.ExecContext(ctx, stmt)
	return err
}

func (s *SQLite) Rows(ctx context.Context, table string, limit, offset int) ([]database.Row, error) {
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s", quoteIdent(table))
	args := []any{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		if limit <= 0 {
			query += " LIMIT -1"
		}
		query += " OFFSET ?"
		args = append(args, offset)
	}
	return s.Query(ctx, query, args...)
}

func (s *SQLite) Insert(ctx context.Context, table string, data database.Row) error {
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("no data to insert into %s", table)
	}
	keys := orderedKeys(data)
	columns := make([]string, len(keys))
	placeholders := make([]string, len(keys))
	values := make([]any, len(keys))
	for i, key := range keys {
		columns[i] = quoteIdent(key)
		placeholders[i] = "?"
		values[i] = data[key]
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoteIdent(table), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	_, err := s.db.ExecContext(ctx, query, values...)
	return err
}

func (s *SQLite) Update(ctx context.Context, table string, key database.Key, data database.Row) error {
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if len(key) == 0 {
		return fmt.Errorf("no primary key provided for %s", table)
	}
	if len(data) == 0 {
		return fmt.Errorf("no data to update for %s", table)
	}
	keys := orderedKeys(data)
	setClauses := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, key := range keys {
		setClauses[i] = fmt.Sprintf("%s = ?", quoteIdent(key))
		args[i] = data[key]
	}
	where, whereArgs, err := buildWhere(key)
	if err != nil {
		return err
	}
	args = append(args, whereArgs...)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", quoteIdent(table), strings.Join(setClauses, ", "), where)
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *SQLite) Delete(ctx context.Context, table string, key database.Key) error {
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if len(key) == 0 {
		return fmt.Errorf("no primary key provided for %s", table)
	}
	where, args, err := buildWhere(key)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", quoteIdent(table), where)
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *SQLite) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	return s.db.ExecContext(ctx, query, args...)
}

func (s *SQLite) Query(ctx context.Context, query string, args ...any) ([]database.Row, error) {
	if err := s.ensureConnected(); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []database.Row
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return nil, err
		}
		row := database.Row{}
		for i, col := range columns {
			switch v := values[i].(type) {
			case []byte:
				row[col] = string(v)
			default:
				row[col] = v
			}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func (s *SQLite) foreignKeys(ctx context.Context, table string) (map[string][]database.ForeignKey, error) {
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", quoteIdent(table))
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]database.ForeignKey)
	for rows.Next() {
		var (
			id, seq  int
			refTbl   string
			from     string
			to       string
			onUpdate string
			onDelete string
			match    string
		)
		if err := rows.Scan(&id, &seq, &refTbl, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}
		fk := database.ForeignKey{
			RefTable: refTbl,
			FromCol:  from,
			ToCol:    to,
			OnDelete: database.ForeignKeyAction(onDelete),
			OnUpdate: database.ForeignKeyAction(onUpdate),
		}
		result[from] = append(result[from], fk)
	}
	return result, rows.Err()
}

func (s *SQLite) ensureConnected() error {
	if s.db == nil {
		return database.ErrNotConnected
	}
	return nil
}

func buildWhere(key database.Key) (string, []any, error) {
	if len(key) == 0 {
		return "", nil, fmt.Errorf("where key is empty")
	}
	cols := orderedKeys(key)
	clauses := make([]string, len(cols))
	args := make([]any, len(cols))
	for i, col := range cols {
		clauses[i] = fmt.Sprintf("%s = ?", quoteIdent(col))
		args[i] = key[col]
	}
	return strings.Join(clauses, " AND "), args, nil
}

func buildCreateTableSQL(name string, columns []database.ColumnDef, ifNotExists bool) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("table name is required")
	}
	if len(columns) == 0 {
		return "", fmt.Errorf("at least one column is required")
	}
	pkCount := 0
	for _, col := range columns {
		if col.PrimaryKey {
			pkCount++
		}
	}
	var defs []string
	var pkCols []string
	for _, col := range columns {
		def, err := buildColumnDefinition(col, pkCount == 1 && col.PrimaryKey)
		if err != nil {
			return "", err
		}
		defs = append(defs, def)
		if col.PrimaryKey {
			pkCols = append(pkCols, quoteIdent(col.Name))
		}
	}
	if len(pkCols) > 1 {
		defs = append(defs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
	}
	stmt := "CREATE TABLE "
	if ifNotExists {
		stmt += "IF NOT EXISTS "
	}
	stmt += fmt.Sprintf("%s (%s)", quoteIdent(name), strings.Join(defs, ", "))
	return stmt, nil
}

func buildColumnDefinition(col database.ColumnDef, allowInlinePK bool) (string, error) {
	if strings.TrimSpace(col.Name) == "" || strings.TrimSpace(col.Type) == "" {
		return "", fmt.Errorf("column name and type are required")
	}
	parts := []string{quoteIdent(col.Name), col.Type}
	if col.NotNull {
		parts = append(parts, "NOT NULL")
	}
	if col.Default != nil {
		parts = append(parts, "DEFAULT "+*col.Default)
	}
	if col.PrimaryKey && allowInlinePK {
		parts = append(parts, "PRIMARY KEY")
	}
	return strings.Join(parts, " "), nil
}

func orderedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func quoteIdent(name string) string {
	escaped := strings.ReplaceAll(name, `"`, `""`)
	return `"` + escaped + `"`
}
