package shop

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

const (
	beginPrefix = "<!-- BEGIN union:"
	endPrefix   = "<!-- END union:"
	markerSfx   = " -->"
)

// qualifiedRE matches "<store>:<clause-path>" where the store is a valid
// store name ([a-z0-9][a-z0-9_-]*) and the clause path has no whitespace.
// Structural clause-path rules (../, //, leading /) are enforced by callers
// via qpath.Parse; the marker parser just needs an unambiguous grammar.
var qualifiedRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*:[^\s]+$`)

// MarkedBlock is one ratified clause region inside a contract.
type MarkedBlock struct {
	Path      string
	StartLine int
	EndLine   int
	Body      []byte
}

// ParseContract scans body for union:<store>:<path> marker pairs.
func ParseContract(body []byte) ([]MarkedBlock, error) {
	lines := splitLines(body)
	var blocks []MarkedBlock
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if path, ok := parseBegin(line); ok {
			end := -1
			for j := i + 1; j < len(lines); j++ {
				if ep, ok := parseEnd(lines[j]); ok {
					if ep != path {
						return nil, fmt.Errorf("marker mismatch at line %d: BEGIN union:%s closed by END union:%s", i+1, path, ep)
					}
					end = j
					break
				}
				if np, ok := parseBegin(lines[j]); ok {
					return nil, fmt.Errorf("nested BEGIN union:%s inside BEGIN union:%s (line %d)", np, path, j+1)
				}
			}
			if end == -1 {
				return nil, fmt.Errorf("unterminated BEGIN union:%s at line %d", path, i+1)
			}
			bodyBytes := joinLines(lines[i+1 : end])
			blocks = append(blocks, MarkedBlock{Path: path, StartLine: i, EndLine: end, Body: bodyBytes})
			i = end + 1
			continue
		}
		if strings.HasPrefix(trimmed, beginPrefix) {
			return nil, fmt.Errorf("invalid BEGIN marker at line %d: expected <store>:<path>, got: %s", i+1, trimmed)
		}
		if path, ok := parseEnd(line); ok {
			return nil, fmt.Errorf("orphan END union:%s at line %d", path, i+1)
		}
		if strings.HasPrefix(trimmed, endPrefix) {
			return nil, fmt.Errorf("invalid END marker at line %d: expected <store>:<path>, got: %s", i+1, trimmed)
		}
		i++
	}
	return blocks, nil
}

// InsertClause appends a marked block for path. No-op if already present.
func InsertClause(contract []byte, path string, body []byte) ([]byte, error) {
	if HasClause(contract, path) {
		return append([]byte(nil), contract...), nil
	}
	var out bytes.Buffer
	out.Write(contract)
	if len(contract) > 0 {
		if !bytes.HasSuffix(contract, []byte("\n")) {
			out.WriteByte('\n')
		}
		if !bytes.HasSuffix(out.Bytes(), []byte("\n\n")) {
			out.WriteByte('\n')
		}
	}
	fmt.Fprintf(&out, "%s%s%s\n", beginPrefix, path, markerSfx)
	out.Write(body)
	if len(body) == 0 || body[len(body)-1] != '\n' {
		out.WriteByte('\n')
	}
	fmt.Fprintf(&out, "%s%s%s\n", endPrefix, path, markerSfx)
	return out.Bytes(), nil
}

// UpdateClause replaces the body of the marked block for path.
func UpdateClause(contract []byte, path string, body []byte) ([]byte, error) {
	blocks, err := ParseContract(contract)
	if err != nil {
		return nil, err
	}
	for _, b := range blocks {
		if b.Path != path {
			continue
		}
		lines := splitLines(contract)
		bodyLines := splitLines(body)
		var out []string
		out = append(out, lines[:b.StartLine+1]...)
		out = append(out, bodyLines...)
		out = append(out, lines[b.EndLine:]...)
		return joinLines(out), nil
	}
	return nil, fmt.Errorf("clause not present in contract: %s", path)
}

// RemoveClause strips the marked block for path (including a trailing blank line if present).
func RemoveClause(contract []byte, path string) ([]byte, error) {
	blocks, err := ParseContract(contract)
	if err != nil {
		return nil, err
	}
	for _, b := range blocks {
		if b.Path != path {
			continue
		}
		lines := splitLines(contract)
		start, end := b.StartLine, b.EndLine
		if end+1 < len(lines) && lines[end+1] == "" {
			end++
		}
		out := append([]string{}, lines[:start]...)
		out = append(out, lines[end+1:]...)
		return joinLines(out), nil
	}
	return nil, fmt.Errorf("clause not present in contract: %s", path)
}

// HasClause reports whether contract contains a BEGIN marker for path.
func HasClause(contract []byte, path string) bool {
	target := beginPrefix + path + markerSfx
	for _, line := range splitLines(contract) {
		if strings.TrimRight(line, " \t") == target {
			return true
		}
	}
	return false
}

func parseBegin(line string) (string, bool) {
	line = strings.TrimRight(line, " \t")
	if !strings.HasPrefix(line, beginPrefix) || !strings.HasSuffix(line, markerSfx) {
		return "", false
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, beginPrefix), markerSfx))
	if !qualifiedRE.MatchString(inner) {
		return "", false
	}
	return inner, true
}

func parseEnd(line string) (string, bool) {
	line = strings.TrimRight(line, " \t")
	if !strings.HasPrefix(line, endPrefix) || !strings.HasSuffix(line, markerSfx) {
		return "", false
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, endPrefix), markerSfx))
	if !qualifiedRE.MatchString(inner) {
		return "", false
	}
	return inner, true
}

func splitLines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}

func joinLines(lines []string) []byte {
	if len(lines) == 0 {
		return nil
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}
