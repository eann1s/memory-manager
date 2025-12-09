package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRunUpAppliesNewMigrations(t *testing.T) {
	dir := t.TempDir()

	first := filepath.Join(dir, "0001_alpha.sql")
	second := filepath.Join(dir, "0002_beta.sql")

	writeFile(t, first, "CREATE TABLE alpha (id INT);")
	writeFile(t, second, "CREATE TABLE beta (id INT);")

	db := &stubDB{
		queryData: []string{"0001_alpha.sql"},
	}

	if err := runUp(context.Background(), db, dir); err != nil {
		t.Fatalf("runUp returned error: %v", err)
	}

	if len(db.execCalls) != 3 {
		t.Fatalf("expected 3 exec calls, got %d (%v)", len(db.execCalls), db.execCalls)
	}

	if !strings.Contains(db.execCalls[0].sql, "CREATE TABLE IF NOT EXISTS schema_migrations") {
		t.Fatalf("unexpected first exec SQL: %s", db.execCalls[0].sql)
	}

	if !strings.Contains(db.execCalls[1].sql, "CREATE TABLE beta") {
		t.Fatalf("expected second exec to run migration SQL, got: %s", db.execCalls[1].sql)
	}

	insertCall := db.execCalls[2]
	if !strings.HasPrefix(insertCall.sql, "INSERT INTO schema_migrations") {
		t.Fatalf("expected insert statement, got: %s", insertCall.sql)
	}
	if len(insertCall.args) != 1 || insertCall.args[0] != "0002_beta.sql" {
		t.Fatalf("expected insert args to include new filename, got: %v", insertCall.args)
	}
}

func TestRunUpNoFilesDoesNothing(t *testing.T) {
	dir := t.TempDir()
	db := &stubDB{}

	if err := runUp(context.Background(), db, dir); err != nil {
		t.Fatalf("runUp returned error: %v", err)
	}

	if len(db.execCalls) != 0 {
		t.Fatalf("expected no exec calls when directory empty, got %d", len(db.execCalls))
	}
}

func TestListMigrationFilesFiltersAndSorts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b.sql"), "SELECT 1;")
	writeFile(t, filepath.Join(dir, "a.sql"), "SELECT 1;")
	writeFile(t, filepath.Join(dir, "ignore.txt"), "noop")

	files, err := listMigrationFiles(dir)
	if err != nil {
		t.Fatalf("listMigrationFiles error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 SQL files, got %d (%v)", len(files), files)
	}
	if filepath.Base(files[0]) != "a.sql" || filepath.Base(files[1]) != "b.sql" {
		t.Fatalf("expected files sorted alphabetically, got %v", files)
	}
}

type execCall struct {
	sql  string
	args []any
}

type stubDB struct {
	execCalls []execCall
	queryData []string
	queryErr  error
}

func (s *stubDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	call := execCall{sql: sql}
	if len(args) > 0 {
		call.args = append(call.args, args...)
	}
	s.execCalls = append(s.execCalls, call)
	return pgconn.CommandTag{}, nil
}

func (s *stubDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	rows := &stubRows{data: append([]string(nil), s.queryData...)}
	return rows, nil
}

type stubRows struct {
	data   []string
	index  int
	closed bool
	err    error
}

func (r *stubRows) Close() {
	r.closed = true
}

func (r *stubRows) Err() error {
	return r.err
}

func (r *stubRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (r *stubRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *stubRows) Next() bool {
	if r.err != nil || r.index >= len(r.data) {
		r.closed = true
		return false
	}
	r.index++
	return true
}

func (r *stubRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.data) {
		return fmt.Errorf("scan called without next")
	}
	if len(dest) == 0 {
		return fmt.Errorf("missing destination")
	}
	ptr, ok := dest[0].(*string)
	if !ok {
		return fmt.Errorf("dest must be *string")
	}
	*ptr = r.data[r.index-1]
	return nil
}

func (r *stubRows) Values() ([]any, error) {
	if r.index == 0 || r.index > len(r.data) {
		return nil, fmt.Errorf("values requested without row")
	}
	return []any{r.data[r.index-1]}, nil
}

func (r *stubRows) RawValues() [][]byte {
	return nil
}

func (r *stubRows) Conn() *pgx.Conn {
	return nil
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("failed writing %s: %v", path, err)
	}
}
