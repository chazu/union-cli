package shop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_LoadMissing_ReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shops.toml")
	r, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if len(r.List()) != 0 {
		t.Errorf("expected empty registry, got %d shops", len(r.List()))
	}
}

func TestRegistry_AddSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shops.toml")
	r, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if err := r.Add("/home/me/proj", "AGENTS.md"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := r.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	r2, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	shops := r2.List()
	if len(shops) != 1 {
		t.Fatalf("want 1 shop, got %d", len(shops))
	}
	if shops[0].Dir != "/home/me/proj" || shops[0].Contract != "AGENTS.md" {
		t.Errorf("unexpected shop: %+v", shops[0])
	}
}

func TestRegistry_AddDuplicate(t *testing.T) {
	r, _ := LoadRegistry(filepath.Join(t.TempDir(), "shops.toml"))
	if err := r.Add("/x", "AGENTS.md"); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	if err := r.Add("/x", "AGENTS.md"); err == nil {
		t.Fatal("expected error adding duplicate shop")
	}
}

func TestRegistry_Remove(t *testing.T) {
	r, _ := LoadRegistry(filepath.Join(t.TempDir(), "shops.toml"))
	_ = r.Add("/a", "AGENTS.md")
	_ = r.Add("/b", "AGENTS.md")
	if err := r.Remove("/a"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(r.List()) != 1 {
		t.Errorf("want 1 shop after remove, got %d", len(r.List()))
	}
}

func TestRegistry_Remove_Missing(t *testing.T) {
	r, _ := LoadRegistry(filepath.Join(t.TempDir(), "shops.toml"))
	if err := r.Remove("/no/such"); err == nil {
		t.Fatal("expected error removing missing shop")
	}
}

func TestRegistry_Get(t *testing.T) {
	r, _ := LoadRegistry(filepath.Join(t.TempDir(), "shops.toml"))
	_ = r.Add("/x", "AGENTS.md")
	got, ok := r.Get("/x")
	if !ok {
		t.Fatal("Get: not found")
	}
	if got.Contract != "AGENTS.md" {
		t.Errorf("Get contract = %q, want AGENTS.md", got.Contract)
	}
	if _, ok := r.Get("/missing"); ok {
		t.Error("Get on missing shop returned ok=true")
	}
}

func TestRegistry_List_SortedByDir(t *testing.T) {
	r, _ := LoadRegistry(filepath.Join(t.TempDir(), "shops.toml"))
	_ = r.Add("/z", "AGENTS.md")
	_ = r.Add("/a", "AGENTS.md")
	_ = r.Add("/m", "AGENTS.md")
	shops := r.List()
	if shops[0].Dir != "/a" || shops[1].Dir != "/m" || shops[2].Dir != "/z" {
		t.Errorf("not sorted: %v", shops)
	}
}

func TestRegistry_FileSurvivesComment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shops.toml")
	if err := os.WriteFile(path, []byte("# top-level comment\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	r, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if len(r.List()) != 0 {
		t.Errorf("expected empty, got %d", len(r.List()))
	}
}
