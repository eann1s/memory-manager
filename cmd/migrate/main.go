package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eann1s/codex-memory-manager/internal/config"
	"github.com/eann1s/codex-memory-manager/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type execQuerier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
}

func main() {
	path := flag.String("path", "migrations", "path to the migrations directory")
	dsn := flag.String("database", "", "database connection string (defaults to env-configured DB)")
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatal("expected a single command (only 'up' is supported)")
	}
	cmd := flag.Arg(0)
	if cmd != "up" {
		log.Fatalf("unsupported command %q (only 'up' is supported)", cmd)
	}

	cfg := config.Load()
	if *dsn == "" {
		*dsn = cfg.DBURL
	}

	ctx := context.Background()
	db, err := store.NewDB(ctx, *dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Pool.Close()

	if err := runUp(ctx, db.Pool, *path); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("migrations applied successfully")
}

func runUp(ctx context.Context, pool execQuerier, dir string) error {
	files, err := listMigrationFiles(dir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		log.Printf("no migration files found in %s", dir)
		return nil
	}

	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return err
	}
	applied, err := loadAppliedMigrations(ctx, pool)
	if err != nil {
		return err
	}

	var appliedCount int
	for _, file := range files {
		name := filepath.Base(file)
		if _, alreadyApplied := applied[name]; alreadyApplied {
			continue
		}
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("applying %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
			return fmt.Errorf("recording %s: %w", name, err)
		}
		appliedCount++
		log.Printf("applied %s", name)
	}

	if appliedCount == 0 {
		log.Println("no new migrations to apply")
	}

	return nil
}

func listMigrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading migrations dir: %w", err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

func ensureMigrationsTable(ctx context.Context, pool execQuerier) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}
	return nil
}

func loadAppliedMigrations(ctx context.Context, pool execQuerier) (map[string]struct{}, error) {
	rows, err := pool.Query(ctx, `SELECT filename FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("querying applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = struct{}{}
	}
	return applied, rows.Err()
}
