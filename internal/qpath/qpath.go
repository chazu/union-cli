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
