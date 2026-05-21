package harness

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents a union.toml project manifest.
type Config struct {
	Harnesses map[string]HarnessConfig `toml:"harnesses"`
	Hooks     HooksConfig              `toml:"hooks"`
	Pointers  PointerConfig            `toml:"pointers"`
}

// PointerConfig declares symlink/redirect files that point harness-specific
// guidance files (e.g., CLAUDE.md) to the canonical contract (e.g., AGENTS.md).
type PointerConfig struct {
	Targets []string `toml:"targets,omitempty"`
}

// HarnessConfig is per-harness configuration in union.toml.
type HarnessConfig struct {
	Settings string `toml:"settings,omitempty"`
	Detected bool   `toml:"detected,omitempty"`
}

// HooksConfig declares which hook clauses are ratified for this shop.
type HooksConfig struct {
	Ratified []string `toml:"ratified,omitempty"`
}

// DefaultConfigName is the filename union looks for in a shop directory.
const DefaultConfigName = "union.toml"

// LoadConfig reads a union.toml from the given directory.
// Returns an empty Config (not an error) if the file doesn't exist.
func LoadConfig(dir string) (*Config, error) {
	path := filepath.Join(dir, DefaultConfigName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Harnesses: map[string]HarnessConfig{}}, nil
		}
		return nil, fmt.Errorf("read union.toml: %w", err)
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse union.toml: %w", err)
	}
	if cfg.Harnesses == nil {
		cfg.Harnesses = map[string]HarnessConfig{}
	}
	return &cfg, nil
}

// SaveConfig writes a union.toml to the given directory atomically.
func SaveConfig(dir string, cfg *Config) error {
	path := filepath.Join(dir, DefaultConfigName)
	tmp, err := os.CreateTemp(dir, ".union-*.toml.tmp")
	if err != nil {
		return fmt.Errorf("create temp union.toml: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.WriteString("# union project manifest\n"); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	enc := toml.NewEncoder(tmp)
	if err := enc.Encode(cfg); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("encode union.toml: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// ResolveHarnesses returns the effective set of adapters for a shop.
// If union.toml declares harnesses explicitly, use those.
// Otherwise, auto-detect from the directory.
func ResolveHarnesses(dir string) ([]Adapter, error) {
	cfg, err := LoadConfig(dir)
	if err != nil {
		return nil, err
	}

	// Explicit declarations take precedence.
	if len(cfg.Harnesses) > 0 {
		var adapters []Adapter
		for name := range cfg.Harnesses {
			a := ByName(name)
			if a == nil {
				return nil, fmt.Errorf("unknown harness: %q", name)
			}
			adapters = append(adapters, a)
		}
		return adapters, nil
	}

	// Fall back to auto-detection.
	return Detect(dir), nil
}
