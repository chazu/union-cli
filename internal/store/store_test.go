package store

import (
	"os/exec"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	s, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return s
}

func TestInit_CreatesGitRepo(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	if _, err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s2, err := Open(dir)
	if err != nil {
		t.Fatalf("Open after Init: %v", err)
	}
	if s2 == nil {
		t.Fatal("Open returned nil store")
	}
}

func TestInit_RefusesExisting(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	if _, err := Init(dir); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	if _, err := Init(dir); err == nil {
		t.Fatal("expected error on re-init, got nil")
	}
}

func TestOpen_ErrorsWhenMissing(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	if _, err := Open(dir); err == nil {
		t.Fatal("expected error opening uninitialized dir")
	}
}

func TestPutGet(t *testing.T) {
	s := newTestStore(t)
	body := []byte("# Identity\n\nBe helpful.\n")
	if err := s.Put("base/identity", body, "new base/identity"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get("base/identity")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("Get returned %q, want %q", got, body)
	}
}

func TestGet_Missing(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Get("no/such/path"); err == nil {
		t.Fatal("expected error for missing clause")
	}
}

func TestHas(t *testing.T) {
	s := newTestStore(t)
	if s.Has("x/y") {
		t.Error("Has on missing returned true")
	}
	if err := s.Put("x/y", []byte("hi"), "add x/y"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !s.Has("x/y") {
		t.Error("Has on existing returned false")
	}
}

func TestList(t *testing.T) {
	s := newTestStore(t)
	paths := []string{"base/identity", "base/style", "lang/go", "tone/snarky"}
	for _, p := range paths {
		if err := s.Put(p, []byte("x"), "add "+p); err != nil {
			t.Fatalf("Put %s: %v", p, err)
		}
	}
	all, err := s.List("")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("List all: got %d, want 4 (%v)", len(all), all)
	}
	base, err := s.List("base/")
	if err != nil {
		t.Fatalf("List base/: %v", err)
	}
	if len(base) != 2 {
		t.Errorf("List base/: got %d, want 2 (%v)", len(base), base)
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	if err := s.Put("tone/snarky", []byte("x"), "add"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Delete("tone/snarky", "expel tone/snarky"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if s.Has("tone/snarky") {
		t.Error("still present after Delete")
	}
}

func TestDelete_Missing(t *testing.T) {
	s := newTestStore(t)
	if err := s.Delete("no/such", "expel"); err == nil {
		t.Fatal("expected error deleting missing clause")
	}
}

func TestPath_Rejects(t *testing.T) {
	s := newTestStore(t)
	bad := []string{"", "/abs", "../escape", "has space", "a//b"}
	for _, p := range bad {
		if err := s.Put(p, []byte("x"), "msg"); err == nil {
			t.Errorf("Put(%q) accepted, want error", p)
		}
	}
}
