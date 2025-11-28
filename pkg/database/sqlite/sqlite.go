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
			Name:       name,
			Type:       colType,
			NotNull:    notNull == 1,
			Default:    defaultVal,
			PrimaryKey: pk == 1,
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

func (s *SQLite) Update(ctx context.Context, table, pkColumn string, pkValue any, data database.Row) error {
	if err := s.ensureConnected(); err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("no data to update for %s", table)
	}
	keys := orderedKeys(data)
	setClauses := make([]string, len(keys))
	args := make([]any, len(keys)+1)
	for i, key := range keys {
		setClauses[i] = fmt.Sprintf("%s = ?", quoteIdent(key))
		args[i] = data[key]
	}
	args[len(args)-1] = pkValue
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", quoteIdent(table), strings.Join(setClauses, ", "), quoteIdent(pkColumn))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *SQLite) Delete(ctx context.Context, table, pkColumn string, pkValue any) error {
	if err := s.ensureConnected(); err != nil {
		return err
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", quoteIdent(table), quoteIdent(pkColumn))
	_, err := s.db.ExecContext(ctx, query, pkValue)
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
