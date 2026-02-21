package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var migrationNameRE = regexp.MustCompile(`^(\d{4})_.+\.sql$`)

// Migration is a single ordered SQL migration file.
type Migration struct {
	Seq    int
	Name   string
	Path   string
	SQL    string
	SHA256 string
}

// DiscoverMigrations loads and validates ordered SQL migrations from a directory.
func DiscoverMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	migs := make([]Migration, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".sql") {
			continue
		}
		m := migrationNameRE.FindStringSubmatch(e.Name())
		if len(m) != 2 {
			return nil, fmt.Errorf("db: invalid migration file name: %s", e.Name())
		}
		seq := 0
		if _, err := fmt.Sscanf(m[1], "%d", &seq); err != nil {
			return nil, fmt.Errorf("db: invalid migration sequence: %s", e.Name())
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sqlText := string(raw)
		if !hasTxGuards(sqlText) {
			return nil, fmt.Errorf("db: migration missing BEGIN/COMMIT guard: %s", e.Name())
		}
		migs = append(migs, Migration{
			Seq:    seq,
			Name:   e.Name(),
			Path:   path,
			SQL:    sqlText,
			SHA256: SHA256Hex(raw),
		})
	}

	sort.Slice(migs, func(i, j int) bool { return migs[i].Seq < migs[j].Seq })
	if err := ValidateMigrationOrder(migs); err != nil {
		return nil, err
	}
	return migs, nil
}

// ValidateMigrationOrder ensures sequences are unique and contiguous from 1.
func ValidateMigrationOrder(migrations []Migration) error {
	if len(migrations) == 0 {
		return fmt.Errorf("db: no migrations found")
	}
	seen := map[int]struct{}{}
	expected := 1
	for _, m := range migrations {
		if _, ok := seen[m.Seq]; ok {
			return fmt.Errorf("db: duplicate migration sequence: %04d", m.Seq)
		}
		seen[m.Seq] = struct{}{}
		if m.Seq != expected {
			return fmt.Errorf("db: migration gap, expected %04d found %04d", expected, m.Seq)
		}
		expected++
	}
	return nil
}

// EnsureMigrationTable ensures the migration state table exists.
func EnsureMigrationTable(ctx context.Context, db *sql.DB) error {
	const q = `
CREATE SCHEMA IF NOT EXISTS ops;
CREATE TABLE IF NOT EXISTS ops.schema_migrations (
    seq          INTEGER PRIMARY KEY,
    name         TEXT NOT NULL,
    sha256       TEXT NOT NULL,
    applied_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);`
	_, err := db.ExecContext(ctx, q)
	return err
}

// AppliedMigrations returns applied sequence -> checksum.
func AppliedMigrations(ctx context.Context, db *sql.DB) (map[int]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT seq, sha256 FROM ops.schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int]string)
	for rows.Next() {
		var seq int
		var sha string
		if err := rows.Scan(&seq, &sha); err != nil {
			return nil, err
		}
		out[seq] = sha
	}
	return out, rows.Err()
}

// ApplyMigrations applies pending migrations or prints them when dryRun=true.
func ApplyMigrations(ctx context.Context, db *sql.DB, migrations []Migration, dryRun bool) error {
	if err := ValidateMigrationOrder(migrations); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	if db == nil {
		return fmt.Errorf("db: nil handle")
	}
	if err := EnsureMigrationTable(ctx, db); err != nil {
		return err
	}
	applied, err := AppliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if existing, ok := applied[m.Seq]; ok {
			if existing != m.SHA256 {
				return fmt.Errorf("db: migration checksum mismatch for %04d", m.Seq)
			}
			continue
		}
		if _, err := db.ExecContext(ctx, m.SQL); err != nil {
			return fmt.Errorf("db: apply %s: %w", m.Name, err)
		}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO ops.schema_migrations (seq, name, sha256, applied_at) VALUES ($1, $2, $3, $4)`,
			m.Seq,
			m.Name,
			m.SHA256,
			time.Now().UTC(),
		); err != nil {
			return fmt.Errorf("db: record migration %s: %w", m.Name, err)
		}
	}
	return nil
}

func hasTxGuards(sqlText string) bool {
	lines := strings.Split(sqlText, "\n")
	hasBegin := false
	hasCommit := false
	for _, l := range lines {
		norm := strings.ToUpper(strings.TrimSpace(l))
		if norm == "BEGIN;" || norm == "BEGIN" {
			hasBegin = true
		}
		if norm == "COMMIT;" || norm == "COMMIT" {
			hasCommit = true
		}
	}
	return hasBegin && hasCommit
}
