package shop

import (
	"strings"
	"testing"
)

func TestParseContract_Empty(t *testing.T) {
	blocks, err := ParseContract(nil)
	if err != nil {
		t.Fatalf("ParseContract: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("want 0 blocks, got %d", len(blocks))
	}
}

func TestParseContract_Single(t *testing.T) {
	in := []byte("preamble\n<!-- BEGIN union:default:base/identity -->\nhello\n<!-- END union:default:base/identity -->\ntrailing\n")
	blocks, err := ParseContract(in)
	if err != nil {
		t.Fatalf("ParseContract: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("want 1 block, got %d", len(blocks))
	}
	if blocks[0].Path != "default:base/identity" {
		t.Errorf("path = %q, want default:base/identity", blocks[0].Path)
	}
	if !strings.Contains(string(blocks[0].Body), "hello") {
		t.Errorf("body missing 'hello': %q", blocks[0].Body)
	}
}

func TestParseContract_Multiple(t *testing.T) {
	in := []byte(`# Contract

<!-- BEGIN union:default:base/identity -->
id
<!-- END union:default:base/identity -->

prose between

<!-- BEGIN union:default:lang/go -->
go rules
<!-- END union:default:lang/go -->
`)
	blocks, err := ParseContract(in)
	if err != nil {
		t.Fatalf("ParseContract: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("want 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Path != "default:base/identity" || blocks[1].Path != "default:lang/go" {
		t.Errorf("paths = %q, %q", blocks[0].Path, blocks[1].Path)
	}
}

func TestParseContract_OrphanBegin(t *testing.T) {
	in := []byte("<!-- BEGIN union:default:foo -->\ncontent\n")
	if _, err := ParseContract(in); err == nil {
		t.Fatal("expected error on orphan BEGIN")
	}
}

func TestParseContract_OrphanEnd(t *testing.T) {
	in := []byte("<!-- END union:default:foo -->\n")
	if _, err := ParseContract(in); err == nil {
		t.Fatal("expected error on orphan END")
	}
}

func TestParseContract_MismatchedPath(t *testing.T) {
	in := []byte("<!-- BEGIN union:default:foo -->\nx\n<!-- END union:default:bar -->\n")
	if _, err := ParseContract(in); err == nil {
		t.Fatal("expected error on mismatched path")
	}
}

func TestParseContract_RejectsLegacyUnqualified(t *testing.T) {
	in := []byte("<!-- BEGIN union:base/identity -->\nhi\n<!-- END union:base/identity -->\n")
	if _, err := ParseContract(in); err == nil {
		t.Fatal("expected error on unqualified marker")
	}
}

func TestParseContract_CrossStore(t *testing.T) {
	in := []byte("<!-- BEGIN union:personal:a -->\n1\n<!-- END union:personal:a -->\n<!-- BEGIN union:work:b -->\n2\n<!-- END union:work:b -->\n")
	blocks, err := ParseContract(in)
	if err != nil {
		t.Fatalf("ParseContract: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("want 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Path != "personal:a" || blocks[1].Path != "work:b" {
		t.Errorf("paths = %q, %q", blocks[0].Path, blocks[1].Path)
	}
}

func TestInsertClause_EmptyFile(t *testing.T) {
	got, err := InsertClause(nil, "default:base/identity", []byte("hello"))
	if err != nil {
		t.Fatalf("InsertClause: %v", err)
	}
	want := "<!-- BEGIN union:default:base/identity -->\nhello\n<!-- END union:default:base/identity -->\n"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInsertClause_AppendsWithBlankLine(t *testing.T) {
	base := []byte("# Existing\n\nSome prose.\n")
	got, err := InsertClause(base, "default:base/identity", []byte("hello"))
	if err != nil {
		t.Fatalf("InsertClause: %v", err)
	}
	want := "# Existing\n\nSome prose.\n\n<!-- BEGIN union:default:base/identity -->\nhello\n<!-- END union:default:base/identity -->\n"
	if string(got) != want {
		t.Errorf("got:\n%q\n\nwant:\n%q", got, want)
	}
}

func TestInsertClause_DuplicateIsNoop(t *testing.T) {
	in := []byte("<!-- BEGIN union:default:base/identity -->\nx\n<!-- END union:default:base/identity -->\n")
	got, err := InsertClause(in, "default:base/identity", []byte("new"))
	if err != nil {
		t.Fatalf("InsertClause: %v", err)
	}
	if string(got) != string(in) {
		t.Errorf("duplicate insert changed file:\ngot %q\nwant %q", got, in)
	}
}

func TestUpdateClause_ReplacesBody(t *testing.T) {
	in := []byte("pre\n<!-- BEGIN union:default:x -->\nold\n<!-- END union:default:x -->\npost\n")
	got, err := UpdateClause(in, "default:x", []byte("NEW"))
	if err != nil {
		t.Fatalf("UpdateClause: %v", err)
	}
	want := "pre\n<!-- BEGIN union:default:x -->\nNEW\n<!-- END union:default:x -->\npost\n"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestUpdateClause_Missing(t *testing.T) {
	in := []byte("nothing here\n")
	if _, err := UpdateClause(in, "default:x", []byte("y")); err == nil {
		t.Fatal("expected error for missing clause")
	}
}

func TestRemoveClause(t *testing.T) {
	in := []byte("pre\n<!-- BEGIN union:default:x -->\nbody\n<!-- END union:default:x -->\npost\n")
	got, err := RemoveClause(in, "default:x")
	if err != nil {
		t.Fatalf("RemoveClause: %v", err)
	}
	want := "pre\npost\n"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRemoveClause_Missing(t *testing.T) {
	if _, err := RemoveClause([]byte("x"), "default:nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestHasClause(t *testing.T) {
	in := []byte("<!-- BEGIN union:default:x -->\nb\n<!-- END union:default:x -->\n")
	if !HasClause(in, "default:x") {
		t.Error("HasClause(default:x) = false, want true")
	}
	if HasClause(in, "default:y") {
		t.Error("HasClause(default:y) = true, want false")
	}
}
