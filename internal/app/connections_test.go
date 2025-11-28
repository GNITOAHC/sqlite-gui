package app

import (
	"context"
	"errors"
	"testing"

	"sqlite-gui/pkg/database"
	"sqlite-gui/pkg/database/sqlite"
)

func TestConnectionManagerAddAndGet(t *testing.T) {
	ctx := context.Background()
	mgr := NewConnectionManager(func() database.Database { return sqlite.New() })
	t.Cleanup(func() {
		_ = mgr.CloseAll()
	})

	if err := mgr.Add(ctx, "primary", ":memory:"); err != nil {
		t.Fatalf("add primary: %v", err)
	}
	if err := mgr.Add(ctx, "secondary", ":memory:"); err != nil {
		t.Fatalf("add secondary: %v", err)
	}

	db, err := mgr.Get("primary")
	if err != nil {
		t.Fatalf("get primary: %v", err)
	}
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping primary: %v", err)
	}

	if _, err := mgr.Get("missing"); !errors.Is(err, ErrConnectionMiss) {
		t.Fatalf("expected ErrConnectionMiss, got %v", err)
	}

	conns := mgr.List()
	if len(conns) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(conns))
	}
	if mgr.Default() != "primary" {
		t.Fatalf("expected default to be primary, got %s", mgr.Default())
	}
}

func TestConnectionManagerRejectsDuplicates(t *testing.T) {
	ctx := context.Background()
	mgr := NewConnectionManager(func() database.Database { return sqlite.New() })
	t.Cleanup(func() {
		_ = mgr.CloseAll()
	})

	if err := mgr.Add(ctx, "primary", ":memory:"); err != nil {
		t.Fatalf("add primary: %v", err)
	}
	if err := mgr.Add(ctx, "primary", ":memory:"); !errors.Is(err, ErrConnectionExists) {
		t.Fatalf("expected ErrConnectionExists, got %v", err)
	}
}
