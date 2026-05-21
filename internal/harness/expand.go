package harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Vars holds template variable values resolved at emit time.
type Vars struct {
	ShopDir     string
	ShopName    string
	UserEmail   string
	HarnessName string
	UnionDir    string
}

// Expand replaces {{var}} placeholders in s with resolved values.
// Supports built-in vars and {{env.NAME}} for environment variables.
func (v *Vars) Expand(s string) string {
	r := strings.NewReplacer(
		"{{shop.dir}}", v.ShopDir,
		"{{shop.name}}", v.ShopName,
		"{{user.email}}", v.UserEmail,
		"{{harness.name}}", v.HarnessName,
		"{{union.dir}}", v.UnionDir,
	)
	s = r.Replace(s)
	s = expandEnvVars(s)
	return s
}

// ExpandHook returns a copy of h with Command expanded.
func (v *Vars) ExpandHook(h Hook) Hook {
	h.Command = v.Expand(h.Command)
	h.Matcher = v.Expand(h.Matcher)
	return h
}

// ResolveVars builds a Vars from the current environment.
// harnessName is filled per-adapter during emit.
func ResolveVars(shopDir, unionDir, harnessName string) *Vars {
	return &Vars{
		ShopDir:     shopDir,
		ShopName:    filepath.Base(shopDir),
		UserEmail:   gitUserEmail(),
		HarnessName: harnessName,
		UnionDir:    unionDir,
	}
}

func gitUserEmail() string {
	out, err := exec.Command("git", "config", "user.email").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// expandEnvVars replaces all {{env.NAME}} with os.Getenv("NAME").
func expandEnvVars(s string) string {
	for {
		start := strings.Index(s, "{{env.")
		if start < 0 {
			return s
		}
		end := strings.Index(s[start:], "}}")
		if end < 0 {
			return s
		}
		end += start
		name := s[start+len("{{env.") : end]
		val := os.Getenv(name)
		s = s[:start] + val + s[end+2:]
	}
}
