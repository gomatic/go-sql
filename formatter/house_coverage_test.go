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

func TestHouseStatementNonSelectBails(t *testing.T) {
	// An INSERT isn't a SELECT, so the house renderer declines and the canonical
	// deparse takes over.
	_, ok := houseStatement(parseStmt(t, "insert into t values (1)"))
	assert.False(t, ok)
}

func TestFormatInsertUsesCanonical(t *testing.T) {
	out, err := New().Format("INSERT INTO t VALUES (1)")
	require.NoError(t, err)
	assert.True(t, preservesMeaning("insert into t values (1)", out))
}

func TestHouseStatementBailsWhenLowercasingFails(t *testing.T) {
	original := houseLower
	t.Cleanup(func() { houseLower = original })
	houseLower = func(string) (sql.SQL, error) { return "", boom }

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
