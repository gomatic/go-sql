package formatter

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
)

func TestFormatExprNil(t *testing.T) {
	t.Parallel()
	assert.Empty(t, New().formatExpr(nil, 0))
}

func TestFormatExprColumnAndConst(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "a", New().formatExpr(columnRefNode("a"), 0))
	assert.Equal(t, "1", New().formatExpr(aconstIntNode(1), 0))
}

func TestFormatExprUnsupportedIsComment(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t where x = 1")
	assert.NoError(t, err)
	assert.Contains(t, out, "/* complex expression */")
}

func TestFormatColumnRefQualified(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select t.a from t")
	assert.NoError(t, err)
	assert.Equal(t, "select t.a\nfrom t", out)
}

func TestFormatAConstVariants(t *testing.T) {
	t.Parallel()
	cases := map[string]*pg_query.A_Const{
		"null":   {Isnull: true},
		"'x'":    {Val: &pg_query.A_Const_Sval{Sval: &pg_query.String{Sval: "x"}}},
		"42":     {Val: &pg_query.A_Const_Ival{Ival: &pg_query.Integer{Ival: 42}}},
		"1.5":    {Val: &pg_query.A_Const_Fval{Fval: &pg_query.Float{Fval: "1.5"}}},
		"true":   {Val: &pg_query.A_Const_Boolval{Boolval: &pg_query.Boolean{Boolval: true}}},
		"b'101'": {Val: &pg_query.A_Const_Bsval{Bsval: &pg_query.BitString{Bsval: "b'101'"}}},
	}
	for want, c := range cases {
		assert.Equal(t, want, formatAConst(c), want)
	}
}

func TestFormatAConstEmptyFallsThroughToNull(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "null", formatAConst(&pg_query.A_Const{}))
}

func TestBoolText(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "true", boolText(true))
	assert.Equal(t, "false", boolText(false))
}

func TestFormatBoolExprAnd(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t where x and y")
	assert.NoError(t, err)
	assert.Equal(t, "select a\nfrom t\nwhere x\n  and y", out)
}

func TestFormatBoolExprOr(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t where x or y")
	assert.NoError(t, err)
	assert.Equal(t, "select a\nfrom t\nwhere x\n  or y", out)
}

func TestBoolOpKeyword(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "or ", boolOpKeyword(pg_query.BoolExprType_OR_EXPR))
	assert.Equal(t, "and ", boolOpKeyword(pg_query.BoolExprType_AND_EXPR))
}

func TestFormatFuncCallArgs(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select f(a, b) from t")
	assert.NoError(t, err)
	assert.Equal(t, "select f(a, b)\nfrom t", out)
}

func TestSortDirection(t *testing.T) {
	t.Parallel()
	assert.Equal(t, " desc", sortDirection(pg_query.SortByDir_SORTBY_DESC))
	assert.Equal(t, " asc", sortDirection(pg_query.SortByDir_SORTBY_ASC))
	assert.Empty(t, sortDirection(pg_query.SortByDir_SORTBY_DEFAULT))
}

func TestFormatSortByDefaultDirection(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t order by a")
	assert.NoError(t, err)
	assert.Equal(t, "select a\nfrom t\norder by a", out)
}

func TestFormatResTargetAlias(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a as x from t")
	assert.NoError(t, err)
	assert.Equal(t, "select a as x\nfrom t", out)
}

func TestFormatRangeVarSchemaQualified(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "s.t", formatRangeVar(&pg_query.RangeVar{Schemaname: "s", Relname: "t"}))
	assert.Equal(t, "t", formatRangeVar(&pg_query.RangeVar{Relname: "t"}))
	assert.Equal(t, "s", formatRangeVar(&pg_query.RangeVar{Schemaname: "s"}))
}
