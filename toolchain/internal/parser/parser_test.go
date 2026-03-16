package parser

import (
	"testing"
)

func TestParse(t *testing.T) {
	file, err := Parse("testdata/create.sqlm")
	if err != nil {
		t.Fatal(err)
	}

	if file.Package != "users" {
		t.Errorf("expected package %q, got %q", "users", file.Package)
	}
	if len(file.Imports) != 1 || file.Imports[0] != "queries" {
		t.Errorf("expected imports [queries], got %v", file.Imports)
	}
	if len(file.Inits) != 1 {
		t.Fatalf("expected 1 init block, got %d", len(file.Inits))
	}
	if file.Body == "" {
		t.Error("expected non-empty body")
	}

	t.Logf("package: %s", file.Package)
	t.Logf("imports: %v", file.Imports)
	t.Logf("init[0] (%d chars):\n%s", len(file.Inits[0]), file.Inits[0])
	t.Logf("body (%d chars):\n%s", len(file.Body), file.Body)
}
