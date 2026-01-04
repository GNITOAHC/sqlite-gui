// Package postgresql provides PostgreSQL database connectivity and operations.
package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"sqlite-gui/pkg/database"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Postgres implements the database.Database interface using the pgx driver.
type Postgres struct {
	db *sql.DB
}

func New() *Postgres {
	return &Postgres{}
}

func (p *Postgres) Connect(ctx context.Context, conn string) error {
	db, err := sql.Open("pgx", conn)
	if err != nil {
		return err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return err
	}
	p.db = db
	return nil
}

func (p *Postgres) Close() error {
	if p.db == nil {
		return nil
	}
	err := p.db.Close()
	p.db = nil
	return err
}

func (p *Postgres) Ping(ctx context.Context) error {
	if err := p.ensureConnected(); err != nil {
		return err
	}
	return p.db.PingContext(ctx)
}

func (p *Postgres) Tables(ctx context.Context) ([]string, error) {
	if err := p.ensureConnected(); err != nil {
		return nil, err
	}
	// Defaults to public schema for now
	query := "SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = 'public' ORDER BY tablename"
	rows, err := p.db.QueryContext(ctx, query)
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

func (p *Postgres) Columns(ctx context.Context, table string) ([]database.Column, error) {
	if err := p.ensureConnected(); err != nil {
		return nil, err
	}

	// 1. Get Primary Keys
	pks := make(map[string]int)
	pkQuery := `
		SELECT kcu.column_name, kcu.ordinal_position
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.table_constraints tc ON kcu.constraint_name = tc.constraint_name
		WHERE kcu.table_name = $1 AND kcu.table_schema = 'public' AND tc.constraint_type = 'PRIMARY KEY'
	`
	pkRows, err := p.db.QueryContext(ctx, pkQuery, table)
	if err != nil {
		return nil, err
	}
	defer pkRows.Close()
	for pkRows.Next() {
		var name string
		var pos int
		if err := pkRows.Scan(&name, &pos); err == nil {
			pks[name] = pos
		}
	}

	// 2. Get Foreign Keys
	fks := make(map[string][]database.ForeignKey)
	fkQuery := `
		SELECT
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.update_rule,
			rc.delete_rule
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.referential_constraints rc ON kcu.constraint_name = rc.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON rc.constraint_name = ccu.constraint_name
		WHERE kcu.table_name = $1 AND kcu.table_schema = 'public'
	`
	fkRows, err := p.db.QueryContext(ctx, fkQuery, table)
	if err != nil {
		return nil, err
	}
	defer fkRows.Close()
	for fkRows.Next() {
		var col, refTable, refCol, upRule, delRule string
		if err := fkRows.Scan(&col, &refTable, &refCol, &upRule, &delRule); err == nil {
			fks[col] = append(fks[col], database.ForeignKey{
				RefTable: refTable,
				FromCol:  col,
				ToCol:    refCol,
				OnUpdate: database.ForeignKeyAction(upRule),
				OnDelete: database.ForeignKeyAction(delRule),
			})
		}
	}

	// 3. Get Columns
	colQuery := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = $1 AND table_schema = 'public'
		ORDER BY ordinal_position
	`
	rows, err := p.db.QueryContext(ctx, colQuery, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []database.Column
	for rows.Next() {
		var name, dataType, isNullable string
		var defaultVal sql.NullString
		if err := rows.Scan(&name, &dataType, &isNullable, &defaultVal); err != nil {
			return nil, err
		}

		pkIdx, isPk := pks[name]
		columns = append(columns, database.Column{
			Name:            name,
			Type:            dataType,
			NotNull:         isNullable == "NO",
			Default:         defaultVal,
			PrimaryKey:      isPk,
			PrimaryKeyIndex: pkIdx,
			ForeignKeys:     fks[name],
		})
	}

	return columns, rows.Err()
}

func (p *Postgres) CreateTable(ctx context.Context, name string, columns []database.ColumnDef, ifNotExists bool) error {
	if err := p.ensureConnected(); err != nil {
		return err
	}
	stmt, err := buildCreateTableSQL(name, columns, ifNotExists)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx, stmt)
	return err
}

func (p *Postgres) AddColumn(ctx context.Context, table string, column database.ColumnDef) error {
	if err := p.ensureConnected(); err != nil {
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
	_, err = p.db.ExecContext(ctx, stmt)
	return err
}

func (p *Postgres) DropColumn(ctx context.Context, table, column string) error {
	if err := p.ensureConnected(); err != nil {
		return err
	}
	if strings.TrimSpace(table) == "" || strings.TrimSpace(column) == "" {
		return fmt.Errorf("table and column are required")
	}
	stmt := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", quoteIdent(table), quoteIdent(column))
	_, err := p.db.ExecContext(ctx, stmt)
	return err
}

func (p *Postgres) DropTable(ctx context.Context, table string, ifExists bool) error {
	if err := p.ensureConnected(); err != nil {
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
	_, err := p.db.ExecContext(ctx, stmt)
	return err
}

func (p *Postgres) Rows(ctx context.Context, table string, limit, offset int) ([]database.Row, error) {
	if err := p.ensureConnected(); err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s", quoteIdent(table))
	args := []any{}
	
	// Postgres LIMIT/OFFSET
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, limit)
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", len(args)+1)
		args = append(args, offset)
	}
	return p.Query(ctx, query, args...)
}

func (p *Postgres) Insert(ctx context.Context, table string, data database.Row) error {
	if err := p.ensureConnected(); err != nil {
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
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		values[i] = data[key]
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoteIdent(table), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	_, err := p.db.ExecContext(ctx, query, values...)
	return err
}

func (p *Postgres) Update(ctx context.Context, table string, key database.Key, data database.Row) error {
	if err := p.ensureConnected(); err != nil {
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
	args := make([]any, 0, len(data)+len(key))
	
	for i, col := range keys {
		args = append(args, data[col])
		setClauses[i] = fmt.Sprintf("%s = $%d", quoteIdent(col), len(args))
	}
	
	where, whereArgs, err := buildWhere(key, len(args)+1)
	if err != nil {
		return err
	}
	args = append(args, whereArgs...)
	
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", quoteIdent(table), strings.Join(setClauses, ", "), where)
	_, err = p.db.ExecContext(ctx, query, args...)
	return err
}

func (p *Postgres) Delete(ctx context.Context, table string, key database.Key) error {
	if err := p.ensureConnected(); err != nil {
		return err
	}
	if len(key) == 0 {
		return fmt.Errorf("no primary key provided for %s", table)
	}
	where, args, err := buildWhere(key, 1)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", quoteIdent(table), where)
	_, err = p.db.ExecContext(ctx, query, args...)
	return err
}

func (p *Postgres) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if err := p.ensureConnected(); err != nil {
		return nil, err
	}
	return p.db.ExecContext(ctx, query, args...)
}

func (p *Postgres) Query(ctx context.Context, query string, args ...any) ([]database.Row, error) {
	if err := p.ensureConnected(); err != nil {
		return nil, err
	}
	rows, err := p.db.QueryContext(ctx, query, args...)
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

func (p *Postgres) ensureConnected() error {
	if p.db == nil {
		return database.ErrNotConnected
	}
	return nil
}

func buildWhere(key database.Key, startParamIndex int) (string, []any, error) {
	if len(key) == 0 {
		return "", nil, fmt.Errorf("where key is empty")
	}
	cols := orderedKeys(key)
	clauses := make([]string, len(cols))
	args := make([]any, len(cols))
	for i, col := range cols {
		clauses[i] = fmt.Sprintf("%s = $%d", quoteIdent(col), startParamIndex+i)
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
