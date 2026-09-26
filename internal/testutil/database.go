// Package testutil creates disposable schemas only on the explicit integration DSN.
package testutil

import (
	"context"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
)

func Database(t *testing.T) *store.Store {
	t.Helper()
	ctx := context.Background()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL required")
	}
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := "test_" + store.ID()
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close(); admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	s := &store.Store{DB: db}
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	return s
}
