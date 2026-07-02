package formatter

import (
	"errors"
	"testing"

	errs "github.com/gomatic/go-error"
	pg_query "github.com/pganalyze/pg_query_go/v6"

	sql "github.com/gomatic/go-sql"
)

// firstTargetVal parses a one-target SELECT and hands back that target's value
// expression node, for exercising leaf rendering.
func firstTargetVal(t *testing.T, query string) *pg_query.Node {
	t.Helper()
	tree, err := sql.Parse(sql.SQL(query))
	if err != nil {
		t.Fatal(err)
	}
	return tree.Stmts[0].Stmt.GetSelectStmt().TargetList[0].GetResTarget().Val
}

func TestDeparseNodeRendersArithmetic(t *testing.T) {
	got, err := deparseNode(firstTargetVal(t, "select a + b"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "a + b" {
		t.Fatalf("got %q, want %q", got, "a + b")
	}
}

func TestDeparseNodeReportsDeparseFailure(t *testing.T) {
	const boom errs.Const = "boom"
	failing := func(*pg_query.ParseResult) (string, error) { return "", boom }
	_, err := deparseNodeWith(failing, firstTargetVal(t, "select a"))
	if !errors.Is(err, ErrDeparseLeaf) {
		t.Fatalf("got %v, want ErrDeparseLeaf", err)
	}
}

// oneStatement parses a single-statement query and returns its raw statement.
func oneStatement(t *testing.T, query string) *pg_query.RawStmt {
	t.Helper()
	tree, err := sql.Parse(sql.SQL(query))
	if err != nil {
		t.Fatal(err)
	}
	return tree.Stmts[0]
}

func TestCanonicalStatementLowercasesDeparse(t *testing.T) {
	got, ok := canonicalStatement(oneStatement(t, "SELECT A FROM T"))
	if !ok || got != "select a from t" {
		t.Fatalf("got %q ok=%v, want %q true", got, ok, "select a from t")
	}
}

func TestCanonicalStatementFalseWhenDeparseFails(t *testing.T) {
	const boom errs.Const = "boom"
	failing := func(*pg_query.ParseResult) (string, error) { return "", boom }
	if _, ok := canonicalStatementWith(failing, sql.LowerKeywords, oneStatement(t, "select a")); ok {
		t.Fatal("want false when deparse fails")
	}
}

func TestCanonicalStatementFalseWhenLowerFails(t *testing.T) {
	const boom errs.Const = "boom"
	ok := func(*pg_query.ParseResult) (string, error) { return "select a", nil }
	failing := func(sql.SQL) (sql.SQL, error) { return "", boom }
	if _, got := canonicalStatementWith(ok, failing, oneStatement(t, "select a")); got {
		t.Fatal("want false when lowering fails")
	}
}
