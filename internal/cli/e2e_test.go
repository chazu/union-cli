package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	origWD, _ := os.Getwd()
	if err := os.Chdir(shopDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWD)

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
	origWD, _ := os.Getwd()
	os.Chdir(shopDir)
	defer os.Chdir(origWD)
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

	origWD, _ := os.Getwd()
	os.Chdir(notAShop)
	defer os.Chdir(origWD)

	_, err := runRoot(t, "ratify", "default:x/y")
	if err == nil {
		t.Fatal("expected ratify to fail in non-organized shop")
	}
	if !strings.Contains(err.Error(), "not an organized shop") {
		t.Errorf("expected 'not an organized shop' in error, got: %v", err)
	}
}
