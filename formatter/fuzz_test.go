package formatter

import (
	"testing"

	"github.com/gomatic/go-sql"
)

// formatSeeds walks the edge matrix for formatting: empty, whitespace-only,
// malformed, comments (which force the verbatim fallback), unbalanced quotes,
// unicode, multi-statement scripts, and assorted statement kinds.
var formatSeeds = []string{
	"",
	"   \t\n ",
	"SELECT   a  FROM t",
	"select 1; select 2",
	"select 1; select 2;",
	"-- keep me\nselect a from t",
	"select a /* c */, b from t",
	"not valid sql ((",
	"select 'unterminated",
	"select 名前 from 表 where x = '你好'",
	"create table t (a int, b text)",
	"grant select on t to r",
	"insert into t (b, a) select y, x from s",
	"seleCt \u2000", // U+2000 is a significant token to pg_query, not boundary whitespace
}

// FuzzFormat asserts the formatter never panics, is idempotent (formatting its
// own output is a no-op), always emits re-parseable SQL, and — for a single
// statement — never changes what the statement does (identical PostgreSQL
// fingerprint). These are the formatter's core promises: it never corrupts SQL
// and never changes a statement's meaning.
//
// Comment preservation is asserted by the unit tests, not here, because it is a
// per-statement guarantee: a comment that falls outside every statement's source
// span (a standalone or trailing comment between statements) is not carried into
// the canonical rendering — a known limitation, distinct from changing meaning.
func FuzzFormat(f *testing.F) {
	for _, s := range formatSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		out, err := New().Format(sql.SQL(in))
		if err != nil {
			return
		}
		reformatted, err := New().Format(sql.SQL(out))
		if err != nil {
			t.Fatalf("formatted output %q does not re-parse: %v", out, err)
		}
		if out != reformatted {
			t.Fatalf("Format not idempotent: %q -> %q -> %q", in, out, reformatted)
		}
		assertMeaningPreserved(t, in, out)
	})
}

// assertMeaningPreserved checks that a single-statement format left the
// statement's executable meaning unchanged: the input and the output share a
// PostgreSQL fingerprint. Multi-statement scripts are left to the idempotence and
// re-parse checks, since the per-statement gate already proves each faithful.
func assertMeaningPreserved(t *testing.T, in, out string) {
	t.Helper()
	tree, err := sql.Parse(sql.SQL(in))
	if err != nil || len(tree.Stmts) != 1 {
		return
	}
	inFP, err := sql.Fingerprint(sql.SQL(in))
	if err != nil {
		return
	}
	outFP, err := sql.Fingerprint(sql.SQL(out))
	if err != nil {
		t.Fatalf("formatted output %q has no fingerprint: %v", out, err)
	}
	if inFP != outFP {
		t.Fatalf("Format changed statement meaning: %q (%s) -> %q (%s)", in, inFP, out, outFP)
	}
}
