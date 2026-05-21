package harness

import (
	"os"
	"testing"
)

func TestVarsExpand(t *testing.T) {
	v := &Vars{
		ShopDir:     "/home/user/myapp",
		ShopName:    "myapp",
		UserEmail:   "dev@example.com",
		HarnessName: "claude",
		UnionDir:    "/home/user/.union",
	}

	tests := []struct {
		input string
		want  string
	}{
		{"echo {{shop.dir}}", "echo /home/user/myapp"},
		{"{{shop.name}}-build", "myapp-build"},
		{"notify {{user.email}}", "notify dev@example.com"},
		{"target={{harness.name}}", "target=claude"},
		{"--dir={{union.dir}}", "--dir=/home/user/.union"},
		{"no vars here", "no vars here"},
		{"{{shop.dir}}/scripts/{{shop.name}}.sh", "/home/user/myapp/scripts/myapp.sh"},
	}

	for _, tt := range tests {
		got := v.Expand(tt.input)
		if got != tt.want {
			t.Errorf("Expand(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestVarsExpandEnv(t *testing.T) {
	os.Setenv("UNION_TEST_VAR", "hello")
	defer os.Unsetenv("UNION_TEST_VAR")

	v := &Vars{}
	got := v.Expand("say {{env.UNION_TEST_VAR}}")
	if got != "say hello" {
		t.Errorf("expected 'say hello', got %q", got)
	}
}

func TestVarsExpandEnvMissing(t *testing.T) {
	os.Unsetenv("UNION_NONEXISTENT_VAR")
	v := &Vars{}
	got := v.Expand("val={{env.UNION_NONEXISTENT_VAR}}")
	if got != "val=" {
		t.Errorf("expected 'val=', got %q", got)
	}
}

func TestVarsExpandMultipleEnv(t *testing.T) {
	os.Setenv("UNION_A", "alpha")
	os.Setenv("UNION_B", "beta")
	defer os.Unsetenv("UNION_A")
	defer os.Unsetenv("UNION_B")

	v := &Vars{}
	got := v.Expand("{{env.UNION_A}}-{{env.UNION_B}}")
	if got != "alpha-beta" {
		t.Errorf("expected 'alpha-beta', got %q", got)
	}
}

func TestExpandHook(t *testing.T) {
	v := &Vars{
		ShopDir:     "/app",
		ShopName:    "app",
		HarnessName: "opencode",
	}
	h := Hook{
		Event:   "SessionStart",
		Command: "cd {{shop.dir}} && run",
		Matcher: "{{harness.name}}",
		Degrade: "skip",
	}
	expanded := v.ExpandHook(h)
	if expanded.Command != "cd /app && run" {
		t.Errorf("command: got %q", expanded.Command)
	}
	if expanded.Matcher != "opencode" {
		t.Errorf("matcher: got %q", expanded.Matcher)
	}
	if expanded.Event != "SessionStart" {
		t.Error("event should not change")
	}
}

func TestResolveVars(t *testing.T) {
	v := ResolveVars("/home/user/proj", "/home/user/.union", "claude")
	if v.ShopDir != "/home/user/proj" {
		t.Errorf("ShopDir: %q", v.ShopDir)
	}
	if v.ShopName != "proj" {
		t.Errorf("ShopName: %q", v.ShopName)
	}
	if v.HarnessName != "claude" {
		t.Errorf("HarnessName: %q", v.HarnessName)
	}
	if v.UnionDir != "/home/user/.union" {
		t.Errorf("UnionDir: %q", v.UnionDir)
	}
}
