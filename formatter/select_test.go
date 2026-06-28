package formatter

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
)

func TestFormatSelectMultiColumn(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a, b from t")
	assert.NoError(t, err)
	assert.Equal(t, "select a\n   , b\nfrom t", out)
}

func TestFormatSelectWithCTE(t *testing.T) {
	t.Parallel()
	out, err := New().Format("with c as (select 1) select x from c")
	assert.NoError(t, err)
	want := "with c as (\n" +
		"  --@formatter:off\n" +
		"  select 1\n" +
		"  --@formatter:on\n" +
		")\nselect x\nfrom c"
	assert.Equal(t, want, out)
}

func TestFormatSelectOrderBy(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t order by a desc")
	assert.NoError(t, err)
	assert.Equal(t, "select a\nfrom t\norder by a desc", out)
}

func TestFormatSelectUnionAll(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t1 union all select a from t2")
	assert.NoError(t, err)
	assert.Contains(t, out, "union all\nselect a\nfrom t2")
}

func TestSetOpKeyword(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "union", setOpKeyword(pg_query.SetOperation_SETOP_UNION))
	assert.Equal(t, "intersect", setOpKeyword(pg_query.SetOperation_SETOP_INTERSECT))
	assert.Equal(t, "except", setOpKeyword(pg_query.SetOperation_SETOP_EXCEPT))
	assert.Empty(t, setOpKeyword(pg_query.SetOperation_SETOP_NONE))
}

func TestFormatSelectIntersectAndExcept(t *testing.T) {
	t.Parallel()
	intersect, err := New().Format("select a from t1 intersect select a from t2")
	assert.NoError(t, err)
	assert.Contains(t, intersect, "intersect\nselect a")

	except, err := New().Format("select a from t1 except select a from t2")
	assert.NoError(t, err)
	assert.Contains(t, except, "except\nselect a")
}

func TestFormatCTENonCTENodeWritesNothing(t *testing.T) {
	t.Parallel()
	var b builder
	New().formatCTE(strNode("x"), &b, "", 0)
	assert.Empty(t, b.String())
}

func TestFormatCTEQueryNonSelectIsEmpty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, New().formatCTEQuery(strNode("x"), 0))
}

func TestFormatFromSubqueryIsComplex(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from (select 1) x")
	assert.NoError(t, err)
	assert.Contains(t, out, "/* complex from expression */")
}

func TestFormatFromAlias(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t x")
	assert.NoError(t, err)
	assert.Equal(t, "select a\nfrom t as x", out)
}

func TestFormatJoinTypes(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"select a from t1 join t2 on a":       "join t2",
		"select a from t1 left join t2 on a":  "left join t2",
		"select a from t1 right join t2 on a": "right join t2",
		"select a from t1 full join t2 on a":  "full join t2",
	}
	for query, want := range cases {
		out, err := New().Format(query)
		assert.NoError(t, err, query)
		assert.Contains(t, out, want, query)
	}
}

func TestFormatJoinWithoutQuals(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t1 cross join t2")
	assert.NoError(t, err)
	assert.Equal(t, "select a\nfrom t1\njoin t2", out)
}

func TestJoinKeyword(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "left join ", joinKeyword(pg_query.JoinType_JOIN_LEFT))
	assert.Equal(t, "right join ", joinKeyword(pg_query.JoinType_JOIN_RIGHT))
	assert.Equal(t, "full join ", joinKeyword(pg_query.JoinType_JOIN_FULL))
	assert.Equal(t, "join ", joinKeyword(pg_query.JoinType_JOIN_INNER))
}
