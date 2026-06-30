package sql_test

import (
	"errors"
	"testing"

	sql "github.com/gomatic/go-sql"
)

func TestLowerKeywordsLowercasesKeywordsOnly(t *testing.T) {
	got, err := sql.LowerKeywords("SELECT A AND B")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "select A and B" {
		t.Fatalf("got %q, want %q", got, "select A and B")
	}
}

func TestLowerKeywordsPreservesStringLiterals(t *testing.T) {
	got, err := sql.LowerKeywords("x IN ('AND', 'OR')")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "x in ('AND', 'OR')" {
		t.Fatalf("got %q, want %q", got, "x in ('AND', 'OR')")
	}
}

func TestLowerKeywordsPreservesQuotedIdentifiers(t *testing.T) {
	got, err := sql.LowerKeywords(`SELECT "Mixed" FROM t`)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `select "Mixed" from t` {
		t.Fatalf("got %q, want %q", got, `select "Mixed" from t`)
	}
}

func TestLowerKeywordsPreservesDollarQuotedBody(t *testing.T) {
	got, err := sql.LowerKeywords("DO $$ BEGIN PERFORM 1; END $$")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "do $$ BEGIN PERFORM 1; END $$" {
		t.Fatalf("got %q, want %q", got, "do $$ BEGIN PERFORM 1; END $$")
	}
}

func TestLowerKeywordsReportsScanFailure(t *testing.T) {
	_, err := sql.LowerKeywords("'unterminated")
	if !errors.Is(err, sql.ErrScan) {
		t.Fatalf("got %v, want ErrScan", err)
	}
}
