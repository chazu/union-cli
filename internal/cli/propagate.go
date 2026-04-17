package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/shop"
)

// propagateUpdate rewrites the matching marked block in every shop whose
// contract currently contains <path>. Does not touch git in those shops.
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
	for _, s := range r.List() {
		contractPath := filepath.Join(s.Dir, s.Contract)
		body, err := os.ReadFile(contractPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Fprintf(w, "warn: %s: %v\n", contractPath, err)
			continue
		}
		if !shop.HasClause(body, clausePath) {
			continue
		}
		newBody, err := fn(contractPath, body)
		if err != nil {
			fmt.Fprintf(w, "warn: %s: %v\n", contractPath, err)
			continue
		}
		if err := os.WriteFile(contractPath, newBody, 0o644); err != nil {
			fmt.Fprintf(w, "warn: %s: %v\n", contractPath, err)
			continue
		}
		fmt.Fprintf(w, "  updated %s\n", contractPath)
	}
	return nil
}
