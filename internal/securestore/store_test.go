package securestore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dbkey.bin")
	secret := "my-db-passphrase-123456"
	if err := Save(path, secret); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, ok := Load(path)
	if !ok {
		t.Fatal("load failed")
	}
	if got != secret {
		t.Fatalf("mismatch: got %q want %q", got, secret)
	}
}

func TestLoadMissing(t *testing.T) {
	if _, ok := Load(filepath.Join(t.TempDir(), "nope.bin")); ok {
		t.Fatal("expected load failure for missing file")
	}
}

func TestLoadCorrupted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dbkey.bin")
	if err := os.WriteFile(path, []byte("not dpapi data at all"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := Load(path); ok {
		t.Fatal("expected load failure for corrupted data")
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dbkey.bin")
	if err := Save(path, "s"); err != nil {
		t.Fatal(err)
	}
	if err := Delete(path); err != nil {
		t.Fatal(err)
	}
	if err := Delete(path); err != nil {
		t.Fatalf("delete missing should succeed: %v", err)
	}
}
