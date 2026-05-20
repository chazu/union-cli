package harness

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// ClauseFrontmatter is YAML-like frontmatter parsed from hook clauses.
type ClauseFrontmatter struct {
	Type      string   // must be "hook"
	Event     string   // normalized event name
	Harnesses []string // empty = all harnesses
	Degrade   string   // "skip", "warn", "error"; default "skip"
	Matcher   string   // tool/pattern matcher; default ""
	Timeout   int      // milliseconds; 0 = harness default
}

// ParseHookClause parses a clause body that has YAML frontmatter delimited by
// "---" lines. Returns the frontmatter and the command body (everything after
// the closing "---").
func ParseHookClause(body []byte) (*ClauseFrontmatter, string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))

	// First line must be "---".
	if !scanner.Scan() {
		return nil, "", fmt.Errorf("empty clause body")
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return nil, "", fmt.Errorf("hook clause must start with ---")
	}

	// Collect frontmatter lines until closing "---".
	var fmLines []string
	closed := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		fmLines = append(fmLines, line)
	}
	if !closed {
		return nil, "", fmt.Errorf("unterminated frontmatter (missing closing ---)")
	}

	fm, err := parseFrontmatterLines(fmLines)
	if err != nil {
		return nil, "", err
	}

	// Remainder is the command body.
	var cmdLines []string
	for scanner.Scan() {
		cmdLines = append(cmdLines, scanner.Text())
	}
	cmd := strings.TrimSpace(strings.Join(cmdLines, "\n"))

	return fm, cmd, nil
}

// parseFrontmatterLines does minimal key:value parsing. No external YAML dep.
func parseFrontmatterLines(lines []string) (*ClauseFrontmatter, error) {
	fm := &ClauseFrontmatter{Degrade: "skip"}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid frontmatter line: %q", line)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		switch key {
		case "type":
			fm.Type = val
		case "event":
			fm.Event = val
		case "degrade":
			fm.Degrade = val
		case "matcher":
			fm.Matcher = val
		case "timeout":
			n := 0
			for _, c := range val {
				if c < '0' || c > '9' {
					break
				}
				n = n*10 + int(c-'0')
			}
			fm.Timeout = n
		case "harnesses":
			fm.Harnesses = parseList(val)
		}
	}

	if fm.Type != "hook" {
		return nil, fmt.Errorf("clause type must be \"hook\", got %q", fm.Type)
	}
	if fm.Event == "" {
		return nil, fmt.Errorf("hook clause must declare an event")
	}
	switch fm.Degrade {
	case "skip", "warn", "error":
	default:
		return nil, fmt.Errorf("invalid degrade value: %q", fm.Degrade)
	}

	return fm, nil
}

// parseList parses "[a, b, c]" into []string{"a", "b", "c"}.
func parseList(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ToHook converts parsed frontmatter + command into a normalized Hook.
func (fm *ClauseFrontmatter) ToHook(command string) Hook {
	return Hook{
		Event:   fm.Event,
		Matcher: fm.Matcher,
		Command: command,
		Timeout: fm.Timeout,
		Degrade: fm.Degrade,
	}
}
