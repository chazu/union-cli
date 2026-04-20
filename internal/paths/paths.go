// Package paths resolves the on-disk layout for the union store.
package paths

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chazu/union/internal/qpath"
)

// StoresSubdir is the directory name under UNION_DIR that holds named stores.
const StoresSubdir = "stores"

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
	return filepath.Join(root, StoresSubdir), nil
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
