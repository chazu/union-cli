package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/shop"
)

// propagateUpdate rewrites the matching marked block in every shop whose
// contract currently contains <path>. Does not touch git in those shops.
// Returns a joined error of per-shop failures; caller surfaces it so the
// command exits non-zero when any shop couldn't be updated.
func propagateUpdate(w io.Writer, clausePath string, newBody []byte) error {
	return eachRatifiedShop(w, clausePath, func(contractPath string, body []byte) ([]byte, error) {
		return shop.UpdateClause(body, clausePath, newBody)
	})
}

// propagateRemoval strips the marked block in every shop whose contract
// contains <path>.
func propagateRemoval(w io.Writer, clausePath string) error {
	return eachRatifiedShop(w, clausePath, func(contractPath string, body []byte) ([]byte, error) {
		return shop.RemoveClause(body, clausePath)
	})
}

type rewriteFn func(contractPath string, body []byte) ([]byte, error)

func eachRatifiedShop(w io.Writer, clausePath string, fn rewriteFn) error {
	shopsPath, err := paths.ShopsFile()
	if err != nil {
		return err
	}
	r, err := shop.LoadRegistry(shopsPath)
	if err != nil {
		return err
	}
	var failures []error
	for _, s := range r.List() {
		contractPath := filepath.Join(s.Dir, s.Contract)
		body, err := os.ReadFile(contractPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			failures = append(failures, fmt.Errorf("%s: %w", contractPath, err))
			continue
		}
		if !shop.HasClause(body, clausePath) {
			continue
		}
		newBody, err := fn(contractPath, body)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", contractPath, err))
			continue
		}
		if err := os.WriteFile(contractPath, newBody, 0o644); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", contractPath, err))
			continue
		}
		fmt.Fprintf(w, "  updated %s\n", contractPath)
	}
	if len(failures) > 0 {
		return fmt.Errorf("propagation failed in %d shop(s): %w", len(failures), errors.Join(failures...))
	}
	return nil
}
