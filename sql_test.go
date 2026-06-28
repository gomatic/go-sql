package sql

import (
	"reflect"
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInvalidWrapsErrParse(t *testing.T) {
	t.Parallel()
	_, err := Parse("NOT VALID SQL ((")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParse)
}

func TestParseValid(t *testing.T) {
	t.Parallel()
	tree, err := Parse("SELECT 1")
	require.NoError(t, err)
	require.Len(t, tree.Stmts, 1)
}

func TestDeparseRoundTrip(t *testing.T) {
	t.Parallel()
	tree, err := Parse("SELECT a FROM t")
	require.NoError(t, err)
	out, err := Deparse(tree)
	require.NoError(t, err)
	assert.Equal(t, SQL("SELECT a FROM t"), out)
}

func TestDeparseInvalidWrapsErrDeparse(t *testing.T) {
	t.Parallel()
	// A RawStmt holding an empty Node has no statement we can render.
	tree := &pg_query.ParseResult{Stmts: []*pg_query.RawStmt{{Stmt: &pg_query.Node{}}}}
	_, err := Deparse(tree)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDeparse)
}

func TestSortColumnListsNil(t *testing.T) {
	t.Parallel()
	assert.NotPanics(t, func() { SortColumnLists(nil) })
}

func TestTraverseAndSortInvalidValue(t *testing.T) {
	t.Parallel()
	// An invalid reflect.Value (say, the Elem of a nil interface) does nothing.
	assert.NotPanics(t, func() { traverseAndSort(reflect.Value{}) })
}

func TestSortColumnListsSimpleSelect(t *testing.T) {
	t.Parallel()
	tree, err := Parse("SELECT b, a")
	require.NoError(t, err)
	SortColumnLists(tree)
	out, err := Deparse(tree)
	require.NoError(t, err)
	assert.Equal(t, SQL("SELECT a, b"), out)
}

func TestSortColumnListsNonSimpleSelectUnchanged(t *testing.T) {
	t.Parallel()
	// GROUP BY makes the SELECT non-simple, so we leave the target order alone.
	tree, err := Parse("SELECT b, a FROM t GROUP BY a, b")
	require.NoError(t, err)
	SortColumnLists(tree)
	out, err := Deparse(tree)
	require.NoError(t, err)
	assert.Equal(t, SQL("SELECT b, a FROM t GROUP BY a, b"), out)
}

func TestSortColumnListsInsertSelect(t *testing.T) {
	t.Parallel()
	tree, err := Parse("INSERT INTO t (b, a) SELECT y, x")
	require.NoError(t, err)
	SortColumnLists(tree)
	out, err := Deparse(tree)
	require.NoError(t, err)
	assert.Equal(t, SQL("INSERT INTO t (a, b) SELECT x, y"), out)
}

func TestSortColumnListsInsertValuesMismatchIgnored(t *testing.T) {
	t.Parallel()
	// VALUES gives an empty target list, so the column count never lines up.
	tree, err := Parse("INSERT INTO t (b, a) VALUES (1, 2)")
	require.NoError(t, err)
	SortColumnLists(tree)
	out, err := Deparse(tree)
	require.NoError(t, err)
	assert.Equal(t, SQL("INSERT INTO t (b, a) VALUES (1, 2)"), out)
}

func TestInsertSelectNilCases(t *testing.T) {
	t.Parallel()
	assert.Nil(t, insertSelect(&pg_query.InsertStmt{}))
}

func TestResTargetNameMissing(t *testing.T) {
	t.Parallel()
	assert.Empty(t, resTargetName(&pg_query.Node{}))
}

func TestGetTargetName(t *testing.T) {
	t.Parallel()
	assert.Empty(t, getTargetName(&pg_query.Node{}))

	named := firstTarget(t, "SELECT a AS alias")
	assert.Equal(t, "alias", getTargetName(named))

	unnamed := firstTarget(t, "SELECT col")
	assert.Equal(t, "col", getTargetName(unnamed))
}

func TestGetNodeNameNil(t *testing.T) {
	t.Parallel()
	assert.Empty(t, getNodeName(nil))
	assert.Empty(t, getNodeName(&pg_query.Node{}))
}

func TestColumnRefName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "tbl.col", getNodeName(firstVal(t, "SELECT tbl.col")))
	assert.Equal(t, "*", getNodeName(firstVal(t, "SELECT *")))
}

func TestConstNameEachKind(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "abc", getNodeName(firstVal(t, "SELECT 'abc'")))
	assert.Equal(t, "42", getNodeName(firstVal(t, "SELECT 42")))
	assert.Equal(t, "1.5", getNodeName(firstVal(t, "SELECT 1.5")))
	assert.Equal(t, "b101", getNodeName(firstVal(t, "SELECT B'101'")))
	assert.Equal(t, "true", getNodeName(firstVal(t, "SELECT true")))
}

func TestConstNameUnset(t *testing.T) {
	t.Parallel()
	assert.Empty(t, constName(&pg_query.A_Const{}))
}

func TestFuncCallName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "now", getNodeName(firstVal(t, "SELECT now()")))
}

func TestAExprName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "a + b", getNodeName(firstVal(t, "SELECT a + b")))
	// An A_Expr with no operator name still renders both (empty) operands around
	// the operator slot — two spaces with an empty operator between them.
	assert.Equal(t, "  ", aExprName(&pg_query.A_Expr{}))
}

// firstTarget hands back the first ResTarget node of the SELECT in sql.
func firstTarget(t *testing.T, sql string) *pg_query.Node {
	t.Helper()
	tree, err := Parse(SQL(sql))
	require.NoError(t, err)
	return tree.Stmts[0].Stmt.GetSelectStmt().TargetList[0]
}

// firstVal hands back the value expression of the SELECT's first target.
func firstVal(t *testing.T, sql string) *pg_query.Node {
	t.Helper()
	return firstTarget(t, sql).GetResTarget().Val
}
