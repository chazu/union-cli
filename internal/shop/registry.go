// Package shop manages the registry of organized shops and the parsing
// and mutation of their contracts.
package shop

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

// Shop is one registered project.
type Shop struct {
	Dir      string
	Contract string
}

// Registry is the in-memory view of shops.toml.
type Registry struct {
	path  string
	shops map[string]Shop
}

type tomlShops struct {
	Shops map[string]tomlShop `toml:"shops"`
}
type tomlShop struct {
	Contract string `toml:"contract"`
}

// LoadRegistry reads shops.toml from path. A missing file is treated as empty.
func LoadRegistry(path string) (*Registry, error) {
	r := &Registry{path: path, shops: map[string]Shop{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("read shops.toml: %w", err)
	}
	var t tomlShops
	if err := toml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse shops.toml: %w", err)
	}
	for dir, s := range t.Shops {
		contract := s.Contract
		if contract == "" {
			contract = "AGENTS.md"
		}
		r.shops[dir] = Shop{Dir: dir, Contract: contract}
	}
	return r, nil
}

// Add registers a new shop.
func (r *Registry) Add(dir, contract string) error {
	if _, ok := r.shops[dir]; ok {
		return fmt.Errorf("shop already organized: %s", dir)
	}
	if contract == "" {
		contract = "AGENTS.md"
	}
	r.shops[dir] = Shop{Dir: dir, Contract: contract}
	return nil
}

// Remove unregisters a shop.
func (r *Registry) Remove(dir string) error {
	if _, ok := r.shops[dir]; !ok {
		return fmt.Errorf("not an organized shop: %s", dir)
	}
	delete(r.shops, dir)
	return nil
}

// Get returns the shop at dir if registered.
func (r *Registry) Get(dir string) (Shop, bool) {
	s, ok := r.shops[dir]
	return s, ok
}

// List returns all shops sorted by Dir.
func (r *Registry) List() []Shop {
	out := make([]Shop, 0, len(r.shops))
	for _, s := range r.shops {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dir < out[j].Dir })
	return out
}

// Save writes the registry to disk atomically (write-then-rename).
func (r *Registry) Save() error {
	t := tomlShops{Shops: map[string]tomlShop{}}
	for dir, s := range r.shops {
		t.Shops[dir] = tomlShop{Contract: s.Contract}
	}

	// Write to a temp file in the same directory so os.Rename is atomic.
	dir := filepath.Dir(r.path)
	tmp, err := os.CreateTemp(dir, ".shops-*.toml.tmp")
	if err != nil {
		return fmt.Errorf("create temp shops.toml: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.WriteString("# union shops registry\n"); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	enc := toml.NewEncoder(tmp)
	if err := enc.Encode(t); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("encode shops.toml: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("flush shops.toml: %w", err)
	}
	if err := os.Rename(tmpPath, r.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename shops.toml: %w", err)
	}
	return nil
}
