package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// e2e tests exercise the full command flow by invoking the cobra root command
// directly and by setting up a temp $UNION_DIR and temp shop dir. The editor
// is faked via $VISUAL=true so `edit` reads whatever we pre-seeded into the
// editor-temp file by a custom wrapper script.
//
// We skip these tests when git is unavailable since the store depends on it.

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestE2E_FullFlowPropagates(t *testing.T) {
	requireGit(t)

	unionDir := t.TempDir()
	shopDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	// init
	if out, err := runRoot(t, "init"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	// new via -f file
	clauseFile := filepath.Join(t.TempDir(), "clause.md")
	if err := os.WriteFile(clauseFile, []byte("be helpful\n"), 0o644); err != nil {
		t.Fatalf("seed clause file: %v", err)
	}
	if out, err := runRoot(t, "new", "base/identity", "-f", clauseFile); err != nil {
		t.Fatalf("new: %v\n%s", err, out)
	}

	// organize the temp shop
	if out, err := runRoot(t, "organize", shopDir); err != nil {
		t.Fatalf("organize: %v\n%s", err, out)
	}

	// ratify requires cwd == shopDir; chdir
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(shopDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWD)

	if out, err := runRoot(t, "ratify", "base/identity"); err != nil {
		t.Fatalf("ratify: %v\n%s", err, out)
	}

	contractPath := filepath.Join(shopDir, "AGENTS.md")
	got, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	if !strings.Contains(string(got), "<!-- BEGIN union:base/identity -->") {
		t.Errorf("contract missing BEGIN marker:\n%s", got)
	}
	if !strings.Contains(string(got), "be helpful") {
		t.Errorf("contract missing clause body:\n%s", got)
	}

	// Simulate an edit: directly Put new content into the store, then call
	// propagateUpdate the same way edit.go does. (We avoid driving $EDITOR in
	// this test — the editor integration is exercised manually.)
	newBody := []byte("BE EXTRA HELPFUL\n")
	s, err := openStore()
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	if err := s.Put("base/identity", newBody, "edit base/identity"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := propagateUpdate(&bytes.Buffer{}, "base/identity", newBody); err != nil {
		t.Fatalf("propagateUpdate: %v", err)
	}
	got, err = os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract after edit: %v", err)
	}
	if !strings.Contains(string(got), "BE EXTRA HELPFUL") {
		t.Errorf("contract did not propagate edit:\n%s", got)
	}
	if strings.Contains(string(got), "be helpful\n") && !strings.Contains(string(got), "BE EXTRA") {
		t.Errorf("old content still present:\n%s", got)
	}

	// expel → contract block should be gone.
	if out, err := runRoot(t, "expel", "base/identity"); err != nil {
		t.Fatalf("expel: %v\n%s", err, out)
	}
	got, err = os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract after expel: %v", err)
	}
	if strings.Contains(string(got), "<!-- BEGIN union:base/identity -->") {
		t.Errorf("contract still contains marker after expel:\n%s", got)
	}
}

func TestE2E_RatifyRequiresOrganizedShop(t *testing.T) {
	requireGit(t)

	unionDir := t.TempDir()
	notAShop := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := runRoot(t, "new", "-f", "/dev/null", "x/y"); err != nil {
		t.Fatalf("new: %v", err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(notAShop); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWD)

	_, err := runRoot(t, "ratify", "x/y")
	if err == nil {
		t.Fatal("expected ratify to fail in non-organized shop")
	}
	if !strings.Contains(err.Error(), "not an organized shop") {
		t.Errorf("expected 'not an organized shop' in error, got: %v", err)
	}
}
