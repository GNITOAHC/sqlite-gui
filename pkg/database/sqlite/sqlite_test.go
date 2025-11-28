package sqlite

import (
	"context"
	"testing"

	"sqlite-gui/pkg/database"
)

func TestConnectPingAndClose(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestTablesAndColumns(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.Exec(ctx, `CREATE TABLE parent (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if _, err := db.Exec(ctx, `CREATE TABLE child (
		id INTEGER PRIMARY KEY,
		parent_id INTEGER,
		name TEXT NOT NULL,
		FOREIGN KEY(parent_id) REFERENCES parent(id) ON DELETE CASCADE
	)`); err != nil {
		t.Fatalf("create child: %v", err)
	}

	tables, err := db.Tables(ctx)
	if err != nil {
		t.Fatalf("tables: %v", err)
	}
	expectedTables := []string{"child", "parent"}
	if len(tables) != len(expectedTables) {
		t.Fatalf("tables length mismatch: got %v want %v", tables, expectedTables)
	}
	for i, name := range expectedTables {
		if tables[i] != name {
			t.Fatalf("tables[%d]=%q want %q", i, tables[i], name)
		}
	}

	cols, err := db.Columns(ctx, "child")
	if err != nil {
		t.Fatalf("columns: %v", err)
	}
	if len(cols) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(cols))
	}

	var parentFK *database.ForeignKey
	for _, col := range cols {
		if col.Name == "id" && !col.PrimaryKey {
			t.Fatalf("id column should be primary key")
		}
		if col.Name == "name" && !col.NotNull {
			t.Fatalf("name column should be NOT NULL")
		}
		if col.Name == "parent_id" && len(col.ForeignKeys) == 1 {
			parentFK = &col.ForeignKeys[0]
		}
	}
	if parentFK == nil {
		t.Fatalf("expected foreign key on parent_id")
	}
	if parentFK.RefTable != "parent" || parentFK.ToCol != "id" || parentFK.OnDelete != database.ForeignKeyActionCascade {
		t.Fatalf("unexpected foreign key %+v", *parentFK)
	}
}

func TestCRUDAndQuery(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.Exec(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if err := db.Insert(ctx, "users", database.Row{"name": "alice", "age": 30}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := db.Rows(ctx, "users", 0, 0)
	if err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(rows) != 1 || rows[0]["name"] != "alice" {
		t.Fatalf("unexpected rows %v", rows)
	}

	if err := db.Update(ctx, "users", "id", 1, database.Row{"age": 31}); err != nil {
		t.Fatalf("update: %v", err)
	}
	rows, err = db.Rows(ctx, "users", 0, 0)
	if err != nil {
		t.Fatalf("rows after update: %v", err)
	}
	if rows[0]["age"] != int64(31) { // SQLite returns int64
		t.Fatalf("expected age 31, got %v", rows[0]["age"])
	}

	resultRows, err := db.Query(ctx, "SELECT name FROM users WHERE age = ?", 31)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(resultRows) != 1 || resultRows[0]["name"] != "alice" {
		t.Fatalf("unexpected query result %v", resultRows)
	}

	if err := db.Delete(ctx, "users", "id", 1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	rows, err = db.Rows(ctx, "users", 0, 0)
	if err != nil {
		t.Fatalf("rows after delete: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no rows after delete, got %v", rows)
	}
}

func newTestDB(t *testing.T) *SQLite {
	t.Helper()
	db := New()
	ctx := context.Background()
	if err := db.Connect(ctx, ":memory:"); err != nil {
		t.Fatalf("connect: %v", err)
	}
	return db
}
