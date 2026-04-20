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

func TestStoresDir(t *testing.T) {
	t.Setenv("UNION_DIR", "/tmp/union-test")
	got, err := StoresDir()
	if err != nil {
		t.Fatalf("StoresDir: %v", err)
	}
	if got != "/tmp/union-test/stores" {
		t.Errorf("got %q, want /tmp/union-test/stores", got)
	}
}

func TestStoreDir_Valid(t *testing.T) {
	t.Setenv("UNION_DIR", "/tmp/union-test")
	got, err := StoreDir("personal")
	if err != nil {
		t.Fatalf("StoreDir: %v", err)
	}
	if got != "/tmp/union-test/stores/personal" {
		t.Errorf("got %q", got)
	}
}

func TestStoreDir_Invalid(t *testing.T) {
	t.Setenv("UNION_DIR", "/tmp/union-test")
	if _, err := StoreDir("Bad Name"); err == nil {
		t.Fatal("expected error for invalid store name")
	}
}
