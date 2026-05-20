package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/store"
)

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

func TestE2E_InitCreatesDefaultStore(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if out, err := runRoot(t, "init"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(unionDir, "stores", "default", ".git")); err != nil {
		t.Fatalf("default store missing: %v", err)
	}
}

func TestE2E_FullFlowPropagatesAcrossStores(t *testing.T) {
	requireGit(t)

	unionDir := t.TempDir()
	shopDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := runRoot(t, "store", "add", "personal"); err != nil {
		t.Fatalf("store add personal: %v", err)
	}

	a := filepath.Join(t.TempDir(), "a.md")
	b := filepath.Join(t.TempDir(), "b.md")
	if err := os.WriteFile(a, []byte("be helpful\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("voice is terse\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "new", "default:base/identity", "-f", a); err != nil {
		t.Fatalf("new default: %v", err)
	}
	if _, err := runRoot(t, "new", "personal:writing/voice", "-f", b); err != nil {
		t.Fatalf("new personal: %v", err)
	}

	if _, err := runRoot(t, "organize", shopDir); err != nil {
		t.Fatalf("organize: %v", err)
	}
	t.Chdir(shopDir)

	if _, err := runRoot(t, "ratify", "default:base/identity"); err != nil {
		t.Fatalf("ratify default: %v", err)
	}
	if _, err := runRoot(t, "ratify", "personal:writing/voice"); err != nil {
		t.Fatalf("ratify personal: %v", err)
	}

	contractPath := filepath.Join(shopDir, "AGENTS.md")
	got, _ := os.ReadFile(contractPath)
	if !strings.Contains(string(got), "<!-- BEGIN union:default:base/identity -->") {
		t.Errorf("missing default marker:\n%s", got)
	}
	if !strings.Contains(string(got), "<!-- BEGIN union:personal:writing/voice -->") {
		t.Errorf("missing personal marker:\n%s", got)
	}

	if _, err := runRoot(t, "new", "default:base/identity", "-f", a); err == nil {
		t.Fatal("re-'new' should fail; clause already exists")
	}

	out, err := runRoot(t, "show", "personal:writing/voice")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !strings.Contains(out, "voice is terse") {
		t.Errorf("show returned %q", out)
	}

	if _, err := runRoot(t, "expel", "personal:writing/voice"); err != nil {
		t.Fatalf("expel: %v", err)
	}
	got, _ = os.ReadFile(contractPath)
	if strings.Contains(string(got), "<!-- BEGIN union:personal:writing/voice -->") {
		t.Errorf("personal marker still present after expel:\n%s", got)
	}
	if !strings.Contains(string(got), "<!-- BEGIN union:default:base/identity -->") {
		t.Errorf("default marker was wrongly removed:\n%s", got)
	}
}

func TestE2E_StoreRemoveRefusedWhenRatified(t *testing.T) {
	requireGit(t)

	unionDir := t.TempDir()
	shopDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "store", "add", "work"); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "x.md")
	os.WriteFile(f, []byte("x\n"), 0o644)
	if _, err := runRoot(t, "new", "work:ops/deploy", "-f", f); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "organize", shopDir); err != nil {
		t.Fatal(err)
	}
	t.Chdir(shopDir)
	if _, err := runRoot(t, "ratify", "work:ops/deploy"); err != nil {
		t.Fatal(err)
	}
	_, err := runRoot(t, "store", "remove", "work")
	if err == nil {
		t.Fatal("expected store remove to be refused")
	}
	if !strings.Contains(err.Error(), "still ratified") {
		t.Errorf("expected 'still ratified' in error, got: %v", err)
	}
}

func TestE2E_StoreRemotePushPull(t *testing.T) {
	requireGit(t)

	unionDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}

	remote := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", "-q", remote)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bare init: %v\n%s", err, out)
	}

	if _, err := runRoot(t, "store", "remote", "add", "default", "origin", remote); err != nil {
		t.Fatalf("remote add: %v", err)
	}
	out, err := runRoot(t, "store", "remote", "list", "default")
	if err != nil {
		t.Fatalf("remote list: %v", err)
	}
	if !strings.Contains(out, "origin") {
		t.Errorf("remote list missing origin:\n%s", out)
	}

	f := filepath.Join(t.TempDir(), "c.md")
	os.WriteFile(f, []byte("hi\n"), 0o644)
	if _, err := runRoot(t, "new", "default:c/d", "-f", f); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "store", "push", "default", "origin"); err != nil {
		t.Fatalf("push: %v", err)
	}
}

func TestE2E_RatifyRequiresOrganizedShop(t *testing.T) {
	requireGit(t)

	unionDir := t.TempDir()
	notAShop := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "y.md")
	os.WriteFile(f, []byte("y"), 0o644)
	if _, err := runRoot(t, "new", "default:x/y", "-f", f); err != nil {
		t.Fatal(err)
	}

	t.Chdir(notAShop)

	_, err := runRoot(t, "ratify", "default:x/y")
	if err == nil {
		t.Fatal("expected ratify to fail in non-organized shop")
	}
	if !strings.Contains(err.Error(), "not an organized shop") {
		t.Errorf("expected 'not an organized shop' in error, got: %v", err)
	}
}

func TestE2E_VerifyCatchesDrift(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	shopDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "x.md")
	os.WriteFile(f, []byte("original\n"), 0o644)
	if _, err := runRoot(t, "new", "default:a/b", "-f", f); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "organize", shopDir); err != nil {
		t.Fatal(err)
	}
	t.Chdir(shopDir)
	if _, err := runRoot(t, "ratify", "default:a/b"); err != nil {
		t.Fatal(err)
	}

	// Verify should pass initially.
	out, err := runRoot(t, "verify")
	if err != nil {
		t.Fatalf("verify should pass: %v\n%s", err, out)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("expected OK, got: %s", out)
	}

	// Manually corrupt the contract to introduce drift.
	contractPath := filepath.Join(shopDir, "AGENTS.md")
	contract, _ := os.ReadFile(contractPath)
	corrupted := strings.Replace(string(contract), "original", "TAMPERED", 1)
	os.WriteFile(contractPath, []byte(corrupted), 0o644)

	out, err = runRoot(t, "verify")
	if err == nil {
		t.Fatal("verify should fail after drift")
	}
	if !strings.Contains(out, "DRIFT") {
		t.Errorf("expected DRIFT in output, got: %s", out)
	}
}

func TestE2E_SyncRepairsDrift(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	shopDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "x.md")
	os.WriteFile(f, []byte("original\n"), 0o644)
	if _, err := runRoot(t, "new", "default:a/b", "-f", f); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "organize", shopDir); err != nil {
		t.Fatal(err)
	}
	t.Chdir(shopDir)
	if _, err := runRoot(t, "ratify", "default:a/b"); err != nil {
		t.Fatal(err)
	}

	// Introduce drift.
	contractPath := filepath.Join(shopDir, "AGENTS.md")
	contract, _ := os.ReadFile(contractPath)
	drifted := strings.Replace(string(contract), "original", "DRIFTED", 1)
	os.WriteFile(contractPath, []byte(drifted), 0o644)

	out, err := runRoot(t, "sync")
	if err != nil {
		t.Fatalf("sync: %v\n%s", err, out)
	}
	if !strings.Contains(out, "synced") {
		t.Errorf("expected 'synced' in output, got: %s", out)
	}

	// Verify should pass now.
	out, err = runRoot(t, "verify")
	if err != nil {
		t.Fatalf("verify should pass after sync: %v\n%s", err, out)
	}
}

func TestE2E_SearchFindsMatches(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "x.md")
	os.WriteFile(f, []byte("be helpful and direct\n"), 0o644)
	if _, err := runRoot(t, "new", "default:style/voice", "-f", f); err != nil {
		t.Fatal(err)
	}

	out, err := runRoot(t, "search", "helpful")
	if err != nil {
		t.Fatalf("search: %v\n%s", err, out)
	}
	if !strings.Contains(out, "default:style/voice") {
		t.Errorf("expected match, got: %s", out)
	}

	out, err = runRoot(t, "search", "NONEXISTENT")
	if err != nil {
		t.Fatalf("search no match: %v", err)
	}
	if !strings.Contains(out, "no matches") {
		t.Errorf("expected 'no matches', got: %s", out)
	}
}

func TestE2E_OrphansFindsUnratified(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	shopDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	f1 := filepath.Join(t.TempDir(), "a.md")
	f2 := filepath.Join(t.TempDir(), "b.md")
	os.WriteFile(f1, []byte("a\n"), 0o644)
	os.WriteFile(f2, []byte("b\n"), 0o644)
	if _, err := runRoot(t, "new", "default:x/a", "-f", f1); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "new", "default:x/b", "-f", f2); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "organize", shopDir); err != nil {
		t.Fatal(err)
	}
	t.Chdir(shopDir)
	if _, err := runRoot(t, "ratify", "default:x/a"); err != nil {
		t.Fatal(err)
	}

	out, err := runRoot(t, "orphans")
	if err != nil {
		t.Fatalf("orphans: %v\n%s", err, out)
	}
	if !strings.Contains(out, "default:x/b") {
		t.Errorf("expected x/b as orphan, got: %s", out)
	}
	if strings.Contains(out, "default:x/a") {
		t.Errorf("x/a should not be orphan: %s", out)
	}
}

func TestE2E_WhichShowsPaths(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	out, err := runRoot(t, "which")
	if err != nil {
		t.Fatalf("which: %v\n%s", err, out)
	}
	if !strings.Contains(out, unionDir) {
		t.Errorf("which should contain union dir, got: %s", out)
	}
	if !strings.Contains(out, "shops file") {
		t.Errorf("which should mention shops file, got: %s", out)
	}
}

func TestE2E_StatusShowsSummary(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "x.md")
	os.WriteFile(f, []byte("x\n"), 0o644)
	if _, err := runRoot(t, "new", "default:a/b", "-f", f); err != nil {
		t.Fatal(err)
	}

	out, err := runRoot(t, "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if !strings.Contains(out, "1 store(s)") {
		t.Errorf("expected store count, got: %s", out)
	}
	if !strings.Contains(out, "1 clause(s)") {
		t.Errorf("expected clause count, got: %s", out)
	}
}

func TestE2E_RenameRewritesMarkers(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	shopDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "x.md")
	os.WriteFile(f, []byte("content here\n"), 0o644)
	if _, err := runRoot(t, "new", "default:old/name", "-f", f); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "organize", shopDir); err != nil {
		t.Fatal(err)
	}
	t.Chdir(shopDir)
	if _, err := runRoot(t, "ratify", "default:old/name"); err != nil {
		t.Fatal(err)
	}

	out, err := runRoot(t, "rename", "default:old/name", "default:new/name")
	if err != nil {
		t.Fatalf("rename: %v\n%s", err, out)
	}

	// Old clause gone, new clause exists.
	if _, err := runRoot(t, "show", "default:new/name"); err != nil {
		t.Fatalf("new clause should exist: %v", err)
	}
	if _, err := runRoot(t, "show", "default:old/name"); err == nil {
		t.Fatal("old clause should be gone")
	}

	// Contract should have new markers, not old.
	contract, _ := os.ReadFile(filepath.Join(shopDir, "AGENTS.md"))
	if strings.Contains(string(contract), "old/name") {
		t.Errorf("old marker still present:\n%s", contract)
	}
	if !strings.Contains(string(contract), "new/name") {
		t.Errorf("new marker missing:\n%s", contract)
	}
	if !strings.Contains(string(contract), "content here") {
		t.Errorf("clause body missing:\n%s", contract)
	}
}

func TestE2E_DoubleRatifyIsIdempotent(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	shopDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "x.md")
	os.WriteFile(f, []byte("clause body\n"), 0o644)
	if _, err := runRoot(t, "new", "default:x/y", "-f", f); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "organize", shopDir); err != nil {
		t.Fatal(err)
	}
	t.Chdir(shopDir)
	if _, err := runRoot(t, "ratify", "default:x/y"); err != nil {
		t.Fatal(err)
	}

	before, _ := os.ReadFile(filepath.Join(shopDir, "AGENTS.md"))

	if _, err := runRoot(t, "ratify", "default:x/y"); err != nil {
		t.Fatalf("second ratify should succeed: %v", err)
	}

	after, _ := os.ReadFile(filepath.Join(shopDir, "AGENTS.md"))
	if string(before) != string(after) {
		t.Errorf("double ratify changed contract:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestE2E_CustomContract(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	shopDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "x.md")
	os.WriteFile(f, []byte("clause\n"), 0o644)
	if _, err := runRoot(t, "new", "default:a/b", "-f", f); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "organize", shopDir, "--contract", "CLAUDE.md"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(shopDir)
	if _, err := runRoot(t, "ratify", "default:a/b"); err != nil {
		t.Fatal(err)
	}

	// Should write to CLAUDE.md, not AGENTS.md.
	if _, err := os.Stat(filepath.Join(shopDir, "CLAUDE.md")); err != nil {
		t.Errorf("CLAUDE.md should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(shopDir, "AGENTS.md")); err == nil {
		t.Error("AGENTS.md should NOT exist when --contract=CLAUDE.md")
	}
}

func TestE2E_RenameCrossStoreFails(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "store", "add", "other"); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "x.md")
	os.WriteFile(f, []byte("x\n"), 0o644)
	if _, err := runRoot(t, "new", "default:a/b", "-f", f); err != nil {
		t.Fatal(err)
	}

	_, err := runRoot(t, "rename", "default:a/b", "other:a/b")
	if err == nil {
		t.Fatal("cross-store rename should fail")
	}
	if !strings.Contains(err.Error(), "same store") {
		t.Errorf("expected 'same store' error, got: %v", err)
	}
}

func TestE2E_EditPropagatesAcrossStoresIsolated(t *testing.T) {
	requireGit(t)
	unionDir := t.TempDir()
	shopDir := t.TempDir()
	t.Setenv("UNION_DIR", unionDir)

	if _, err := runRoot(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "store", "add", "personal"); err != nil {
		t.Fatal(err)
	}

	a := filepath.Join(t.TempDir(), "a.md")
	b := filepath.Join(t.TempDir(), "b.md")
	os.WriteFile(a, []byte("A1\n"), 0o644)
	os.WriteFile(b, []byte("B1\n"), 0o644)

	if _, err := runRoot(t, "new", "default:x/a", "-f", a); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "new", "personal:x/b", "-f", b); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "organize", shopDir); err != nil {
		t.Fatal(err)
	}

	t.Chdir(shopDir)

	if _, err := runRoot(t, "ratify", "default:x/a"); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "ratify", "personal:x/b"); err != nil {
		t.Fatal(err)
	}

	// Simulate an edit to personal:x/b by Put + propagateUpdate.
	newBody := []byte("B2_EDITED\n")
	unionRoot, err := paths.UnionDir()
	if err != nil {
		t.Fatal(err)
	}
	s, err := store.OpenNamed(unionRoot, "personal")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put("x/b", newBody, "edit"); err != nil {
		t.Fatal(err)
	}
	if err := propagateUpdate(&bytes.Buffer{}, "personal:x/b", newBody); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(filepath.Join(shopDir, "AGENTS.md"))
	if !strings.Contains(string(got), "B2_EDITED") {
		t.Errorf("edit didn't propagate:\n%s", got)
	}
	if !strings.Contains(string(got), "A1") {
		t.Errorf("default block was touched:\n%s", got)
	}
	if strings.Contains(string(got), "B1\n") {
		t.Errorf("old personal content still present:\n%s", got)
	}
}
