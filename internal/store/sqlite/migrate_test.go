package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNewRunsMigrations(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, err := New(ctx, dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	var count int
	if err := store.DB().QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query migrations: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least one applied migration")
	}
}
