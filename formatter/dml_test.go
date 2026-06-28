package formatter

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
)

func TestFormatInsertSelect(t *testing.T) {
	t.Parallel()
	out, err := New().Format("insert into t (a, b) select 1, 2")
	assert.NoError(t, err)
	assert.Equal(t, "insert into t (a, b)\nselect 1\n   , 2;", out)
}

func TestFormatInsertNoColumns(t *testing.T) {
	t.Parallel()
	out, err := New().Format("insert into t select 1")
	assert.NoError(t, err)
	assert.Equal(t, "insert into t\nselect 1;", out)
}

func TestInsertColumnsEmpty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, insertColumns(nil))
}

func TestInsertColumnsSkipsNonResTarget(t *testing.T) {
	t.Parallel()
	assert.Equal(t, " ()", insertColumns([]*pg_query.Node{strNode("x")}))
}

func TestWriteInsertSourceNonSelect(t *testing.T) {
	t.Parallel()
	var b builder
	New().writeInsertSource(&b, strNode("x"))
	assert.Equal(t, "\n", b.String())
}

func TestFormatInsertOnConflictNothing(t *testing.T) {
	t.Parallel()
	out, err := New().Format("insert into t (a) select 1 on conflict (a) do nothing")
	assert.NoError(t, err)
	assert.Contains(t, out, "on conflict (a) do nothing;")
}

func TestFormatInsertOnConflictUpdate(t *testing.T) {
	t.Parallel()
	out, err := New().Format("insert into t (a) select 1 on conflict (a) do update set a = 2")
	assert.NoError(t, err)
	assert.Contains(t, out, "on conflict (a) do update set a = 2;")
}

func TestConflictTargetNilAndEmpty(t *testing.T) {
	t.Parallel()
	f := New()
	assert.Empty(t, f.conflictTarget(nil))
	assert.Empty(t, f.conflictTarget(&pg_query.InferClause{}))
}

func TestConflictTargetSkipsNonIndexElem(t *testing.T) {
	t.Parallel()
	out := New().conflictTarget(&pg_query.InferClause{IndexElems: []*pg_query.Node{strNode("x")}})
	assert.Equal(t, " ()", out)
}

func TestConflictActionDefault(t *testing.T) {
	t.Parallel()
	assert.Empty(t, New().conflictAction(&pg_query.OnConflictClause{Action: pg_query.OnConflictAction_ONCONFLICT_NONE}))
}

func TestFormatUpdateMultipleAssignments(t *testing.T) {
	t.Parallel()
	out, err := New().Format("update t set a = 1, b = 2")
	assert.NoError(t, err)
	assert.Equal(t, "update t\n  set a = 1,\n    b = 2;", out)
}

func TestFormatUpdateNoTargets(t *testing.T) {
	t.Parallel()
	out := New().formatUpdate(&pg_query.UpdateStmt{Relation: &pg_query.RangeVar{Relname: "t"}})
	assert.Equal(t, "update t;", out)
}

func TestFormatDeleteWithWhere(t *testing.T) {
	t.Parallel()
	out, err := New().Format("delete from t where a")
	assert.NoError(t, err)
	assert.Equal(t, "delete from t\nwhere a;", out)
}

func TestFormatDeleteNoWhere(t *testing.T) {
	t.Parallel()
	out, err := New().Format("delete from t")
	assert.NoError(t, err)
	assert.Equal(t, "delete from t;", out)
}

func TestAssignmentListSkipsNonResTarget(t *testing.T) {
	t.Parallel()
	assert.Empty(t, New().assignmentList([]*pg_query.Node{strNode("x")}, ", "))
}
