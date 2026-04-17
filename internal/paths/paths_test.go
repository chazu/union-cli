package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnionDir_Default(t *testing.T) {
	t.Setenv("UNION_DIR", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	got, err := UnionDir()
	if err != nil {
		t.Fatalf("UnionDir: %v", err)
	}
	want := filepath.Join(home, ".union")
	if got != want {
		t.Errorf("UnionDir() = %q, want %q", got, want)
	}
}

func TestUnionDir_EnvOverride(t *testing.T) {
	t.Setenv("UNION_DIR", "/tmp/my-union")
	got, err := UnionDir()
	if err != nil {
		t.Fatalf("UnionDir: %v", err)
	}
	if got != "/tmp/my-union" {
		t.Errorf("UnionDir() = %q, want /tmp/my-union", got)
	}
}

func TestClausesDir(t *testing.T) {
	t.Setenv("UNION_DIR", "/tmp/u")
	got, err := ClausesDir()
	if err != nil {
		t.Fatalf("ClausesDir: %v", err)
	}
	if got != "/tmp/u/clauses" {
		t.Errorf("ClausesDir() = %q, want /tmp/u/clauses", got)
	}
}

func TestShopsFile(t *testing.T) {
	t.Setenv("UNION_DIR", "/tmp/u")
	got, err := ShopsFile()
	if err != nil {
		t.Fatalf("ShopsFile: %v", err)
	}
	if got != "/tmp/u/shops.toml" {
		t.Errorf("ShopsFile() = %q, want /tmp/u/shops.toml", got)
	}
}
