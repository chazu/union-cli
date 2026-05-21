package harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncPointers(t *testing.T) {
	dir := t.TempDir()
	created, err := SyncPointers(dir, "AGENTS.md", []string{"CLAUDE.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 || created[0] != "CLAUDE.md" {
		t.Fatalf("expected [CLAUDE.md], got %v", created)
	}

	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "@AGENTS.md\n" {
		t.Fatalf("expected '@AGENTS.md\\n', got %q", string(data))
	}
}

func TestSyncPointersSkipsSameAsContract(t *testing.T) {
	dir := t.TempDir()
	created, err := SyncPointers(dir, "AGENTS.md", []string{"AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Fatal("should skip target that matches contract")
	}
}

func TestSyncPointersRefusesNonPointer(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("real content here"), 0o644)

	_, err := SyncPointers(dir, "AGENTS.md", []string{"CLAUDE.md"})
	if err == nil {
		t.Fatal("should refuse to overwrite non-pointer file")
	}
}

func TestSyncPointersUpdatesExistingPointer(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("@OLD.md\n"), 0o644)

	created, err := SyncPointers(dir, "AGENTS.md", []string{"CLAUDE.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 {
		t.Fatal("should update existing pointer")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if string(data) != "@AGENTS.md\n" {
		t.Fatalf("expected updated pointer, got %q", string(data))
	}
}

func TestDefaultPointerTargets(t *testing.T) {
	adapters := []Adapter{&Claude{}, &OpenCode{}}
	targets := DefaultPointerTargets("AGENTS.md", adapters)
	if len(targets) != 1 || targets[0] != "CLAUDE.md" {
		t.Fatalf("expected [CLAUDE.md], got %v", targets)
	}
}

func TestDefaultPointerTargetsNoneNeeded(t *testing.T) {
	adapters := []Adapter{&Claude{}}
	targets := DefaultPointerTargets("CLAUDE.md", adapters)
	if len(targets) != 0 {
		t.Fatalf("expected no targets when contract is CLAUDE.md, got %v", targets)
	}
}
