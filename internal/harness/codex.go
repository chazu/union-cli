package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Codex is the adapter for OpenAI's Codex CLI harness.
// Codex uses AGENTS.md for instructions and a codex.json or .codex/ directory
// for configuration. Hook support is limited to pre-commit style checks.
type Codex struct{}

func (c *Codex) Name() string { return "codex" }

func (c *Codex) Detect(dir string) bool {
	// .codex/ directory
	info, err := os.Stat(filepath.Join(dir, ".codex"))
	if err == nil && info.IsDir() {
		return true
	}
	// codex.json
	_, err = os.Stat(filepath.Join(dir, "codex.json"))
	if err == nil {
		return true
	}
	// codex.toml
	_, err = os.Stat(filepath.Join(dir, "codex.toml"))
	return err == nil
}

func (c *Codex) SettingsPath() string { return "codex.toml" }

func (c *Codex) Capabilities() []string {
	return []string{"PreCommit"}
}

func (c *Codex) Supports(event string) bool {
	for _, e := range c.Capabilities() {
		if e == event {
			return true
		}
	}
	return false
}

// codexConfig is the TOML structure for codex.toml.
type codexConfig struct {
	Hooks codexHooks `toml:"hooks"`
	Rest  map[string]interface{}
}

type codexHooks struct {
	PreCommit []string `toml:"pre_commit,omitempty"`
}

func (c *Codex) Emit(hooks []Hook, existing []byte) ([]byte, error) {
	var cfg map[string]interface{}
	if existing != nil {
		if err := toml.Unmarshal(existing, &cfg); err != nil {
			return nil, fmt.Errorf("parse existing codex.toml: %w", err)
		}
	}
	if cfg == nil {
		cfg = map[string]interface{}{}
	}

	// Extract existing hooks section.
	hooksSection, _ := cfg["hooks"].(map[string]interface{})
	if hooksSection == nil {
		hooksSection = map[string]interface{}{}
	}

	// Get existing pre_commit, filter out union-managed ones.
	var preCommit []string
	if existing, ok := hooksSection["pre_commit"].([]interface{}); ok {
		for _, v := range existing {
			if s, ok := v.(string); ok && !isUnionManaged(s) {
				preCommit = append(preCommit, s)
			}
		}
	}

	// Add new union-managed hooks.
	for _, h := range hooks {
		if !c.Supports(h.Event) {
			continue
		}
		preCommit = append(preCommit, unionManagedPrefix+h.Command)
	}

	if len(preCommit) > 0 {
		hooksSection["pre_commit"] = preCommit
		cfg["hooks"] = hooksSection
	}

	return tomlMarshal(cfg)
}

func (c *Codex) Import(config []byte) ([]Hook, error) {
	var cfg struct {
		Hooks struct {
			PreCommit []string `toml:"pre_commit"`
		} `toml:"hooks"`
	}
	if err := toml.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("parse codex.toml: %w", err)
	}

	var out []Hook
	for _, cmd := range cfg.Hooks.PreCommit {
		out = append(out, Hook{
			Event:   "PreCommit",
			Command: cmd,
			Degrade: "skip",
		})
	}
	return out, nil
}

// tomlMarshal encodes a map to TOML bytes with a header comment.
func tomlMarshal(cfg map[string]interface{}) ([]byte, error) {
	// Use JSON round-trip to get consistent output since BurntSushi/toml
	// encoder wants typed structs. We'll use a simple manual approach.
	buf, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	// Re-parse as generic and encode via toml.
	var generic interface{}
	if err := json.Unmarshal(buf, &generic); err != nil {
		return nil, err
	}

	var out []byte
	out = append(out, "# codex configuration\n"...)

	// Use the toml encoder with a struct-like map.
	enc, err := toml.Marshal(generic)
	if err != nil {
		return nil, fmt.Errorf("encode codex.toml: %w", err)
	}
	out = append(out, enc...)
	return out, nil
}
