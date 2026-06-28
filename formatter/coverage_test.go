package formatter

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
)

func TestFormatCreateTableIfNotExists(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create table if not exists t (a int)")
	assert.NoError(t, err)
	assert.Equal(t, "create table if not exists t (\n  a pg_catalog.int4\n);", out)
}

func TestConstraintBodyForeign(t *testing.T) {
	t.Parallel()
	out := New().constraintBody(&pg_query.Constraint{
		Contype: pg_query.ConstrType_CONSTR_FOREIGN,
		Pktable: &pg_query.RangeVar{Relname: "other"},
	})
	assert.Equal(t, "foreign key references other", out)
}

func TestWriteFunctionReturnsNil(t *testing.T) {
	t.Parallel()
	var b builder
	New().writeFunctionReturns(&b, nil)
	assert.Empty(t, b.String())
}

func TestFormatCreateOrReplaceTrigger(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create or replace trigger trg before insert on t execute function fn()")
	assert.NoError(t, err)
	assert.Contains(t, out, "create or replace trigger trg")
}

func TestWriteInsertSourceNil(t *testing.T) {
	t.Parallel()
	var b builder
	New().writeInsertSource(&b, nil)
	assert.Empty(t, b.String())
}

func TestFormatSelectMultipleCTEs(t *testing.T) {
	t.Parallel()
	out, err := New().Format("with a as (select 1), b as (select 2) select x from a")
	assert.NoError(t, err)
	assert.Contains(t, out, "with a as (")
	assert.Contains(t, out, ",\n     b as (")
}

func TestFormatSelectMultipleFromItems(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t1, t2")
	assert.NoError(t, err)
	assert.Equal(t, "select a\nfrom t1, t2", out)
}

func TestFormatSelectMultipleOrderBy(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t order by a, b")
	assert.NoError(t, err)
	assert.Equal(t, "select a\nfrom t\norder by a, b", out)
}
