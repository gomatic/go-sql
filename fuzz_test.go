package sql_test

import (
	"fmt"
	"testing"

	sql "github.com/gomatic/go-sql"
)

// rootSeeds is the shared seed corpus for the root package's fuzz targets. It
// walks the edge matrix: empty, whitespace-only, malformed, unbalanced quotes
// and parens, line and block comments, unicode identifiers and literals, very
// long input, multi-statement scripts, and dollar-quoted bodies.
var rootSeeds = []string{
	"",
	"   \t\n ",
	"SELECT 1",
	"select   A , b  from T where x = 1",
	"INSERT INTO t (b, a) SELECT y, x FROM s",
	"-- line comment\nselect a from t",
	"select a /* block */, b from t",
	"select 1; select 2;",
	`SELECT "Mixed" FROM t WHERE x IN ('AND', 'OR')`,
	"NOT VALID SQL ((",
	"select 'unterminated",
	"select ((((1",
	"select 名前 from 表 where city = '你好世界'",
	"DO $$ BEGIN PERFORM 1; END $$",
	"CREATE TABLE t (a int PRIMARY KEY, b text)",
	"select " + longIdentifier(2000),
	// pg_query parses the out-of-range positional parameter $000000000 but
	// deparses it to the non-PostgreSQL `?`, which will not re-parse. This is a
	// quirk of the underlying deparser, not of go-sql; the fuzz invariants below
	// tolerate it rather than asserting it away.
	"select $000000000",
}

// longIdentifier builds an n-character identifier so a fuzz seed exercises very
// long input.
func longIdentifier(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

// FuzzLowerKeywords asserts [sql.LowerKeywords] never panics and is idempotent:
// once the input scans cleanly, lowering its keywords a second time changes
// nothing, because the keywords are already lower case.
func FuzzLowerKeywords(f *testing.F) {
	for _, s := range rootSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		once, err := sql.LowerKeywords(sql.SQL(in))
		if err != nil {
			return
		}
		twice, err := sql.LowerKeywords(once)
		if err != nil {
			t.Fatalf("re-lowering already-lowered SQL %q failed: %v", once, err)
		}
		if once != twice {
			t.Fatalf("LowerKeywords not idempotent: %q -> %q -> %q", in, once, twice)
		}
	})
}

// FuzzParseDeparse asserts the parse/deparse pair and the column-list sorter
// on adversarial input: beyond crash-freedom, the sorter is IDEMPOTENT —
// sorting a second time changes nothing the deparser can see. go-sql's
// Parse/Deparse are thin wrappers over pg_query, which does not itself promise
// deparse output re-parses or is stable (the out-of-range parameter $0 renders
// to the non-PostgreSQL `?`), so round-tripping is deliberately not asserted
// here; the meaning-preserving guarantees live in the formatter, normalize,
// and compare packages that build on this layer.
func FuzzParseDeparse(f *testing.F) {
	for _, s := range rootSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		tree, err := sql.Parse(sql.SQL(in))
		if err != nil {
			return
		}
		_, _ = sql.Deparse(tree)
		sql.SortColumnLists(tree)
		once, onceErr := sql.Deparse(tree)
		sql.SortColumnLists(tree)
		twice, twiceErr := sql.Deparse(tree)
		if (onceErr == nil) != (twiceErr == nil) || once != twice {
			t.Fatalf("SortColumnLists is not idempotent for %q: %q/%v vs %q/%v", in, once, onceErr, twice, twiceErr)
		}
	})
}

// FuzzRootInspectors asserts the read-only inspectors on arbitrary input:
// beyond crash-freedom, each is DETERMINISTIC — the same input yields the
// same fingerprint, the same comments, and the same JSON, every time. Their
// meaning-preserving roles are asserted where they are consumed (the
// formatter gate and the compare engine).
func FuzzRootInspectors(f *testing.F) {
	for _, s := range rootSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		print1, err1 := sql.Fingerprint(sql.SQL(in))
		print2, err2 := sql.Fingerprint(sql.SQL(in))
		if print1 != print2 || (err1 == nil) != (err2 == nil) {
			t.Fatalf("Fingerprint is not deterministic for %q", in)
		}
		comments1, _ := sql.Comments(sql.SQL(in))
		comments2, _ := sql.Comments(sql.SQL(in))
		if len(comments1) != len(comments2) {
			t.Fatalf("Comments is not deterministic for %q", in)
		}
		if tree, err := sql.Parse(sql.SQL(in)); err == nil {
			json1, jsonErr1 := sql.ToJSON(tree)
			json2, jsonErr2 := sql.ToJSON(tree)
			if fmt.Sprint(json1) != fmt.Sprint(json2) || (jsonErr1 == nil) != (jsonErr2 == nil) {
				t.Fatalf("ToJSON is not deterministic for %q", in)
			}
		}
	})
}
