package formatter

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gomatic/go-sql"
)

func TestNewDefaultIndent(t *testing.T) {
	t.Parallel()
	assert.Equal(t, defaultIndent, New().indentSize)
}

func TestFormatSimpleSelect(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t")
	require.NoError(t, err)
	assert.Equal(t, "select a\nfrom t", out)
}

func TestFormatMultipleStatementsJoinedByBlankLine(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select 1; select 2")
	require.NoError(t, err)
	assert.Equal(t, "select 1\n\nselect 2", out)
}

func TestFormatParseErrorWrapsErrParse(t *testing.T) {
	t.Parallel()
	_, err := New().Format("not valid sql ((")
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrParse)
}

func TestFormatUnsupportedStatementReturnsSentinel(t *testing.T) {
	t.Parallel()
	_, err := New().Format("set search_path = x")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedStatement)
}

func TestFormatStatementEmptyStmtRendersEmpty(t *testing.T) {
	t.Parallel()
	out, err := New().formatStatement(&pg_query.RawStmt{})
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestFormatStatementNilNodeReportsType(t *testing.T) {
	t.Parallel()
	_, err := New().formatStatement(&pg_query.RawStmt{Stmt: &pg_query.Node{}})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedStatement)
}

func TestBuilderWriteAppendsParts(t *testing.T) {
	t.Parallel()
	var b builder
	b.write("a", "b")
	b.write("c")
	assert.Equal(t, "abc", b.String())
}

func TestPad(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", pad(0))
	assert.Equal(t, "    ", pad(4))
}

func TestStringNodeValuesSkipsNonStrings(t *testing.T) {
	t.Parallel()
	values := stringNodeValues([]*pg_query.Node{strNode("a"), intNode(1), strNode("b")})
	assert.Equal(t, []string{"a", "b"}, values)
}

func TestJoinStringNodesDots(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "a.b", joinStringNodes([]*pg_query.Node{strNode("a"), strNode("b")}))
}
