package qpath

import "testing"

func TestParse_Valid(t *testing.T) {
	cases := []struct {
		in        string
		wantStore string
		wantPath  string
	}{
		{"personal:writing/voice", "personal", "writing/voice"},
		{"default:x", "default", "x"},
		{"a1_b-c:deep/path/to/clause", "a1_b-c", "deep/path/to/clause"},
	}
	for _, tc := range cases {
		q, err := Parse(tc.in)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tc.in, err)
			continue
		}
		if q.Store != tc.wantStore || q.Path != tc.wantPath {
			t.Errorf("Parse(%q) = {%q,%q}, want {%q,%q}", tc.in, q.Store, q.Path, tc.wantStore, tc.wantPath)
		}
		if q.String() != tc.in {
			t.Errorf("String() = %q, want %q", q.String(), tc.in)
		}
	}
}

func TestParse_Invalid(t *testing.T) {
	bad := []string{
		"",
		"no-colon",
		":empty-store",
		"empty-path:",
		"Bad:store",
		"has space:path",
		"store:/abs",
		"store:../escape",
		"store:a//b",
		"store:has space",
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", s)
		}
	}
}

func TestValidateStoreName(t *testing.T) {
	ok := []string{"a", "default", "personal", "work1", "a_b-c", "0abc"}
	for _, s := range ok {
		if err := ValidateStoreName(s); err != nil {
			t.Errorf("ValidateStoreName(%q) error: %v", s, err)
		}
	}
	bad := []string{"", "A", "has space", "has/slash", "has:colon", "-leading"}
	for _, s := range bad {
		if err := ValidateStoreName(s); err == nil {
			t.Errorf("ValidateStoreName(%q) succeeded, want error", s)
		}
	}
}
