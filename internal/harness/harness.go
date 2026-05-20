// Package harness provides cross-harness hook management for AI coding tools.
// Each supported harness (Claude Code, OpenCode, Codex, etc.) implements the
// Adapter interface, translating normalized hooks into native config formats.
package harness

// Hook is the normalized representation of a hook parsed from a clause.
type Hook struct {
	Event   string // normalized event name (e.g., "SessionStart")
	Matcher string // tool/pattern matcher (empty = match all)
	Command string // shell command to execute
	Timeout int    // milliseconds, 0 = harness default
	Degrade string // "skip", "warn", or "error"
}

// Adapter translates normalized hooks into a harness-native config format.
type Adapter interface {
	// Name returns the harness identifier (e.g., "claude", "opencode").
	Name() string

	// Detect reports whether this harness is configured in the given directory.
	Detect(shopDir string) bool

	// SettingsPath returns the config file path relative to the shop directory.
	SettingsPath() string

	// Capabilities returns which normalized event names this harness supports.
	Capabilities() []string

	// Supports reports whether this harness handles the given normalized event.
	Supports(event string) bool

	// Emit merges hooks into the existing native config, returning updated bytes.
	// existing may be nil if the config file does not yet exist.
	Emit(hooks []Hook, existing []byte) ([]byte, error)

	// Import reads a native config and returns normalized hooks.
	Import(config []byte) ([]Hook, error)
}

// NormalizedEvents enumerates all events in the union event model.
var NormalizedEvents = []string{
	"SessionStart",
	"PreToolUse",
	"PostToolUse",
	"UserPrompt",
	"Stop",
	"PreCommit",
}

// All returns every registered adapter.
func All() []Adapter {
	return []Adapter{
		&Claude{},
		&OpenCode{},
		&Codex{},
		&JCode{},
	}
}

// Detect returns adapters for harnesses detected in the given directory.
func Detect(shopDir string) []Adapter {
	var found []Adapter
	for _, a := range All() {
		if a.Detect(shopDir) {
			found = append(found, a)
		}
	}
	return found
}

// ByName returns the adapter with the given name, or nil.
func ByName(name string) Adapter {
	for _, a := range All() {
		if a.Name() == name {
			return a
		}
	}
	return nil
}
