// Package store is a git-backed, path-addressed clause store.
//
// Every mutating operation (Put, Delete) auto-commits. Paths are
// hierarchical and map to files on disk with a ".md" extension.
package store

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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
	if err := os.MkdirAll(filepath.Join(dir, "clauses"), 0o755); err != nil {
		return nil, fmt.Errorf("create clauses dir: %w", err)
	}
	shopsFile := filepath.Join(dir, "shops.toml")
	if _, err := os.Stat(shopsFile); os.IsNotExist(err) {
		if err := os.WriteFile(shopsFile, []byte("# union shops registry\n"), 0o644); err != nil {
			return nil, fmt.Errorf("seed shops.toml: %w", err)
		}
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
