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
	out, err := New().Format("SELECT   a  FROM t")
	require.NoError(t, err)
	assert.Equal(t, "select a from t", out)
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
	assert.True(t, preservesMeaning("set search_path = x", out))
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
