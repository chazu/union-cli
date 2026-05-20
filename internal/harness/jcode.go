package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// JCode is the adapter for the JCode harness.
// JCode uses a .jcode/ directory with a settings.json for configuration.
// Hook model is similar to Claude Code (event-based, command type).
type JCode struct{}

func (j *JCode) Name() string { return "jcode" }

func (j *JCode) Detect(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".jcode"))
	return err == nil && info.IsDir()
}

func (j *JCode) SettingsPath() string { return filepath.Join(".jcode", "settings.json") }

func (j *JCode) Capabilities() []string {
	return []string{"SessionStart", "PreToolUse", "PostToolUse", "Stop"}
}

func (j *JCode) Supports(event string) bool {
	for _, e := range j.Capabilities() {
		if e == event {
			return true
		}
	}
	return false
}

func jcodeEventName(event string) string {
	switch event {
	case "SessionStart":
		return "on_session_start"
	case "PreToolUse":
		return "before_tool"
	case "PostToolUse":
		return "after_tool"
	case "Stop":
		return "on_session_end"
	default:
		return event
	}
}

func normalizeJCodeEvent(event string) string {
	switch event {
	case "on_session_start":
		return "SessionStart"
	case "before_tool":
		return "PreToolUse"
	case "after_tool":
		return "PostToolUse"
	case "on_session_end":
		return "Stop"
	default:
		return event
	}
}

type jcodeHookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Matcher string `json:"matcher,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

func (j *JCode) Emit(hooks []Hook, existing []byte) ([]byte, error) {
	raw := map[string]json.RawMessage{}
	if existing != nil {
		if err := json.Unmarshal(existing, &raw); err != nil {
			return nil, fmt.Errorf("parse existing jcode settings.json: %w", err)
		}
	}

	var existingHooks map[string][]jcodeHookEntry
	if h, ok := raw["hooks"]; ok {
		if err := json.Unmarshal(h, &existingHooks); err != nil {
			return nil, fmt.Errorf("parse existing hooks: %w", err)
		}
	}
	if existingHooks == nil {
		existingHooks = map[string][]jcodeHookEntry{}
	}

	// Remove old union-managed entries.
	for event, entries := range existingHooks {
		var kept []jcodeHookEntry
		for _, e := range entries {
			if !isUnionManaged(e.Command) {
				kept = append(kept, e)
			}
		}
		if len(kept) > 0 {
			existingHooks[event] = kept
		} else {
			delete(existingHooks, event)
		}
	}

	// Add new union-managed hooks.
	for _, hook := range hooks {
		if !j.Supports(hook.Event) {
			continue
		}
		nativeEvent := jcodeEventName(hook.Event)
		entry := jcodeHookEntry{
			Type:    "command",
			Command: unionManagedPrefix + hook.Command,
			Matcher: hook.Matcher,
			Timeout: hook.Timeout,
		}
		existingHooks[nativeEvent] = append(existingHooks[nativeEvent], entry)
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

func (j *JCode) Import(config []byte) ([]Hook, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(config, &raw); err != nil {
		return nil, fmt.Errorf("parse jcode settings.json: %w", err)
	}
	h, ok := raw["hooks"]
	if !ok {
		return nil, nil
	}
	var hooksMap map[string][]jcodeHookEntry
	if err := json.Unmarshal(h, &hooksMap); err != nil {
		return nil, fmt.Errorf("parse hooks: %w", err)
	}

	var out []Hook
	for event, entries := range hooksMap {
		normalEvent := normalizeJCodeEvent(event)
		for _, e := range entries {
			if e.Type != "command" {
				continue
			}
			out = append(out, Hook{
				Event:   normalEvent,
				Matcher: e.Matcher,
				Command: e.Command,
				Timeout: e.Timeout,
				Degrade: "skip",
			})
		}
	}
	return out, nil
}
