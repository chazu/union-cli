package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeDetect(t *testing.T) {
	dir := t.TempDir()
	c := &Claude{}

	if c.Detect(dir) {
		t.Fatal("should not detect without .claude dir")
	}

	os.Mkdir(filepath.Join(dir, ".claude"), 0o755)
	if !c.Detect(dir) {
		t.Fatal("should detect with .claude dir")
	}
}

func TestOpenCodeDetect(t *testing.T) {
	dir := t.TempDir()
	o := &OpenCode{}

	if o.Detect(dir) {
		t.Fatal("should not detect without indicators")
	}

	os.Mkdir(filepath.Join(dir, ".opencode"), 0o755)
	if !o.Detect(dir) {
		t.Fatal("should detect with .opencode dir")
	}
}

func TestOpenCodeEmitPlugin(t *testing.T) {
	hooks := []Hook{
		{Event: "SessionStart", Command: "echo start", Degrade: "skip"},
		{Event: "PreToolUse", Command: "lint check", Degrade: "warn"},
	}

	out, err := (&OpenCode{}).Emit(hooks, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !contains(s, "session.created") {
		t.Fatal("expected session.created event in plugin output")
	}
	if !contains(s, "tool.execute.before") {
		t.Fatal("expected tool.execute.before event in plugin output")
	}
	if !contains(s, "[union] echo start") {
		t.Fatal("expected union-managed command prefix")
	}
	if !contains(s, "execSync(") {
		t.Fatal("expected execSync call in plugin")
	}
	if !contains(s, `console.warn`) {
		t.Fatal("expected warn handler for degrade=warn hook")
	}
}

func TestOpenCodeEmitEmpty(t *testing.T) {
	out, err := (&OpenCode{}).Emit(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(out), "no hooks configured") {
		t.Fatalf("expected empty plugin stub, got:\n%s", string(out))
	}
}

func TestOpenCodeImportPlugin(t *testing.T) {
	hooks := []Hook{
		{Event: "SessionStart", Command: "echo hello", Degrade: "skip"},
		{Event: "PostToolUse", Command: "echo done", Degrade: "skip"},
	}

	out, err := (&OpenCode{}).Emit(hooks, nil)
	if err != nil {
		t.Fatal(err)
	}

	imported, err := (&OpenCode{}).Import(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != 2 {
		t.Fatalf("expected 2 imported hooks, got %d", len(imported))
	}

	events := map[string]bool{}
	for _, h := range imported {
		events[h.Event] = true
	}
	if !events["SessionStart"] {
		t.Fatal("expected SessionStart in imported hooks")
	}
	if !events["PostToolUse"] {
		t.Fatal("expected PostToolUse in imported hooks")
	}
}

func TestOpenCodeSkipsUnsupported(t *testing.T) {
	hooks := []Hook{
		{Event: "SomeUnknownEvent", Command: "nope", Degrade: "skip"},
	}
	out, err := (&OpenCode{}).Emit(hooks, nil)
	if err != nil {
		t.Fatal(err)
	}
	if contains(string(out), "nope") {
		t.Fatal("unsupported event should be skipped")
	}
}

func TestOpenCodeCapabilities(t *testing.T) {
	o := &OpenCode{}
	caps := o.Capabilities()
	expected := []string{"SessionStart", "PreToolUse", "PostToolUse", "UserPrompt", "Stop", "PreCommit"}
	if len(caps) != len(expected) {
		t.Fatalf("expected %d capabilities, got %d", len(expected), len(caps))
	}
	for i, e := range expected {
		if caps[i] != e {
			t.Fatalf("capability[%d]: expected %s, got %s", i, e, caps[i])
		}
	}
}

func TestDetectMultiple(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".claude"), 0o755)
	os.Mkdir(filepath.Join(dir, ".opencode"), 0o755)

	found := Detect(dir)
	if len(found) != 2 {
		t.Fatalf("expected 2 harnesses, got %d", len(found))
	}
}

func TestClaudeEmitAndImport(t *testing.T) {
	hooks := []Hook{
		{Event: "SessionStart", Command: "echo hello", Degrade: "skip"},
		{Event: "UserPrompt", Command: "lint check", Matcher: "Bash", Degrade: "skip"},
	}

	out, err := (&Claude{}).Emit(hooks, nil)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["hooks"]; !ok {
		t.Fatal("expected hooks key in output")
	}

	// Verify union-managed prefix.
	var hooksMap map[string][]claudeHookGroup
	json.Unmarshal(raw["hooks"], &hooksMap)

	groups, ok := hooksMap["SessionStart"]
	if !ok || len(groups) == 0 {
		t.Fatal("expected SessionStart hook group")
	}
	if groups[0].Hooks[0].Command != "[union] echo hello" {
		t.Fatalf("unexpected command: %s", groups[0].Hooks[0].Command)
	}

	// UserPrompt -> UserPromptSubmit
	groups, ok = hooksMap["UserPromptSubmit"]
	if !ok || len(groups) == 0 {
		t.Fatal("expected UserPromptSubmit hook group")
	}
	if groups[0].Matcher != "Bash" {
		t.Fatalf("unexpected matcher: %s", groups[0].Matcher)
	}

	// Import round-trip.
	imported, err := (&Claude{}).Import(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != 2 {
		t.Fatalf("expected 2 imported hooks, got %d", len(imported))
	}
}

func TestClaudeEmitPreservesExisting(t *testing.T) {
	existing := []byte(`{
  "permissions": {"allow": ["Read"]},
  "hooks": {
    "SessionStart": [{"matcher": "", "hooks": [{"type": "command", "command": "user hook"}]}]
  }
}`)

	hooks := []Hook{{Event: "SessionStart", Command: "union hook", Degrade: "skip"}}
	out, err := (&Claude{}).Emit(hooks, existing)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	json.Unmarshal(out, &raw)

	// Permissions preserved.
	if _, ok := raw["permissions"]; !ok {
		t.Fatal("permissions should be preserved")
	}

	// Both hooks present.
	var hooksMap map[string][]claudeHookGroup
	json.Unmarshal(raw["hooks"], &hooksMap)
	groups := hooksMap["SessionStart"]
	if len(groups) != 2 {
		t.Fatalf("expected 2 hook groups (user + union), got %d", len(groups))
	}
}

func TestParseHookClause(t *testing.T) {
	body := []byte(`---
type: hook
event: SessionStart
harnesses: [claude, opencode]
degrade: warn
---
echo hello && echo world
`)

	fm, cmd, err := ParseHookClause(body)
	if err != nil {
		t.Fatal(err)
	}
	if fm.Event != "SessionStart" {
		t.Fatalf("expected SessionStart, got %s", fm.Event)
	}
	if fm.Degrade != "warn" {
		t.Fatalf("expected warn, got %s", fm.Degrade)
	}
	if len(fm.Harnesses) != 2 {
		t.Fatalf("expected 2 harnesses, got %d", len(fm.Harnesses))
	}
	if cmd != "echo hello && echo world" {
		t.Fatalf("unexpected command: %q", cmd)
	}
}

func TestParseHookClauseMinimal(t *testing.T) {
	body := []byte(`---
type: hook
event: PreCommit
---
make test
`)

	fm, cmd, err := ParseHookClause(body)
	if err != nil {
		t.Fatal(err)
	}
	if fm.Event != "PreCommit" {
		t.Fatalf("expected PreCommit, got %s", fm.Event)
	}
	if fm.Degrade != "skip" {
		t.Fatalf("expected default degrade=skip, got %s", fm.Degrade)
	}
	if len(fm.Harnesses) != 0 {
		t.Fatalf("expected no harness filter, got %v", fm.Harnesses)
	}
	if cmd != "make test" {
		t.Fatalf("unexpected command: %q", cmd)
	}
}

func TestParseHookClauseErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty", ""},
		{"no opening", "type: hook\nevent: X\n"},
		{"no closing", "---\ntype: hook\nevent: X\n"},
		{"wrong type", "---\ntype: skill\nevent: X\n---\ncmd\n"},
		{"no event", "---\ntype: hook\n---\ncmd\n"},
		{"bad degrade", "---\ntype: hook\nevent: X\ndegrade: explode\n---\ncmd\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseHookClause([]byte(tt.body))
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestLoadConfigMissing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Harnesses) != 0 {
		t.Fatal("expected empty harnesses")
	}
}

func TestLoadConfigExplicit(t *testing.T) {
	dir := t.TempDir()
	content := `
[harnesses.claude]
settings = ".claude/settings.json"

[harnesses.opencode]
settings = "opencode.json"

[hooks]
ratified = ["default:hooks/session-start"]
`
	os.WriteFile(filepath.Join(dir, "union.toml"), []byte(content), 0o644)

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Harnesses) != 2 {
		t.Fatalf("expected 2 harnesses, got %d", len(cfg.Harnesses))
	}
	if len(cfg.Hooks.Ratified) != 1 {
		t.Fatalf("expected 1 ratified hook, got %d", len(cfg.Hooks.Ratified))
	}
}

func TestResolveHarnessesFromConfig(t *testing.T) {
	dir := t.TempDir()
	content := `[harnesses.claude]
`
	os.WriteFile(filepath.Join(dir, "union.toml"), []byte(content), 0o644)

	adapters, err := ResolveHarnesses(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Name() != "claude" {
		t.Fatalf("expected [claude], got %v", adapters)
	}
}

func TestCodexDetect(t *testing.T) {
	dir := t.TempDir()
	c := &Codex{}

	if c.Detect(dir) {
		t.Fatal("should not detect without indicators")
	}

	os.Mkdir(filepath.Join(dir, ".codex"), 0o755)
	if !c.Detect(dir) {
		t.Fatal("should detect with .codex dir")
	}
}

func TestCodexDetectToml(t *testing.T) {
	dir := t.TempDir()
	c := &Codex{}
	os.WriteFile(filepath.Join(dir, "codex.toml"), []byte(""), 0o644)
	if !c.Detect(dir) {
		t.Fatal("should detect with codex.toml")
	}
}

func TestCodexEmitAndImport(t *testing.T) {
	hooks := []Hook{
		{Event: "PreCommit", Command: "make test", Degrade: "skip"},
	}

	out, err := (&Codex{}).Emit(hooks, nil)
	if err != nil {
		t.Fatal(err)
	}
	outStr := string(out)
	if !contains(outStr, "[union] make test") {
		t.Fatalf("expected union-managed command in output, got:\n%s", outStr)
	}

	imported, err := (&Codex{}).Import(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != 1 {
		t.Fatalf("expected 1 imported hook, got %d", len(imported))
	}
	if imported[0].Event != "PreCommit" {
		t.Fatalf("expected PreCommit, got %s", imported[0].Event)
	}
}

func TestCodexEmitSkipsUnsupported(t *testing.T) {
	hooks := []Hook{
		{Event: "SessionStart", Command: "echo hi", Degrade: "skip"},
	}

	out, err := (&Codex{}).Emit(hooks, nil)
	if err != nil {
		t.Fatal(err)
	}
	if contains(string(out), "echo hi") {
		t.Fatal("SessionStart should be skipped for Codex")
	}
}

func TestCodexEmitPreservesExisting(t *testing.T) {
	existing := []byte(`# codex configuration
[hooks]
pre_commit = ["user lint"]
`)

	hooks := []Hook{{Event: "PreCommit", Command: "make vet", Degrade: "skip"}}
	out, err := (&Codex{}).Emit(hooks, existing)
	if err != nil {
		t.Fatal(err)
	}
	outStr := string(out)
	if !contains(outStr, "user lint") {
		t.Fatalf("user hook should be preserved, got:\n%s", outStr)
	}
	if !contains(outStr, "[union] make vet") {
		t.Fatalf("union hook should be added, got:\n%s", outStr)
	}
}

func TestJCodeDetect(t *testing.T) {
	dir := t.TempDir()
	j := &JCode{}

	if j.Detect(dir) {
		t.Fatal("should not detect without .jcode dir")
	}

	os.Mkdir(filepath.Join(dir, ".jcode"), 0o755)
	if !j.Detect(dir) {
		t.Fatal("should detect with .jcode dir")
	}
}

func TestJCodeEmitAndImport(t *testing.T) {
	hooks := []Hook{
		{Event: "SessionStart", Command: "echo start", Degrade: "skip"},
		{Event: "PreToolUse", Command: "check tool", Matcher: "Bash", Degrade: "skip"},
	}

	out, err := (&JCode{}).Emit(hooks, nil)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatal(err)
	}

	var hooksMap map[string][]jcodeHookEntry
	json.Unmarshal(raw["hooks"], &hooksMap)

	entries, ok := hooksMap["on_session_start"]
	if !ok || len(entries) == 0 {
		t.Fatal("expected on_session_start hook")
	}
	if entries[0].Command != "[union] echo start" {
		t.Fatalf("unexpected command: %s", entries[0].Command)
	}

	entries, ok = hooksMap["before_tool"]
	if !ok || len(entries) == 0 {
		t.Fatal("expected before_tool hook")
	}
	if entries[0].Matcher != "Bash" {
		t.Fatalf("unexpected matcher: %s", entries[0].Matcher)
	}

	// Import round-trip.
	imported, err := (&JCode{}).Import(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != 2 {
		t.Fatalf("expected 2 imported hooks, got %d", len(imported))
	}
}

func TestJCodeEmitPreservesExisting(t *testing.T) {
	existing := []byte(`{
  "theme": "dark",
  "hooks": {
    "on_session_start": [{"type": "command", "command": "user hook"}]
  }
}`)

	hooks := []Hook{{Event: "SessionStart", Command: "union hook", Degrade: "skip"}}
	out, err := (&JCode{}).Emit(hooks, existing)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	json.Unmarshal(out, &raw)

	if _, ok := raw["theme"]; !ok {
		t.Fatal("theme should be preserved")
	}

	var hooksMap map[string][]jcodeHookEntry
	json.Unmarshal(raw["hooks"], &hooksMap)
	entries := hooksMap["on_session_start"]
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (user + union), got %d", len(entries))
	}
}

func TestJCodeSkipsUnsupported(t *testing.T) {
	hooks := []Hook{
		{Event: "UserPrompt", Command: "echo nope", Degrade: "skip"},
	}

	out, err := (&JCode{}).Emit(hooks, nil)
	if err != nil {
		t.Fatal(err)
	}
	if contains(string(out), "echo nope") {
		t.Fatal("UserPrompt should be skipped for JCode")
	}
}

func TestDetectAll(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".claude"), 0o755)
	os.Mkdir(filepath.Join(dir, ".opencode"), 0o755)
	os.Mkdir(filepath.Join(dir, ".codex"), 0o755)
	os.Mkdir(filepath.Join(dir, ".jcode"), 0o755)

	found := Detect(dir)
	if len(found) != 4 {
		names := make([]string, len(found))
		for i, a := range found {
			names[i] = a.Name()
		}
		t.Fatalf("expected 4 harnesses, got %d: %v", len(found), names)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestResolveHarnessesFallsBackToDetection(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".claude"), 0o755)

	adapters, err := ResolveHarnesses(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Name() != "claude" {
		t.Fatalf("expected [claude] from detection, got %v", adapters)
	}
}
