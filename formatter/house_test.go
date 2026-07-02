package formatter

import (
	"strings"
	"testing"

	errs "github.com/gomatic/go-error"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sql "github.com/gomatic/go-sql"
)

const boom errs.Const = "boom"

// parseStmt parses a single-statement query and returns its raw statement.
func parseStmt(t *testing.T, query string) *pg_query.RawStmt {
	t.Helper()
	tree, err := sql.Parse(sql.SQL(query))
	require.NoError(t, err)
	return tree.Stmts[0]
}

func TestFormatHouseSelectLeadingCommas(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a, b from t")
	require.NoError(t, err)
	assert.Equal(t, "select a\n     , b\n  from t", out)
}

func TestFormatHouseSelectWhereSplitsAnd(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t where x = 1 and y = 2")
	require.NoError(t, err)
	assert.Equal(t, "select a\n  from t\n where x = 1\n   and y = 2", out)
}

func TestFormatHouseSelectWithoutFrom(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select 1")
	require.NoError(t, err)
	assert.Equal(t, "select 1", out)
}

func TestFormatHouseSelectAliasesAndSchema(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a as x, b from s.t as u")
	require.NoError(t, err)
	assert.Equal(t, "select a as x\n     , b\n  from s.t as u", out)
}

func TestFormatHouseSelectKeepsExpressionsAndLowercases(t *testing.T) {
	t.Parallel()
	out, err := New().Format("SELECT a + b AS total FROM t WHERE x::int IN (1,2)")
	require.NoError(t, err)
	assert.Equal(t, "select a + b as total\n  from t\n where x::int in (1, 2)", out)
}

func TestFormatHouseBailsToCanonicalForOrderBy(t *testing.T) {
	t.Parallel()
	// order by isn't covered by the house renderer yet, so the gate falls back to
	// the canonical single-line form — still correct, just not house-styled.
	out, err := New().Format("select a, b from t order by a")
	require.NoError(t, err)
	assert.Equal(t, "select a, b from t order by a", out)
}

func TestFormatHouseBailsToCanonicalForJoin(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t1 join t2 on t1.id = t2.id")
	require.NoError(t, err)
	// A join in FROM isn't covered yet; the canonical deparse renders it on one line.
	assert.NotContains(t, out, "\n  from")
	assert.True(t, preservesMeaning("select a from t1 join t2 on t1.id = t2.id", sql.SQL(out)))
}

// The house output must always round-trip to the same statement.
func TestFormatHousePreservesMeaning(t *testing.T) {
	t.Parallel()
	for _, in := range []sql.SQL{
		"select a, b from t",
		"select a from t where x = 1 and y = 2",
		"select a as x, b from s.t as u",
	} {
		out, err := New().Format(in)
		require.NoError(t, err)
		assert.True(t, preservesMeaning(originalSQL(in), sql.SQL(out)), "input %q", in)
	}
}

func TestHouseStatementNonSelectBails(t *testing.T) {
	// An INSERT isn't a SELECT, so the house renderer declines and the canonical
	// deparse takes over.
	_, ok := houseStatement(parseStmt(t, "insert into t values (1)"))
	assert.False(t, ok)
}

func TestFormatInsertUsesCanonical(t *testing.T) {
	out, err := New().Format("INSERT INTO t VALUES (1)")
	require.NoError(t, err)
	assert.True(t, preservesMeaning("insert into t values (1)", sql.SQL(out)))
}

func TestHouseStatementBailsWhenLowercasingFails(t *testing.T) {
	original := houseLower
	t.Cleanup(func() { houseLower = original })
	houseLower = func(sql.SQL) (sql.SQL, error) { return "", boom }

	_, ok := houseStatement(parseStmt(t, "select a from t"))
	assert.False(t, ok)
}

func TestTargetTextsRejectsNonResTarget(t *testing.T) {
	// A node that isn't a result target can't be a select-list item.
	notATarget := &pg_query.Node{Node: &pg_query.Node_Integer{Integer: &pg_query.Integer{Ival: 1}}}
	_, ok := targetTexts([]*pg_query.Node{notATarget})
	assert.False(t, ok)
}

func TestFormatBailsWhenLeafWontDeparse(t *testing.T) {
	original := leafDeparse
	t.Cleanup(func() { leafDeparse = original })
	leafDeparse = func(*pg_query.Node) (string, error) { return "", boom }

	// Targets can't render, so the house renderer bails to the canonical deparse.
	out, err := New().Format("select a from t")
	require.NoError(t, err)
	assert.Equal(t, "select a from t", out)
}

func TestFormatBailsWhenWhereLeafWontDeparse(t *testing.T) {
	original := leafDeparse
	t.Cleanup(func() { leafDeparse = original })
	// Fail only the comparison leaves, so targets render but the WHERE doesn't.
	leafDeparse = func(node *pg_query.Node) (string, error) {
		text, err := deparseNode(node)
		if err != nil || strings.Contains(text, " = ") {
			return "", boom
		}
		return text, nil
	}

	single, err := New().Format("select x from t where a = 1")
	require.NoError(t, err)
	assert.Equal(t, "select x from t where a = 1", single)

	chained, err := New().Format("select x from t where a = 1 and b = 2")
	require.NoError(t, err)
	assert.Equal(t, "select x from t where a = 1 and b = 2", chained)
}
