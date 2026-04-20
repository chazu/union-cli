// Package store is a git-backed, path-addressed clause store.
//
// Every mutating operation (Put, Delete) auto-commits. Paths are
// hierarchical and map to files on disk with a ".md" extension.
package store

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// streamOut and streamErr are the sinks for gitStream. Overridable by tests
// to silence git progress output and enable capture.
var (
	streamOut io.Writer = os.Stdout
	streamErr io.Writer = os.Stderr
)

const clauseExt = ".md"

// Store is a handle to a union clause store rooted at a directory.
type Store struct {
	root string // $UNION_DIR
}

// Init creates a new store at dir. dir must not already contain a .git.
func Init(dir string) (*Store, error) {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return nil, fmt.Errorf("store already initialized at %s", dir)
	}
	clausesDir := filepath.Join(dir, "clauses")
	if err := os.MkdirAll(clausesDir, 0o755); err != nil {
		return nil, fmt.Errorf("create clauses dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(clausesDir, ".gitkeep"), nil, 0o644); err != nil {
		return nil, fmt.Errorf("seed .gitkeep: %w", err)
	}
	s := &Store{root: dir}
	if err := s.git("init", "-q"); err != nil {
		return nil, fmt.Errorf("git init: %w", err)
	}
	if err := s.git("add", "-A"); err != nil {
		return nil, fmt.Errorf("git add: %w", err)
	}
	if err := s.git("commit", "-q", "-m", "init union store"); err != nil {
		return nil, fmt.Errorf("initial commit: %w", err)
	}
	return s, nil
}

// Open attaches to an existing store at dir.
func Open(dir string) (*Store, error) {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return nil, fmt.Errorf("no union store at %s: run 'union init'", dir)
	}
	return &Store{root: dir}, nil
}

// Root returns the store's root directory.
func (s *Store) Root() string { return s.root }

// Put writes body at logical path and commits with msg. Creates dirs as needed.
func (s *Store) Put(path string, body []byte, msg string) error {
	if err := validatePath(path); err != nil {
		return err
	}
	fp := s.filePath(path)
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(fp, body, 0o644); err != nil {
		return fmt.Errorf("write clause: %w", err)
	}
	rel, _ := filepath.Rel(s.root, fp)
	if err := s.git("add", rel); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	dirty, err := s.hasStagedChanges()
	if err != nil {
		return err
	}
	if !dirty {
		return nil
	}
	if err := s.git("commit", "-q", "-m", msg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// Get returns the body at logical path.
func (s *Store) Get(path string) ([]byte, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(s.filePath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no such clause: %s", path)
		}
		return nil, err
	}
	return b, nil
}

// Has reports whether a clause exists at the logical path.
func (s *Store) Has(path string) bool {
	if err := validatePath(path); err != nil {
		return false
	}
	_, err := os.Stat(s.filePath(path))
	return err == nil
}

// Delete removes the clause at logical path and commits with msg.
func (s *Store) Delete(path, msg string) error {
	if err := validatePath(path); err != nil {
		return err
	}
	if !s.Has(path) {
		return fmt.Errorf("no such clause: %s", path)
	}
	fp := s.filePath(path)
	rel, _ := filepath.Rel(s.root, fp)
	if err := s.git("rm", "-q", rel); err != nil {
		return fmt.Errorf("git rm: %w", err)
	}
	if err := s.git("commit", "-q", "-m", msg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// List returns all clause logical paths under prefix, sorted.
// prefix "" returns all.
func (s *Store) List(prefix string) ([]string, error) {
	clausesRoot := filepath.Join(s.root, "clauses")
	var out []string
	err := filepath.WalkDir(clausesRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, clauseExt) {
			return nil
		}
		rel, err := filepath.Rel(clausesRoot, p)
		if err != nil {
			return err
		}
		logical := strings.TrimSuffix(filepath.ToSlash(rel), clauseExt)
		if prefix == "" || strings.HasPrefix(logical, prefix) {
			out = append(out, logical)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func (s *Store) filePath(logical string) string {
	return filepath.Join(s.root, "clauses", filepath.FromSlash(logical)+clauseExt)
}

func (s *Store) git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return nil
}

func (s *Store) hasStagedChanges() (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = s.root
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
		return true, nil
	}
	return false, fmt.Errorf("git diff --cached: %w", err)
}

func validatePath(p string) error {
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

// InitNamed creates $unionDir/stores/<name>/ and initializes it as a store.
func InitNamed(unionDir, name string) (*Store, error) {
	if err := validateStoreName(name); err != nil {
		return nil, err
	}
	dir := filepath.Join(unionDir, "stores", name)
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
// (keeping the store package free of higher-level dependencies).
var storeNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func validateStoreName(name string) error {
	if !storeNameRE.MatchString(name) {
		return fmt.Errorf("invalid store name %q: must match [a-z0-9][a-z0-9_-]*", name)
	}
	return nil
}

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
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Prefer the fetch URL when fetch and push URLs differ.
		if _, ok := seen[fields[0]]; ok && !(len(fields) >= 3 && fields[2] == "(fetch)") {
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

// Push runs `git push` in the store. Empty branch pushes HEAD (current branch).
func (s *Store) Push(remote, branch string) error {
	args := []string{"push"}
	if remote != "" {
		args = append(args, remote)
	}
	if branch != "" {
		args = append(args, branch)
	} else {
		// Push HEAD by name so git doesn't require an upstream tracking branch.
		args = append(args, "HEAD")
	}
	return s.gitStream(args...)
}

// Pull runs `git pull --rebase` in the store. Empty branch uses the current branch.
func (s *Store) Pull(remote, branch string) error {
	args := []string{"pull", "--rebase"}
	if remote != "" {
		args = append(args, remote)
		// git pull requires a branch when a remote is explicitly given and no
		// upstream tracking is configured.  Resolve the current branch name.
		b := branch
		if b == "" {
			var err error
			b, err = s.gitCapture("rev-parse", "--abbrev-ref", "HEAD")
			if err != nil {
				return fmt.Errorf("resolve current branch: %w", err)
			}
			b = strings.TrimSpace(b)
		}
		args = append(args, b)
	} else if branch != "" {
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
	cmd.Stdout = streamOut
	cmd.Stderr = streamErr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil
}
