package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverMigrationsOrdered(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "0001_one.sql"), "BEGIN;\nSELECT 1;\nCOMMIT;\n")
	writeFile(t, filepath.Join(dir, "0002_two.sql"), "BEGIN;\nSELECT 2;\nCOMMIT;\n")

	migs, err := DiscoverMigrations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(migs) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migs))
	}
	if migs[0].Seq != 1 || migs[1].Seq != 2 {
		t.Fatalf("unexpected migration order: %+v", migs)
	}
}

func TestDiscoverMigrationsRejectGap(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "0001_one.sql"), "BEGIN;\nSELECT 1;\nCOMMIT;\n")
	writeFile(t, filepath.Join(dir, "0003_three.sql"), "BEGIN;\nSELECT 3;\nCOMMIT;\n")

	_, err := DiscoverMigrations(dir)
	if err == nil {
		t.Fatalf("expected sequence gap error")
	}
}

func TestDiscoverMigrationsRequiresTxGuards(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "0001_one.sql"), "SELECT 1;\n")

	_, err := DiscoverMigrations(dir)
	if err == nil {
		t.Fatalf("expected tx guard error")
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
