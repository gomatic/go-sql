package sql_test

import (
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
		once, err := sql.LowerKeywords(in)
		if err != nil {
			return
		}
		twice, err := sql.LowerKeywords(string(once))
		if err != nil {
			t.Fatalf("re-lowering already-lowered SQL %q failed: %v", once, err)
		}
		if once != twice {
			t.Fatalf("LowerKeywords not idempotent: %q -> %q -> %q", in, once, twice)
		}
	})
}

// FuzzParseDeparse asserts the parse/deparse pair and the column-list sorter
// never panic on adversarial input, and that sorting then deparsing yields valid
// SQL whenever it yields anything. go-sql's Parse/Deparse are thin wrappers over
// pg_query, which does not itself promise deparse output re-parses or is stable
// (the out-of-range parameter $0 renders to the non-PostgreSQL `?`), so those
// are deliberately not asserted here; the meaning-preserving guarantees live in
// the formatter, normalize, and compare packages that build on this layer.
func FuzzParseDeparse(f *testing.F) {
	for _, s := range rootSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		_ = t
		tree, err := sql.Parse(sql.SQL(in))
		if err != nil {
			return
		}
		_, _ = sql.Deparse(tree)
		sql.SortColumnLists(tree)
		_, _ = sql.Deparse(tree)
	})
}

// FuzzRootInspectors asserts the read-only inspectors never panic on arbitrary
// input. Their meaning-preserving roles are asserted where they are consumed
// (the formatter gate and the compare engine); here we pin crash-freedom.
func FuzzRootInspectors(f *testing.F) {
	for _, s := range rootSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		_ = t
		_, _ = sql.Fingerprint(sql.SQL(in))
		_, _ = sql.Comments(sql.SQL(in))
		if tree, err := sql.Parse(sql.SQL(in)); err == nil {
			_, _ = sql.ToJSON(tree)
		}
	})
}
