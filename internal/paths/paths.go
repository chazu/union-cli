// Package paths resolves the on-disk layout for the union store.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
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

// ClausesDir returns $UNION_DIR/clauses.
func ClausesDir() (string, error) {
	root, err := UnionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "clauses"), nil
}

// ShopsFile returns $UNION_DIR/shops.toml.
func ShopsFile() (string, error) {
	root, err := UnionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "shops.toml"), nil
}
