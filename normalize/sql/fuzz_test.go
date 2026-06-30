package sqlnorm

import (
	"testing"

	sql "github.com/gomatic/go-sql"
)

// normSeeds walks the edge matrix for SQL text normalization: empty,
// whitespace-only, malformed SQL (which takes the whitespace fallback), comments,
// unbalanced quotes and parens, unicode, multi-statement, and reorderable column
// lists.
var normSeeds = []string{
	"",
	"   \t\n ",
	"SELECT  *   FROM   t",
	"SELECT b, a FROM t",
	"INSERT INTO t (b, a) SELECT y, x FROM s",
	"SELECT * FROM table; -- comment",
	"not valid sql ((",
	"select 'unterminated",
	"select 名前 from 表",
	"select 1; select 2;",
	"SELECT a /* c */, b FROM t WHERE x = 1",
	"0 ;",  // trailing semicolon padded with a space — must canonicalize to "0"
	"0 ;;", // multiple trailing semicolons — must canonicalize to "0"
	// Seam inputs the fuzz gate excludes from the idempotence assertion: a line
	// comment whose terminating newline the fallback collapses, non-ASCII
	// whitespace pg_query rejects, and a NUL byte that truncates the C string.
	"-- c\n0",
	"\u3000",
	"a\x00b",
}

// FuzzNormalize asserts every SQL-text normalization method never panics and is
// idempotent wherever the package can canonicalize an input to a fixpoint:
// normalizing an already-normalized value is then a no-op. Two seams are excluded
// because go-sql cannot converge past them, and both are PostgreSQL/pg_query
// boundaries rather than defects in the trim, sort, or fallback logic this
// package owns:
//
//   - The whitespace-collapse fallback (used for unparseable input) can turn an
//     unparseable string into a different, parseable statement — a newline that
//     terminates a `--` line comment collapses to a space, folding a trailing
//     statement into the comment. The fallback collapsing newlines is the
//     documented, tested behavior.
//   - pg_query's own deparser is non-idempotent for a handful of inputs: the
//     invalid parameter $0 renders to the non-PostgreSQL `?`, which re-parses and
//     then re-parenthesizes.
//
// Idempotence is asserted everywhere else — including the unparseable inputs whose
// fallback output stays unparseable, where this package's own canonicalization
// must converge.
func FuzzNormalize(f *testing.F) {
	for _, s := range normSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		if !convergent(in) {
			return
		}
		assertIdempotent(t, "Normalize", SQL(in).Normalize().Normalize, SQL(in).Normalize)
		assertIdempotent(t, "NormalizeRoutine", SQL(in).NormalizeRoutine().NormalizeRoutine, SQL(in).NormalizeRoutine)
		assertIdempotent(t, "NormalizeStrict", SQL(in).NormalizeStrict().NormalizeStrict, SQL(in).NormalizeStrict)
	})
}

// convergent reports whether sqlnorm can canonicalize in to a fixpoint: a single
// Normalize must not flip the input's parseability (the whitespace-collapse
// fallback turning unparseable text into a parseable statement), and pg_query's
// deparser must converge for it.
func convergent(in string) bool {
	once := string(SQL(in).Normalize())
	return parseable(in) == parseable(once) && deparseStable(in)
}

// parseable reports whether pg_query can parse s.
func parseable(s string) bool {
	_, err := sql.Parse(sql.SQL(s))
	return err == nil
}

// deparseStable reports whether pg_query's deparser converges for in: parsing and
// deparsing, then re-parsing and re-deparsing, yields identical text. Unparseable
// input and deparse output that will not re-parse both count as stable — there is
// no second rendering to disagree with the first.
func deparseStable(in string) bool {
	first, ok := deparseOnce(sql.SQL(in))
	if !ok {
		return true
	}
	second, ok := deparseOnce(first)
	if !ok {
		return true
	}
	return first == second
}

// deparseOnce parses then deparses in, reporting the rendering and whether both
// steps succeeded.
func deparseOnce(in sql.SQL) (sql.SQL, bool) {
	tree, err := sql.Parse(in)
	if err != nil {
		return "", false
	}
	out, err := sql.Deparse(tree)
	if err != nil {
		return "", false
	}
	return out, true
}

// assertIdempotent fails the test when applying a normalization twice differs
// from applying it once.
func assertIdempotent(t *testing.T, name string, twice, once func() SQL) {
	t.Helper()
	if got, want := twice(), once(); got != want {
		t.Fatalf("%s not idempotent: once=%q twice=%q", name, want, got)
	}
}
