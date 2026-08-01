package formatter

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gomatic/go-sql"
)

func TestFormatCanonicalisesWhitespaceAndCase(t *testing.T) {
	t.Parallel()
	// Messy whitespace and uppercase collapse to the canonical lowercase house
	// layout: one clause per line, FROM under SELECT.
	out, err := New().Format("SELECT   a  FROM t")
	require.NoError(t, err)
	assert.Equal(t, "select a\n  from t", out)
}

func TestFormatMultipleStatementsSeparatedBySemicolonAndBlankLine(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select 1; select 2")
	require.NoError(t, err)
	// The semicolon separator is what keeps the rendering valid SQL: the output
	// re-parses as the same two statements.
	assert.Equal(t, "select 1;\n\nselect 2", out)
	_, err = New().Format(sql.SQL(out))
	require.NoError(t, err, "formatted multi-statement output must re-parse")
}

func TestFormatParseErrorWrapsErrParse(t *testing.T) {
	t.Parallel()
	_, err := New().Format("not valid sql ((")
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrParse)
}

func TestFormatRendersEveryStatementKindFaithfully(t *testing.T) {
	t.Parallel()
	out, err := New().Format("set search_path = x")
	require.NoError(t, err)
	assert.True(t, preservesMeaning("set search_path = x", sql.SQL(out)))
}

func TestFormatPreservesCommentsByEmittingVerbatim(t *testing.T) {
	t.Parallel()
	out, err := New().Format("-- keep me\nselect a from t")
	require.NoError(t, err)
	assert.Equal(t, "-- keep me\nselect a from t", out)
}

func TestFormatStatementEmptyStmtRendersEmpty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, formatStatement("", &pg_query.RawStmt{}))
}

func TestFormatStatementNilNodeRendersEmpty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, formatStatement("", &pg_query.RawStmt{Stmt: &pg_query.Node{}}))
}

func TestStatementSourceSlicesInteriorStatement(t *testing.T) {
	t.Parallel()
	const query sql.SQL = "select 1;   select 2  ;"
	tree, err := sql.Parse(query)
	require.NoError(t, err)
	assert.Equal(t, "select 2", statementSource(query, tree.Stmts[1]))
}

func TestFormatPreservesMeaningWithNonASCIIWhitespace(t *testing.T) {
	t.Parallel()
	// U+2000 is whitespace to Go but a significant token to pg_query, so
	// "seleCt  " is a different statement from "seleCt". The verbatim
	// fallback must keep the U+2000 (trimming only PostgreSQL whitespace) so the
	// formatter never emits a slice that changed what the statement does.
	const query sql.SQL = "seleCt \u2000"
	out, err := New().Format(query)
	require.NoError(t, err)
	inFP, err := sql.Fingerprint(query)
	require.NoError(t, err)
	outFP, err := sql.Fingerprint(sql.SQL(out))
	require.NoError(t, err)
	assert.Equal(t, inFP, outFP, "format must not change the statement's fingerprint")
}

func TestStatementSourceClampsOverlongLength(t *testing.T) {
	t.Parallel()
	stmt := &pg_query.RawStmt{StmtLocation: 0, StmtLen: 999}
	assert.Equal(t, "select 1", statementSource("select 1", stmt))
}

// TestFormatIsIdempotent names the invariant statementSeparator documents. The
// separator carries a semicolon precisely so multi-statement output re-parses as
// the same statements; drop it and `select 1\n\nselect 2` is a syntax error, so
// the second pass over the formatter's own output fails or means something else.
// Formatting an already-formatted query must be a no-op — a canonical form that
// moves under re-application is not one.
func TestStatementSeparatorMakesFormatIdempotent(t *testing.T) {
	t.Parallel()
	for _, query := range []sql.SQL{
		"select 1",
		"SELECT 1; SELECT 2",
		"select a from t where b = 1;\nselect c from u;",
		"select 1;;",
	} {
		once, err := New().Format(query)
		require.NoError(t, err, "query %q", query)

		twice, err := New().Format(sql.SQL(once))
		require.NoError(t, err, "reformatting %q produced unparseable output: %q", query, once)
		assert.Equal(t, once, twice, "Format is not a fixed point on %q", query)
	}
}

// TestStatementSourceKeepsSignificantSpace names pgSpace's invariant: it is
// EXACTLY the whitespace PostgreSQL's lexer ignores, so a non-ASCII space such as
// U+00A0 is part of the statement rather than padding around it. Trimming with
// Go's wider unicode notion would cut a character the server would have read,
// producing a verbatim fallback that no longer means the same thing.
func TestPgSpaceExcludesSignificantWhitespace(t *testing.T) {
	t.Parallel()
	for _, r := range []rune{' ', ' ', '　'} {
		assert.NotContains(t, pgSpace, string(r),
			"U+%04X is significant to pg_query and must not be trimmed as boundary whitespace", r)
	}
	for _, r := range []rune{' ', '\t', '\n', '\r', '\f', '\v'} {
		assert.Contains(t, pgSpace, string(r), "%q is boundary whitespace to PostgreSQL", r)
	}
}
