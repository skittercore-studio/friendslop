package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/skittercore-studio/friendslop/internal/db"
)

func TestOpenAndMigrateIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	ctx := context.Background()
	d, err := db.Open(ctx, path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	// Schema rows present?
	var rooms int
	if err := d.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='rooms'`,
	).Scan(&rooms); err != nil {
		t.Fatalf("schema check: %v", err)
	}
	if rooms != 1 {
		t.Fatalf("expected rooms table, got count=%d", rooms)
	}

	// Migration ledger has the init row.
	var applied int
	if err := d.QueryRowContext(ctx,
		`SELECT count(*) FROM schema_migrations WHERE name = '0001_init.sql'`,
	).Scan(&applied); err != nil {
		t.Fatalf("ledger check: %v", err)
	}
	if applied != 1 {
		t.Fatalf("expected init applied once, got %d", applied)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Re-open: should be a no-op for migrations.
	d2, err := db.Open(ctx, path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer d2.Close()
	if err := d2.QueryRowContext(ctx,
		`SELECT count(*) FROM schema_migrations`,
	).Scan(&applied); err != nil {
		t.Fatalf("ledger check 2: %v", err)
	}
	if applied != 1 {
		t.Fatalf("expected migration ledger to remain at 1, got %d", applied)
	}
}

func TestSchemaConstraints(t *testing.T) {
	dir := t.TempDir()
	d, err := db.Open(context.Background(), filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer d.Close()

	// Insert a room with bogus state should fail the CHECK.
	if _, err := d.Exec(`INSERT INTO rooms (
		id, code, state, mode, pool_source, created_at, last_activity_at
	) VALUES ('r','XYZA','not-a-state','live','curated',1,1)`); err == nil {
		t.Fatalf("expected CHECK violation for state")
	}
	if _, err := d.Exec(`INSERT INTO rooms (
		id, code, state, mode, pool_source, created_at, last_activity_at
	) VALUES ('r','XYZA','lobby','telegraph','curated',1,1)`); err == nil {
		t.Fatalf("expected CHECK violation for mode")
	}
	if _, err := d.Exec(`INSERT INTO rooms (
		id, code, state, mode, pool_source, created_at, last_activity_at
	) VALUES ('r','XYZA','lobby','live','wiki',1,1)`); err == nil {
		t.Fatalf("expected CHECK violation for pool_source")
	}

	// Valid insert.
	if _, err := d.Exec(`INSERT INTO rooms (
		id, code, state, mode, pool_source, created_at, last_activity_at
	) VALUES ('r','XYZA','lobby','live','curated',1,1)`); err != nil {
		t.Fatalf("valid insert failed: %v", err)
	}
}
