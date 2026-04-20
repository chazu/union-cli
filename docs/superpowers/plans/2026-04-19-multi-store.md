# Multi-Store Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a user keep multiple independent clause stores under `$UNION_DIR/stores/<name>`, each its own git repo with remotes; reference clauses across stores via `store:path` syntax; add `union store ...` commands for lifecycle and git sync.

**Architecture:** Each store is an independent git repo at `$UNION_DIR/stores/<name>/`. A new `internal/qpath` package owns the `store:path` grammar; markers in contract files embed the full qualified form (`<!-- BEGIN union:personal:writing/voice -->`). The existing single-`Store` type remains single-repo; CLI commands resolve the right store from the clause path's `store:` prefix. A new `union store ...` command tree adds lifecycle (`add`/`list`/`remove`) and git ops (`remote add/remove/list`, `push`/`pull`/`fetch`/`status`).

**Tech Stack:** Go, cobra, BurntSushi/toml, git CLI (via `exec.Command`).

---

## Spec Reference

Implements `docs/superpowers/specs/2026-04-19-multi-store-design.md`.

This is a **breaking** change to the on-disk layout and marker format. There are no existing users, so no migration.

## File Structure

**New files:**

- `internal/qpath/qpath.go` — qualified-path type and parser
- `internal/qpath/qpath_test.go`
- `internal/cli/store.go` — `union store` command tree

**Modified files:**

- `internal/paths/paths.go` — add `StoresDir`, `StoreDir`; remove `ClausesDir`
- `internal/paths/paths_test.go` — cover the new helpers
- `internal/store/store.go` — add `InitNamed`, `OpenNamed`, `ListStores`, and git ops (`RemoteAdd`, `RemoteRemove`, `Remotes`, `Push`, `Pull`, `Fetch`, `Status`); remove `Init`/`Open` usages from within CLI (but keep exports)
- `internal/store/store_test.go` — tests for new helpers and git ops
- `internal/shop/markers.go` — qualified-only marker grammar
- `internal/shop/markers_test.go` — updated fixtures
- `internal/cli/init.go` — init creates `stores/default/` (or `stores/<name>` with arg)
- `internal/cli/new.go`, `show.go`, `edit.go`, `expel.go`, `ratify.go`, `strike.go`, `clauses.go`, `contract.go`, `propagate.go` — switch to qualified clause paths
- `internal/cli/root.go` — register `newStoreCmd`
- `internal/cli/e2e_test.go` — rewritten for multi-store
- `README.md` — document new layout and commands

---

## Task 1: qpath package

**Files:**
- Create: `internal/qpath/qpath.go`
- Create: `internal/qpath/qpath_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/qpath/qpath_test.go`:

```go
package qpath

import "testing"

func TestParse_Valid(t *testing.T) {
	cases := []struct {
		in        string
		wantStore string
		wantPath  string
	}{
		{"personal:writing/voice", "personal", "writing/voice"},
		{"default:x", "default", "x"},
		{"a1_b-c:deep/path/to/clause", "a1_b-c", "deep/path/to/clause"},
	}
	for _, tc := range cases {
		q, err := Parse(tc.in)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tc.in, err)
			continue
		}
		if q.Store != tc.wantStore || q.Path != tc.wantPath {
			t.Errorf("Parse(%q) = {%q,%q}, want {%q,%q}", tc.in, q.Store, q.Path, tc.wantStore, tc.wantPath)
		}
		if q.String() != tc.in {
			t.Errorf("String() = %q, want %q", q.String(), tc.in)
		}
	}
}

func TestParse_Invalid(t *testing.T) {
	bad := []string{
		"",
		"no-colon",
		":empty-store",
		"empty-path:",
		"Bad:store",
		"has space:path",
		"store:/abs",
		"store:../escape",
		"store:a//b",
		"store:has space",
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", s)
		}
	}
}

func TestValidateStoreName(t *testing.T) {
	ok := []string{"a", "default", "personal", "work1", "a_b-c", "0abc"}
	for _, s := range ok {
		if err := ValidateStoreName(s); err != nil {
			t.Errorf("ValidateStoreName(%q) error: %v", s, err)
		}
	}
	bad := []string{"", "A", "has space", "has/slash", "has:colon", "-leading"}
	for _, s := range bad {
		if err := ValidateStoreName(s); err == nil {
			t.Errorf("ValidateStoreName(%q) succeeded, want error", s)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/qpath/...`
Expected: FAIL (package does not exist)

- [ ] **Step 3: Implement the package**

`internal/qpath/qpath.go`:

```go
// Package qpath parses and validates qualified clause paths of the form
// "<store>:<clause-path>".
package qpath

import (
	"fmt"
	"regexp"
	"strings"
)

// Qualified is a store-qualified clause path.
type Qualified struct {
	Store string
	Path  string
}

// String returns the canonical "store:path" form.
func (q Qualified) String() string { return q.Store + ":" + q.Path }

var storeNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ValidateStoreName reports whether name matches [a-z0-9][a-z0-9_-]*.
func ValidateStoreName(name string) error {
	if !storeNameRE.MatchString(name) {
		return fmt.Errorf("invalid store name %q: must match [a-z0-9][a-z0-9_-]*", name)
	}
	return nil
}

// ValidateClausePath enforces the clause-path subset of the path grammar.
// (Same rules the store already applies: no '', '..', leading '/', ' ', '//'.)
func ValidateClausePath(p string) error {
	if p == "" {
		return fmt.Errorf("clause path is empty")
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("clause path must be relative: %q", p)
	}
	if strings.Contains(p, "..") {
		return fmt.Errorf("clause path may not contain '..': %q", p)
	}
	if strings.Contains(p, " ") {
		return fmt.Errorf("clause path may not contain spaces: %q", p)
	}
	if strings.Contains(p, "//") {
		return fmt.Errorf("clause path may not contain '//': %q", p)
	}
	return nil
}

// Parse splits s on the first ':' and validates both halves.
func Parse(s string) (Qualified, error) {
	i := strings.IndexByte(s, ':')
	if i < 0 {
		return Qualified{}, fmt.Errorf("clause path must be qualified as <store>:<path>, got %q", s)
	}
	store, path := s[:i], s[i+1:]
	if err := ValidateStoreName(store); err != nil {
		return Qualified{}, err
	}
	if err := ValidateClausePath(path); err != nil {
		return Qualified{}, err
	}
	return Qualified{Store: store, Path: path}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/qpath/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/qpath
git commit -m "feat(qpath): add qualified clause path package"
```

---

## Task 2: paths helpers for multi-store layout

**Files:**
- Modify: `internal/paths/paths.go`
- Modify: `internal/paths/paths_test.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/paths/paths_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/paths/...`
Expected: FAIL (undefined `StoresDir`, `StoreDir`)

- [ ] **Step 3: Implement**

Rewrite `internal/paths/paths.go`:

```go
// Package paths resolves the on-disk layout for the union store.
package paths

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chazu/union/internal/qpath"
)

// UnionDir returns the root directory of the union store.
// Honors $UNION_DIR; defaults to ~/.union.
func UnionDir() (string, error) {
	if v := os.Getenv("UNION_DIR"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".union"), nil
}

// StoresDir returns $UNION_DIR/stores.
func StoresDir() (string, error) {
	root, err := UnionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "stores"), nil
}

// StoreDir returns $UNION_DIR/stores/<name>, validating the name.
func StoreDir(name string) (string, error) {
	if err := qpath.ValidateStoreName(name); err != nil {
		return "", err
	}
	root, err := StoresDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

// ShopsFile returns $UNION_DIR/shops.toml.
func ShopsFile() (string, error) {
	root, err := UnionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "shops.toml"), nil
}
```

Note: `ClausesDir` is removed. Any caller will now fail to compile — Task 7 will fix them up; for now we accept a temporarily broken build and fix callers below. *But* `cmd/union` and `internal/cli` both still reference `ClausesDir`. **Keep `ClausesDir` temporarily** by adding a deprecated stub returning `filepath.Join($UNION_DIR, "clauses")` so the tree compiles until Task 7 removes the last caller.

Append to `paths.go`:

```go
// ClausesDir is retained temporarily so existing call sites compile during
// the multi-store migration. Removed in Task 7.
//
// Deprecated: use StoreDir(name) + "/clauses" instead.
func ClausesDir() (string, error) {
	root, err := UnionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "clauses"), nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/paths/... ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/paths
git commit -m "feat(paths): add StoresDir and StoreDir helpers"
```

---

## Task 3: store package multi-store helpers

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/store/store_test.go`:

```go
func TestInitNamed_CreatesLayout(t *testing.T) {
	requireGit(t)
	union := t.TempDir()
	s, err := InitNamed(union, "default")
	if err != nil {
		t.Fatalf("InitNamed: %v", err)
	}
	wantRoot := filepath.Join(union, "stores", "default")
	if s.Root() != wantRoot {
		t.Errorf("Root = %q, want %q", s.Root(), wantRoot)
	}
	if _, err := os.Stat(filepath.Join(wantRoot, ".git")); err != nil {
		t.Errorf(".git missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wantRoot, "clauses", ".gitkeep")); err != nil {
		t.Errorf(".gitkeep missing: %v", err)
	}
}

func TestInitNamed_RefusesInvalidName(t *testing.T) {
	requireGit(t)
	if _, err := InitNamed(t.TempDir(), "Bad Name"); err == nil {
		t.Fatal("expected error on bad store name")
	}
}

func TestOpenNamed_RoundTrip(t *testing.T) {
	requireGit(t)
	union := t.TempDir()
	if _, err := InitNamed(union, "work"); err != nil {
		t.Fatalf("InitNamed: %v", err)
	}
	s, err := OpenNamed(union, "work")
	if err != nil {
		t.Fatalf("OpenNamed: %v", err)
	}
	if err := s.Put("x/y", []byte("hello"), "new"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, _ := s.Get("x/y")
	if string(got) != "hello" {
		t.Errorf("Get = %q", got)
	}
}

func TestListStores(t *testing.T) {
	requireGit(t)
	union := t.TempDir()
	for _, name := range []string{"default", "personal", "work"} {
		if _, err := InitNamed(union, name); err != nil {
			t.Fatalf("InitNamed %s: %v", name, err)
		}
	}
	got, err := ListStores(union)
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"default", "personal", "work"}) {
		t.Errorf("got %v", got)
	}
}

func TestListStores_Empty(t *testing.T) {
	union := t.TempDir()
	got, err := ListStores(union)
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want []", got)
	}
}
```

Add imports as needed: `path/filepath`, `reflect`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/store/...`
Expected: FAIL (undefined `InitNamed`, `OpenNamed`, `ListStores`)

- [ ] **Step 3: Implement**

Append to `internal/store/store.go`:

```go
// InitNamed creates $unionDir/stores/<name>/ and initializes it as a store.
func InitNamed(unionDir, name string) (*Store, error) {
	if err := validateStoreName(name); err != nil {
		return nil, err
	}
	dir := filepath.Join(unionDir, "stores", name)
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return nil, fmt.Errorf("create stores dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	return Init(dir)
}

// OpenNamed opens the store at $unionDir/stores/<name>/.
func OpenNamed(unionDir, name string) (*Store, error) {
	if err := validateStoreName(name); err != nil {
		return nil, err
	}
	dir := filepath.Join(unionDir, "stores", name)
	return Open(dir)
}

// ListStores scans $unionDir/stores/ for subdirectories containing .git,
// returning their names sorted.
func ListStores(unionDir string) ([]string, error) {
	storesDir := filepath.Join(unionDir, "stores")
	entries, err := os.ReadDir(storesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read stores dir: %w", err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gitPath := filepath.Join(storesDir, e.Name(), ".git")
		if _, err := os.Stat(gitPath); err != nil {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out, nil
}

// validateStoreName mirrors qpath.ValidateStoreName without importing qpath
// (the store package is deeper in the dependency graph).
var storeNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func validateStoreName(name string) error {
	if !storeNameRE.MatchString(name) {
		return fmt.Errorf("invalid store name %q: must match [a-z0-9][a-z0-9_-]*", name)
	}
	return nil
}
```

Add `"regexp"` to imports.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/store/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store
git commit -m "feat(store): add InitNamed, OpenNamed, ListStores"
```

---

## Task 4: store package git operations

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/store/store_test.go`:

```go
func bareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", "-q", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	return dir
}

func TestRemoteAddListRemove(t *testing.T) {
	s := newTestStore(t)
	if err := s.RemoteAdd("origin", "https://example.com/r.git"); err != nil {
		t.Fatalf("RemoteAdd: %v", err)
	}
	rs, err := s.Remotes()
	if err != nil {
		t.Fatalf("Remotes: %v", err)
	}
	if len(rs) != 1 || rs[0].Name != "origin" || rs[0].URL != "https://example.com/r.git" {
		t.Errorf("got %+v", rs)
	}
	if err := s.RemoteRemove("origin"); err != nil {
		t.Fatalf("RemoteRemove: %v", err)
	}
	rs, _ = s.Remotes()
	if len(rs) != 0 {
		t.Errorf("expected no remotes after remove, got %+v", rs)
	}
}

func TestPushPullRoundTrip(t *testing.T) {
	s := newTestStore(t)
	remote := bareRepo(t)
	if err := s.RemoteAdd("origin", remote); err != nil {
		t.Fatalf("RemoteAdd: %v", err)
	}
	if err := s.Put("a/b", []byte("x"), "add a/b"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Push("origin", ""); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if err := s.Fetch("origin"); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	// Pull is a rebase no-op here; just verify it doesn't error.
	if err := s.Pull("origin", ""); err != nil {
		t.Fatalf("Pull: %v", err)
	}
}

func TestStatus(t *testing.T) {
	s := newTestStore(t)
	out, err := s.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !strings.Contains(out, "##") {
		t.Errorf("expected porcelain branch line, got %q", out)
	}
}
```

Add imports as needed: `strings`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/store/...`
Expected: FAIL (undefined methods)

- [ ] **Step 3: Implement**

Append to `internal/store/store.go`:

```go
// Remote is a name/URL pair.
type Remote struct {
	Name string
	URL  string
}

// RemoteAdd runs `git remote add <name> <url>` in the store.
func (s *Store) RemoteAdd(name, url string) error {
	return s.git("remote", "add", name, url)
}

// RemoteRemove runs `git remote remove <name>`.
func (s *Store) RemoteRemove(name string) error {
	return s.git("remote", "remove", name)
}

// Remotes returns the configured remotes, sorted by name.
func (s *Store) Remotes() ([]Remote, error) {
	out, err := s.gitCapture("remote", "-v")
	if err != nil {
		return nil, err
	}
	seen := map[string]string{}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		// Format: "<name>\t<url> (fetch|push)"
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		seen[fields[0]] = fields[1]
	}
	var rs []Remote
	for n, u := range seen {
		rs = append(rs, Remote{Name: n, URL: u})
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].Name < rs[j].Name })
	return rs, nil
}

// Push runs `git push` in the store. Empty remote/branch defers to git defaults.
func (s *Store) Push(remote, branch string) error {
	args := []string{"push"}
	if remote != "" {
		args = append(args, remote)
	}
	if branch != "" {
		args = append(args, branch)
	}
	return s.gitStream(args...)
}

// Pull runs `git pull --rebase` in the store.
func (s *Store) Pull(remote, branch string) error {
	args := []string{"pull", "--rebase"}
	if remote != "" {
		args = append(args, remote)
	}
	if branch != "" {
		args = append(args, branch)
	}
	return s.gitStream(args...)
}

// Fetch runs `git fetch [remote]`.
func (s *Store) Fetch(remote string) error {
	args := []string{"fetch"}
	if remote != "" {
		args = append(args, remote)
	}
	return s.gitStream(args...)
}

// Status returns `git status --short --branch` output.
func (s *Store) Status() (string, error) {
	return s.gitCapture("status", "--short", "--branch")
}

// gitCapture runs git and returns stdout.
func (s *Store) gitCapture(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.root
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return out.String(), nil
}

// gitStream runs git with stdout/stderr wired to the user's terminal so
// progress output from push/pull/fetch surfaces. Used for long-running ops.
func (s *Store) gitStream(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/store/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store
git commit -m "feat(store): add remote/push/pull/fetch/status operations"
```

---

## Task 5: qualified-only marker grammar

**Files:**
- Modify: `internal/shop/markers.go`
- Modify: `internal/shop/markers_test.go`

The marker grammar accepts *only* qualified paths now. `block.Path` holds the full `"store:path"` string so propagation and insertion keep working with a minimal type change.

- [ ] **Step 1: Update tests to use qualified paths**

Rewrite `internal/shop/markers_test.go` so every fixture uses qualified paths. For each existing test, change:

- `"<!-- BEGIN union:base/identity -->"` → `"<!-- BEGIN union:default:base/identity -->"`
- `"<!-- BEGIN union:x -->"` → `"<!-- BEGIN union:default:x -->"`
- `Path: "base/identity"` → `Path: "default:base/identity"`
- `Path: "x"` → `Path: "default:x"`
- `"<!-- BEGIN union:lang/go -->"` → `"<!-- BEGIN union:default:lang/go -->"`
- `"<!-- BEGIN union:foo -->"` → `"<!-- BEGIN union:default:foo -->"` (orphan tests)
- `"<!-- END union:bar -->"` → `"<!-- END union:default:bar -->"` (mismatch test)

Add one new test:

```go
func TestParseContract_RejectsLegacyUnqualified(t *testing.T) {
	in := []byte("<!-- BEGIN union:base/identity -->\nhi\n<!-- END union:base/identity -->\n")
	if _, err := ParseContract(in); err == nil {
		t.Fatal("expected error on unqualified marker")
	}
}

func TestParseContract_CrossStore(t *testing.T) {
	in := []byte("<!-- BEGIN union:personal:a -->\n1\n<!-- END union:personal:a -->\n<!-- BEGIN union:work:b -->\n2\n<!-- END union:work:b -->\n")
	blocks, err := ParseContract(in)
	if err != nil {
		t.Fatalf("ParseContract: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("want 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Path != "personal:a" || blocks[1].Path != "work:b" {
		t.Errorf("paths = %q, %q", blocks[0].Path, blocks[1].Path)
	}
}
```

- [ ] **Step 2: Run tests to verify the qualified tests fail**

Run: `go test ./internal/shop/...`
Expected: FAIL — markers.go accepts unqualified tokens, so `RejectsLegacyUnqualified` and `CrossStore` don't behave correctly yet.

- [ ] **Step 3: Implement qualified-only parsing**

Rewrite the marker helpers in `internal/shop/markers.go`:

Replace the entire top region with:

```go
package shop

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

const (
	beginPrefix = "<!-- BEGIN union:"
	endPrefix   = "<!-- END union:"
	markerSfx   = " -->"
)

// qualifiedRE matches "<store>:<clause-path>" where the store is a valid
// store name and the clause path has no whitespace. Clause-path structural
// rules (../, //, leading /) are enforced by callers via qpath.Parse when
// needed; the marker parser just needs an unambiguous grammar.
var qualifiedRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*:[^\s]+$`)
```

Replace `parseBegin` and `parseEnd`:

```go
func parseBegin(line string) (string, bool) {
	line = strings.TrimRight(line, " \t")
	if !strings.HasPrefix(line, beginPrefix) || !strings.HasSuffix(line, markerSfx) {
		return "", false
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, beginPrefix), markerSfx))
	if !qualifiedRE.MatchString(inner) {
		return "", false
	}
	return inner, true
}

func parseEnd(line string) (string, bool) {
	line = strings.TrimRight(line, " \t")
	if !strings.HasPrefix(line, endPrefix) || !strings.HasSuffix(line, markerSfx) {
		return "", false
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, endPrefix), markerSfx))
	if !qualifiedRE.MatchString(inner) {
		return "", false
	}
	return inner, true
}
```

Important: an unqualified BEGIN marker will now be reported as an orphan END / unterminated because `parseBegin` rejects it. The `TestParseContract_RejectsLegacyUnqualified` test relies on this: the BEGIN is not recognized as a BEGIN, which lets the scanner fall through and — here we need the scanner to error. Simplest: when a line starts with `beginPrefix` but `qualifiedRE` fails, return an error immediately.

Add a stricter check at the top of the `ParseContract` loop:

```go
// inside ParseContract, before the `if path, ok := parseBegin(line); ok {` block:
if strings.HasPrefix(strings.TrimSpace(line), beginPrefix) {
    _, ok := parseBegin(line)
    if !ok {
        return nil, fmt.Errorf("invalid BEGIN marker at line %d: expected <store>:<path>, got: %s", i+1, strings.TrimSpace(line))
    }
}
if strings.HasPrefix(strings.TrimSpace(line), endPrefix) {
    _, ok := parseEnd(line)
    if !ok {
        return nil, fmt.Errorf("invalid END marker at line %d: expected <store>:<path>, got: %s", i+1, strings.TrimSpace(line))
    }
}
```

Delete `parseBeginPath` (it is no longer needed in the nested-begin message; inline call to `parseBegin` there — or leave it as-is if still used; verify via compile).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/shop/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/shop
git commit -m "feat(shop): require qualified store:path in markers"
```

---

## Task 6: init creates stores/<name>

**Files:**
- Modify: `internal/cli/init.go`

- [ ] **Step 1: Rewrite init**

Replace `internal/cli/init.go`:

```go
package cli

import (
	"fmt"
	"os"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize the union root and a first store (default: 'default').",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "default"
			if len(args) == 1 {
				name = args[0]
			}
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(unionDir, 0o755); err != nil {
				return fmt.Errorf("create %s: %w", unionDir, err)
			}
			shopsPath, err := paths.ShopsFile()
			if err != nil {
				return err
			}
			if _, err := os.Stat(shopsPath); os.IsNotExist(err) {
				if err := os.WriteFile(shopsPath, []byte("# union shops registry\n"), 0o644); err != nil {
					return fmt.Errorf("seed shops.toml: %w", err)
				}
			}
			s, err := store.InitNamed(unionDir, name)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "initialized store %q at %s\n", name, s.Root())
			return nil
		},
	}
}
```

- [ ] **Step 2: Commit** (tests will be fixed in later tasks)

```bash
git add internal/cli/init.go
git commit -m "feat(cli): init creates stores/<name> (default: 'default')"
```

Note: the full build/test will only pass again after Task 7. That's acceptable — these tasks are sequentially dependent.

---

## Task 7: CLI commands take qualified clause paths

**Files:**
- Modify: `internal/cli/new.go`
- Modify: `internal/cli/show.go`
- Modify: `internal/cli/edit.go`
- Modify: `internal/cli/expel.go`
- Modify: `internal/cli/ratify.go`
- Modify: `internal/cli/strike.go`
- Modify: `internal/cli/contract.go` (no code change — its output already uses `block.Path`, which now carries qualified form)
- Modify: `internal/paths/paths.go` (remove the deprecated `ClausesDir`)

This task introduces a helper `openStoreForQPath(q qpath.Qualified) (*store.Store, error)` that every command uses.

- [ ] **Step 1: Add the helper**

Edit `internal/cli/new.go` — replace the `openStore` helper at the bottom with a name-based helper. Actually, move the helper to `root.go` so all commands share it.

Add to `internal/cli/root.go`:

```go
import (
	// existing imports...
	"github.com/chazu/union/internal/qpath"
	"github.com/chazu/union/internal/store"
)

func openStoreFor(q qpath.Qualified) (*store.Store, error) {
	unionDir, err := paths.UnionDir()
	if err != nil {
		return nil, err
	}
	s, err := store.OpenNamed(unionDir, q.Store)
	if err != nil {
		return nil, fmt.Errorf("no such store: %s (run 'union store list')", q.Store)
	}
	return s, nil
}
```

Delete the old `openStore` helper in `internal/cli/new.go`.

- [ ] **Step 2: Update `new.go`**

Replace `newNewCmd`'s RunE:

```go
RunE: func(cmd *cobra.Command, args []string) error {
	q, err := qpath.Parse(args[0])
	if err != nil {
		return err
	}
	s, err := openStoreFor(q)
	if err != nil {
		return err
	}
	if s.Has(q.Path) {
		return fmt.Errorf("clause already exists: %s (use 'union edit' to change it)", q)
	}
	body, err := readClauseInput(fromFile, q.Path)
	if err != nil {
		return err
	}
	if err := s.Put(q.Path, body, "new "+q.String()); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "created clause %s\n", q)
	return nil
},
```

Add import `"github.com/chazu/union/internal/qpath"`.

- [ ] **Step 3: Update `show.go`**

```go
RunE: func(cmd *cobra.Command, args []string) error {
	q, err := qpath.Parse(args[0])
	if err != nil {
		return err
	}
	s, err := openStoreFor(q)
	if err != nil {
		return err
	}
	body, err := s.Get(q.Path)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
},
```

- [ ] **Step 4: Update `edit.go`**

```go
RunE: func(cmd *cobra.Command, args []string) error {
	q, err := qpath.Parse(args[0])
	if err != nil {
		return err
	}
	s, err := openStoreFor(q)
	if err != nil {
		return err
	}
	cur, err := s.Get(q.Path)
	if err != nil {
		return err
	}
	body, err := openEditor(cur, q.Path)
	if err != nil {
		return err
	}
	if err := s.Put(q.Path, body, "edit "+q.String()); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "edited clause %s\n", q)
	return propagateUpdate(cmd.OutOrStdout(), q.String(), body)
},
```

- [ ] **Step 5: Update `expel.go`**

```go
RunE: func(cmd *cobra.Command, args []string) error {
	q, err := qpath.Parse(args[0])
	if err != nil {
		return err
	}
	s, err := openStoreFor(q)
	if err != nil {
		return err
	}
	if err := s.Delete(q.Path, "expel "+q.String()); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "expelled clause %s\n", q)
	return propagateRemoval(cmd.OutOrStdout(), q.String())
},
```

- [ ] **Step 6: Update `ratify.go`**

```go
RunE: func(cmd *cobra.Command, args []string) error {
	q, err := qpath.Parse(args[0])
	if err != nil {
		return err
	}
	s, err := openStoreFor(q)
	if err != nil {
		return err
	}
	if !s.Has(q.Path) {
		return fmt.Errorf("no such clause: %s. See 'union clauses' for available paths.", q)
	}
	body, err := s.Get(q.Path)
	if err != nil {
		return err
	}
	_, contractPath, err := currentShop()
	if err != nil {
		return err
	}
	cur, err := os.ReadFile(contractPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	out, err := shop.InsertClause(cur, q.String(), body)
	if err != nil {
		return err
	}
	if err := os.WriteFile(contractPath, out, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "ratified %s into %s\n", q, contractPath)
	return nil
},
```

- [ ] **Step 7: Update `strike.go`**

```go
RunE: func(cmd *cobra.Command, args []string) error {
	q, err := qpath.Parse(args[0])
	if err != nil {
		return err
	}
	_, contractPath, err := currentShop()
	if err != nil {
		return err
	}
	cur, err := os.ReadFile(contractPath)
	if err != nil {
		return err
	}
	out, err := shop.RemoveClause(cur, q.String())
	if err != nil {
		return err
	}
	if err := os.WriteFile(contractPath, out, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "struck %s from %s\n", q, contractPath)
	return nil
},
```

- [ ] **Step 8: Remove deprecated ClausesDir**

Delete the `ClausesDir` function from `internal/paths/paths.go` added in Task 2.

- [ ] **Step 9: Build and run unit tests**

Run: `go build ./... && go test ./internal/paths/... ./internal/qpath/... ./internal/shop/... ./internal/store/...`
Expected: PASS

(`internal/cli/...` e2e tests are still the old single-store shape; they'll be fixed in Task 12.)

- [ ] **Step 10: Commit**

```bash
git add internal/cli internal/paths
git commit -m "feat(cli): route clause paths through qpath and per-store resolver"
```

---

## Task 8: clauses command lists across all stores

**Files:**
- Modify: `internal/cli/clauses.go`

- [ ] **Step 1: Rewrite `clauses.go`**

```go
package cli

import (
	"fmt"
	"strings"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newClausesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clauses [<store>:<prefix>]",
		Short: "List clauses across stores (store:path form). Optional store:<prefix> filter.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			stores, err := store.ListStores(unionDir)
			if err != nil {
				return err
			}

			wantStore := ""
			wantPrefix := ""
			if len(args) == 1 {
				arg := args[0]
				i := strings.IndexByte(arg, ':')
				if i < 0 {
					return fmt.Errorf("filter must be qualified as <store>:<prefix> (got %q)", arg)
				}
				wantStore = arg[:i]
				wantPrefix = arg[i+1:]
			}

			for _, name := range stores {
				if wantStore != "" && name != wantStore {
					continue
				}
				s, err := store.OpenNamed(unionDir, name)
				if err != nil {
					return err
				}
				paths, err := s.List(wantPrefix)
				if err != nil {
					return err
				}
				for _, p := range paths {
					fmt.Fprintf(cmd.OutOrStdout(), "%s:%s\n", name, p)
				}
			}
			return nil
		},
	}
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/clauses.go
git commit -m "feat(cli): clauses lists across stores with store:path prefix filter"
```

---

## Task 9: propagation routes to correct store

**Files:**
- Modify: `internal/cli/propagate.go`

The existing `propagateUpdate`/`propagateRemoval` just mutate contract files in place. Since the marker key now includes the store name (`personal:writing/voice`), lookup by string matches on the qualified form — no store resolution is actually needed *inside* propagation for the mutation itself.

But `edit`/`expel` call it with the qualified string (Task 7 already does `q.String()`), so markers.HasClause/UpdateClause/RemoveClause just need to accept and match that string. They already do — Path is an opaque string. No change needed, but we should add a safety check.

- [ ] **Step 1: Add defensive qualified-path validation**

Edit `internal/cli/propagate.go`. At the top of `eachRatifiedShop`, add:

```go
if _, err := qpath.Parse(clausePath); err != nil {
	return err
}
```

Add import `"github.com/chazu/union/internal/qpath"`.

- [ ] **Step 2: Build and run existing tests**

Run: `go build ./... && go test ./internal/shop/... ./internal/store/... ./internal/qpath/... ./internal/paths/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/cli/propagate.go
git commit -m "feat(cli): validate qualified path at propagation boundary"
```

---

## Task 10: union store add / list / remove

**Files:**
- Create: `internal/cli/store.go`
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Create `store.go` with parent + add/list/remove**

```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/qpath"
	"github.com/chazu/union/internal/shop"
	unionstore "github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newStoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store",
		Short: "Manage clause stores (add, list, remove, remotes, push/pull).",
	}
	cmd.AddCommand(
		newStoreAddCmd(),
		newStoreListCmd(),
		newStoreRemoveCmd(),
		newStoreRemoteCmd(),
		newStorePushCmd(),
		newStorePullCmd(),
		newStoreFetchCmd(),
		newStoreStatusCmd(),
	)
	return cmd
}

func newStoreAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new store at $UNION_DIR/stores/<name>.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := qpath.ValidateStoreName(name); err != nil {
				return err
			}
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			dir := filepath.Join(unionDir, "stores", name)
			if _, err := os.Stat(dir); err == nil {
				return fmt.Errorf("store already exists: %s", name)
			}
			s, err := unionstore.InitNamed(unionDir, name)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created store %q at %s\n", name, s.Root())
			return nil
		},
	}
}

func newStoreListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stores.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			names, err := unionstore.ListStores(unionDir)
			if err != nil {
				return err
			}
			for _, n := range names {
				fmt.Fprintln(cmd.OutOrStdout(), n)
			}
			return nil
		},
	}
}

func newStoreRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a store (refused if clauses from it are ratified).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			name := args(cmd)[0]
			if err := qpath.ValidateStoreName(name); err != nil {
				return err
			}
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			dir := filepath.Join(unionDir, "stores", name)
			if _, err := os.Stat(dir); err != nil {
				return fmt.Errorf("no such store: %s", name)
			}
			offenders, err := shopsReferencingStore(name)
			if err != nil {
				return err
			}
			if len(offenders) > 0 {
				sort.Strings(offenders)
				return fmt.Errorf("refusing to remove store %q: still ratified in shop(s): %v", name, offenders)
			}
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("remove store: %w", err)
			}
			return nil
		},
	}
}

// args is a tiny helper so cobra's Args-less closure above can reach its args.
// Use cobra's idiom instead:
func args(cmd *cobra.Command) []string { return cmd.Flags().Args() }

// shopsReferencingStore returns absolute shop directories whose contracts
// contain any marker from the given store.
func shopsReferencingStore(storeName string) ([]string, error) {
	shopsPath, err := paths.ShopsFile()
	if err != nil {
		return nil, err
	}
	r, err := shop.LoadRegistry(shopsPath)
	if err != nil {
		return nil, err
	}
	var hits []string
	for _, s := range r.List() {
		contractPath := filepath.Join(s.Dir, s.Contract)
		body, err := os.ReadFile(contractPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		blocks, err := shop.ParseContract(body)
		if err != nil {
			continue // malformed contract: don't block removal on it
		}
		for _, b := range blocks {
			q, err := qpath.Parse(b.Path)
			if err != nil {
				continue
			}
			if q.Store == storeName {
				hits = append(hits, s.Dir)
				break
			}
		}
	}
	return hits, nil
}
```

Note the `args` helper above is a workaround — prefer using cobra's positional args directly. Rewrite `newStoreRemoveCmd` cleanly:

```go
func newStoreRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a store (refused if clauses from it are ratified).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			name := cmdArgs[0]
			if err := qpath.ValidateStoreName(name); err != nil {
				return err
			}
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			dir := filepath.Join(unionDir, "stores", name)
			if _, err := os.Stat(dir); err != nil {
				return fmt.Errorf("no such store: %s", name)
			}
			offenders, err := shopsReferencingStore(name)
			if err != nil {
				return err
			}
			if len(offenders) > 0 {
				sort.Strings(offenders)
				return fmt.Errorf("refusing to remove store %q: still ratified in shop(s): %v", name, offenders)
			}
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("remove store: %w", err)
			}
			return nil
		},
	}
}
```

Delete the intermediate `args` helper. Tasks 11 and 12 add the remaining subcommands; for this task, add empty stubs so the package compiles:

```go
func newStoreRemoteCmd() *cobra.Command { return &cobra.Command{Use: "remote", Short: "stub"} }
func newStorePushCmd() *cobra.Command   { return &cobra.Command{Use: "push", Short: "stub"} }
func newStorePullCmd() *cobra.Command   { return &cobra.Command{Use: "pull", Short: "stub"} }
func newStoreFetchCmd() *cobra.Command  { return &cobra.Command{Use: "fetch", Short: "stub"} }
func newStoreStatusCmd() *cobra.Command { return &cobra.Command{Use: "status", Short: "stub"} }
```

These will be replaced in Tasks 11 and 12.

- [ ] **Step 2: Wire into root**

Edit `internal/cli/root.go` and add `newStoreCmd()` to the `AddCommand(...)` block:

```go
root.AddCommand(
    newInitCmd(),
    newNewCmd(),
    newClausesCmd(),
    newShowCmd(),
    newEditCmd(),
    newExpelCmd(),
    newOrganizeCmd(),
    newShopsCmd(),
    newDisbandCmd(),
    newRatifyCmd(),
    newStrikeCmd(),
    newContractCmd(),
    newStoreCmd(),
)
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add internal/cli
git commit -m "feat(cli): add 'union store' with add, list, remove"
```

---

## Task 11: union store remote add / remove / list

**Files:**
- Modify: `internal/cli/store.go`

- [ ] **Step 1: Replace the `newStoreRemoteCmd` stub**

Replace the stub in `internal/cli/store.go`:

```go
func newStoreRemoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Manage a store's git remotes.",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "add <store> <name> <url>",
			Short: "Add a git remote to the store.",
			Args:  cobra.ExactArgs(3),
			RunE: func(cmd *cobra.Command, args []string) error {
				s, err := openStoreByName(args[0])
				if err != nil {
					return err
				}
				return s.RemoteAdd(args[1], args[2])
			},
		},
		&cobra.Command{
			Use:   "remove <store> <name>",
			Short: "Remove a git remote from the store.",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				s, err := openStoreByName(args[0])
				if err != nil {
					return err
				}
				return s.RemoteRemove(args[1])
			},
		},
		&cobra.Command{
			Use:   "list <store>",
			Short: "List git remotes for the store.",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				s, err := openStoreByName(args[0])
				if err != nil {
					return err
				}
				rs, err := s.Remotes()
				if err != nil {
					return err
				}
				for _, r := range rs {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", r.Name, r.URL)
				}
				return nil
			},
		},
	)
	return cmd
}

func openStoreByName(name string) (*unionstore.Store, error) {
	if err := qpath.ValidateStoreName(name); err != nil {
		return nil, err
	}
	unionDir, err := paths.UnionDir()
	if err != nil {
		return nil, err
	}
	s, err := unionstore.OpenNamed(unionDir, name)
	if err != nil {
		return nil, fmt.Errorf("no such store: %s", name)
	}
	return s, nil
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/store.go
git commit -m "feat(cli): add union store remote add/remove/list"
```

---

## Task 12: union store push / pull / fetch / status

**Files:**
- Modify: `internal/cli/store.go`

- [ ] **Step 1: Replace the sync-command stubs**

Replace the four stubs in `internal/cli/store.go`:

```go
func newStorePushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push <store> [remote] [branch]",
		Short: "git push in the store's repo.",
		Args:  cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStoreByName(args[0])
			if err != nil {
				return err
			}
			var remote, branch string
			if len(args) > 1 {
				remote = args[1]
			}
			if len(args) > 2 {
				branch = args[2]
			}
			return s.Push(remote, branch)
		},
	}
}

func newStorePullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <store> [remote] [branch]",
		Short: "git pull --rebase in the store's repo.",
		Args:  cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStoreByName(args[0])
			if err != nil {
				return err
			}
			var remote, branch string
			if len(args) > 1 {
				remote = args[1]
			}
			if len(args) > 2 {
				branch = args[2]
			}
			return s.Pull(remote, branch)
		},
	}
}

func newStoreFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch <store> [remote]",
		Short: "git fetch in the store's repo.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStoreByName(args[0])
			if err != nil {
				return err
			}
			remote := ""
			if len(args) > 1 {
				remote = args[1]
			}
			return s.Fetch(remote)
		},
	}
}

func newStoreStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <store>",
		Short: "git status --short --branch for the store.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStoreByName(args[0])
			if err != nil {
				return err
			}
			out, err := s.Status()
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil
		},
	}
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/store.go
git commit -m "feat(cli): add union store push/pull/fetch/status"
```

---

## Task 13: rewrite e2e tests for multi-store

**Files:**
- Modify: `internal/cli/e2e_test.go`

- [ ] **Step 1: Replace the e2e test file**

Overwrite `internal/cli/e2e_test.go`:

```go
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

	// Seed one clause per store.
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

	// Organize the shop and ratify both clauses into one AGENTS.md.
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

	// Edit only the personal clause via the store API + propagate helper.
	if _, err := runRoot(t, "new", "default:base/identity", "-f", a); err == nil {
		t.Fatal("re-'new' should fail; clause already exists")
	}

	// Use show to confirm routing works.
	out, err := runRoot(t, "show", "personal:writing/voice")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !strings.Contains(out, "voice is terse") {
		t.Errorf("show returned %q", out)
	}

	// expel the personal clause; only its block should disappear.
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

	// Add a clause so we have something to push.
	f := filepath.Join(t.TempDir(), "c.md")
	os.WriteFile(f, []byte("hi\n"), 0o644)
	if _, err := runRoot(t, "new", "default:c/d", "-f", f); err != nil {
		t.Fatal(err)
	}
	if _, err := runRoot(t, "store", "push", "default", "origin", "master"); err != nil {
		// Newer git versions default to main; retry with that.
		if _, err2 := runRoot(t, "store", "push", "default", "origin", "main"); err2 != nil {
			t.Fatalf("push: %v / %v", err, err2)
		}
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
```

- [ ] **Step 2: Run the full test suite**

Run: `go test ./...`
Expected: PASS

If `go vet` or linters are wired into the test target, address any findings before continuing.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/e2e_test.go
git commit -m "test(cli): multi-store end-to-end coverage"
```

---

## Task 14: documentation

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the command reference and Quick Start**

Rewrite the "Quick start" and "Command reference" sections of `README.md` to reflect the new syntax:

Replace the Quick start block with:

````markdown
## Quick start

```bash
# one-time setup — creates a 'default' store
union init

# or: name the initial store
# union init personal

# author a clause (qualified as <store>:<path>)
printf 'Be helpful and direct.\n' | union new default:base/identity

# register a project, ratify the clause into its AGENTS.md
cd ~/dev/my-project
union organize .
union ratify default:base/identity
union contract                      # → default:base/identity
cat AGENTS.md                       # shows the marked block

# edits propagate automatically
union edit default:base/identity    # opens $EDITOR; save → updates AGENTS.md here

# add a second store, ratify from both
union store add personal
union new personal:writing/voice -f voice.md
union ratify personal:writing/voice
```
````

Replace the Command reference table with:

````markdown
## Command reference

| Command | Purpose |
|---|---|
| `union init [name]` | Create `$UNION_DIR` and an initial store (default name: `default`) |
| `union new <store:path> [-f FILE]` | Author a new clause (editor, stdin, or `-f FILE`; `-f -` reads stdin) |
| `union clauses [store:prefix]` | List clauses across stores; optional `store:prefix` filter |
| `union show <store:path>` | Print a clause |
| `union edit <store:path>` | Edit a clause in `$VISUAL`/`$EDITOR`; propagates to ratified shops |
| `union expel <store:path>` | Delete a clause; strikes it from ratified shops |
| `union organize [dir] [--contract NAME]` | Register a directory as a shop |
| `union shops` | List registered shops |
| `union disband <dir>` | Unregister a shop |
| `union ratify <store:path>` | Add a clause to this shop's contract |
| `union strike <store:path>` | Remove a clause from this shop's contract |
| `union contract` | Show clauses currently in this shop's contract |
| `union store add <name>` | Create a new store |
| `union store list` | List stores |
| `union store remove <name>` | Delete a store (refused if any shop still ratifies from it) |
| `union store remote add <store> <name> <url>` | Add a git remote to a store |
| `union store remote remove <store> <name>` | Remove a git remote |
| `union store remote list <store>` | List a store's remotes |
| `union store push <store> [remote] [branch]` | `git push` in a store |
| `union store pull <store> [remote] [branch]` | `git pull --rebase` in a store |
| `union store fetch <store> [remote]` | `git fetch` in a store |
| `union store status <store>` | `git status --short --branch` for a store |
````

And update the contract markers section:

````markdown
## Contract markers

Ratified clauses are wrapped in HTML-comment markers that carry the full
`store:path`:

```markdown
<!-- BEGIN union:default:base/identity -->
...clause content...
<!-- END union:default:base/identity -->
```

Content outside markers is preserved untouched across rewrites.
````

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: multi-store CLI and store:path marker syntax"
```

---

## Final verification

- [ ] **Step 1: Full test run**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 2: `go vet`**

Run: `go vet ./...`
Expected: clean.

- [ ] **Step 3: Manual smoke**

```bash
go build -o /tmp/union ./cmd/union
export UNION_DIR=$(mktemp -d)
/tmp/union init
/tmp/union store add personal
/tmp/union store list
printf 'hello\n' | /tmp/union new default:a/b
/tmp/union clauses
/tmp/union show default:a/b
```

Expected output includes both stores listed, the clause printed, etc.
