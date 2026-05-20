package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Claude is the adapter for Claude Code.
type Claude struct{}

func (c *Claude) Name() string { return "claude" }

func (c *Claude) Detect(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".claude"))
	return err == nil && info.IsDir()
}

func (c *Claude) SettingsPath() string { return filepath.Join(".claude", "settings.json") }

func (c *Claude) Capabilities() []string {
	return []string{"SessionStart", "PreToolUse", "PostToolUse", "UserPrompt", "Stop"}
}

func (c *Claude) Supports(event string) bool {
	for _, e := range c.Capabilities() {
		if e == event {
			return true
		}
	}
	return false
}

func claudeEventName(event string) string {
	switch event {
	case "UserPrompt":
		return "UserPromptSubmit"
	default:
		return event
	}
}

func normalizeClaudeEvent(event string) string {
	switch event {
	case "UserPromptSubmit":
		return "UserPrompt"
	default:
		return event
	}
}

type claudeHookGroup struct {
	Matcher string       `json:"matcher"`
	Hooks   []claudeHook `json:"hooks"`
}

type claudeHook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

const unionManagedPrefix = "[union] "

func (c *Claude) Emit(hooks []Hook, existing []byte) ([]byte, error) {
	raw := map[string]json.RawMessage{}
	if existing != nil {
		if err := json.Unmarshal(existing, &raw); err != nil {
			return nil, fmt.Errorf("parse existing settings.json: %w", err)
		}
	}

	var existingHooks map[string][]claudeHookGroup
	if h, ok := raw["hooks"]; ok {
		if err := json.Unmarshal(h, &existingHooks); err != nil {
			return nil, fmt.Errorf("parse existing hooks: %w", err)
		}
	}
	if existingHooks == nil {
		existingHooks = map[string][]claudeHookGroup{}
	}

	// Remove old union-managed hook entries.
	for event, groups := range existingHooks {
		var kept []claudeHookGroup
		for _, g := range groups {
			var keptHooks []claudeHook
			for _, h := range g.Hooks {
				if !isUnionManaged(h.Command) {
					keptHooks = append(keptHooks, h)
				}
			}
			if len(keptHooks) > 0 {
				g.Hooks = keptHooks
				kept = append(kept, g)
			}
		}
		if len(kept) > 0 {
			existingHooks[event] = kept
		} else {
			delete(existingHooks, event)
		}
	}

	// Insert new union-managed hooks.
	for _, hook := range hooks {
		if !c.Supports(hook.Event) {
			continue
		}
		nativeEvent := claudeEventName(hook.Event)
		ch := claudeHook{
			Type:    "command",
			Command: unionManagedPrefix + hook.Command,
			Timeout: hook.Timeout,
		}
		group := claudeHookGroup{
			Matcher: hook.Matcher,
			Hooks:   []claudeHook{ch},
		}
		existingHooks[nativeEvent] = append(existingHooks[nativeEvent], group)
	}

	if len(existingHooks) > 0 {
		hooksJSON, err := json.Marshal(existingHooks)
		if err != nil {
			return nil, err
		}
		raw["hooks"] = hooksJSON
	} else {
		delete(raw, "hooks")
	}

	return json.MarshalIndent(raw, "", "  ")
}

func (c *Claude) Import(config []byte) ([]Hook, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(config, &raw); err != nil {
		return nil, fmt.Errorf("parse settings.json: %w", err)
	}
	h, ok := raw["hooks"]
	if !ok {
		return nil, nil
	}
	var hooksMap map[string][]claudeHookGroup
	if err := json.Unmarshal(h, &hooksMap); err != nil {
		return nil, fmt.Errorf("parse hooks: %w", err)
	}

	var out []Hook
	for event, groups := range hooksMap {
		normalEvent := normalizeClaudeEvent(event)
		for _, g := range groups {
			for _, ch := range g.Hooks {
				if ch.Type != "command" {
					continue
				}
				out = append(out, Hook{
					Event:   normalEvent,
					Matcher: g.Matcher,
					Command: ch.Command,
					Timeout: ch.Timeout,
					Degrade: "skip",
				})
			}
		}
	}
	return out, nil
}

func isUnionManaged(cmd string) bool {
	return len(cmd) >= len(unionManagedPrefix) && cmd[:len(unionManagedPrefix)] == unionManagedPrefix
}
