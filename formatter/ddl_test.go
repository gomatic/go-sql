package formatter

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
)

func TestFormatCreateSchema(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create schema if not exists s")
	assert.NoError(t, err)
	assert.Equal(t, "create schema if not exists s;", out)
}

func TestFormatCreateSchemaPlain(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create schema s")
	assert.NoError(t, err)
	assert.Equal(t, "create schema s;", out)
}

func TestFormatCreateView(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create or replace view v as select a from t")
	assert.NoError(t, err)
	assert.Equal(t, "create or replace view v as\nselect a\nfrom t;", out)
}

func TestFormatCreateViewNilQuery(t *testing.T) {
	t.Parallel()
	out := New().formatCreateView(&pg_query.ViewStmt{View: &pg_query.RangeVar{Relname: "v"}})
	assert.Equal(t, "create view v as\n;", out)
}

func TestFormatCreateViewNonSelectQuery(t *testing.T) {
	t.Parallel()
	out := New().formatCreateView(&pg_query.ViewStmt{
		View:  &pg_query.RangeVar{Relname: "v"},
		Query: strNode("x"),
	})
	assert.Equal(t, "create view v as\n;", out)
}

func TestFormatCreateTable(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create table foo (id int primary key, name text not null)")
	assert.NoError(t, err)
	want := "create table foo (\n  id pg_catalog.int4 primary key,\n  name text not null\n);"
	assert.Equal(t, want, out)
}

func TestFormatCreateTableEmptyBody(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create table foo ()")
	assert.NoError(t, err)
	assert.Equal(t, "create table foo ();", out)
}

func TestFormatTableElementUnsupported(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/* unsupported table element */", New().formatTableElement(strNode("x")))
}

func TestFormatTypeNameDefaultsToText(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "text", New().formatTypeName(nil))
	assert.Equal(t, "text", New().formatTypeName(&pg_query.TypeName{}))
}

func TestFormatTypeNameWithModifiers(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create table t (a varchar(10), b numeric(5,2))")
	assert.NoError(t, err)
	assert.Contains(t, out, "a pg_catalog.varchar(10)")
	assert.Contains(t, out, "b pg_catalog.numeric(5, 2)")
}

func TestConstraintBodyKinds(t *testing.T) {
	t.Parallel()
	f := New()
	assert.Equal(t, "primary key", f.constraintBody(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_PRIMARY}))
	assert.Equal(t, "unique", f.constraintBody(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_UNIQUE}))
	assert.Equal(t, "check (a)", f.constraintBody(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_CHECK, RawExpr: columnRefNode("a")}))
	assert.Equal(t, "/* unknown constraint */", f.constraintBody(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_NOTNULL}))
}

func TestForeignKeyBody(t *testing.T) {
	t.Parallel()
	f := New()
	assert.Equal(t, "foreign key", f.foreignKeyBody(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_FOREIGN}))
	withTable := f.foreignKeyBody(&pg_query.Constraint{Pktable: &pg_query.RangeVar{Relname: "other"}})
	assert.Equal(t, "foreign key references other", withTable)
}

func TestFormatConstraintNamedWithKeys(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create table t (primary key (a, b))")
	assert.NoError(t, err)
	assert.Contains(t, out, "primary key (a, b)")
}

func TestConstraintKeysEmpty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, constraintKeys(nil))
}

func TestFormatConstraintNamed(t *testing.T) {
	t.Parallel()
	out := New().formatConstraint(&pg_query.Constraint{
		Conname: "pk",
		Contype: pg_query.ConstrType_CONSTR_PRIMARY,
	})
	assert.Equal(t, "constraint pk primary key", out)
}

func TestFormatColumnConstraintKinds(t *testing.T) {
	t.Parallel()
	f := New()
	assert.Equal(t, "not null", f.formatColumnConstraint(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_NOTNULL}))
	assert.Equal(t, "null", f.formatColumnConstraint(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_NULL}))
	assert.Equal(t, "default 1", f.formatColumnConstraint(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_DEFAULT, RawExpr: aconstIntNode(1)}))
	assert.Equal(t, "primary key", f.formatColumnConstraint(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_PRIMARY}))
	assert.Equal(t, "unique", f.formatColumnConstraint(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_UNIQUE}))
	assert.Equal(t, "/* unknown column constraint */", f.formatColumnConstraint(&pg_query.Constraint{Contype: pg_query.ConstrType_CONSTR_CHECK}))
}

func TestFormatCreateFunction(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create or replace function f(a int, out b text) returns int language sql")
	assert.NoError(t, err)
	want := "create or replace function f(a pg_catalog.int4, out b text)\n  returns pg_catalog.int4\n  language sql;"
	assert.Equal(t, want, out)
}

func TestFunctionParamsEmpty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, New().functionParams(nil))
}

func TestFormatFunctionParameterUnknownNode(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/* unknown parameter */", New().formatFunctionParameter(strNode("x")))
}

func TestFormatFunctionParameterDefault(t *testing.T) {
	t.Parallel()
	node := &pg_query.Node{Node: &pg_query.Node_FunctionParameter{FunctionParameter: &pg_query.FunctionParameter{
		Name:    "a",
		ArgType: &pg_query.TypeName{Names: []*pg_query.Node{strNode("int")}},
		Defexpr: aconstIntNode(7),
	}}}
	assert.Equal(t, "a int default 7", New().formatFunctionParameter(node))
}

func TestParamMode(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "out ", paramMode(pg_query.FunctionParameterMode_FUNC_PARAM_OUT))
	assert.Equal(t, "inout ", paramMode(pg_query.FunctionParameterMode_FUNC_PARAM_INOUT))
	assert.Empty(t, paramMode(pg_query.FunctionParameterMode_FUNC_PARAM_IN))
}

func TestFormatDefElem(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		elem *pg_query.DefElem
		want string
	}{
		{"as", &pg_query.DefElem{Defname: "as", Arg: strNode("body")}, "as\n$$\nbody\n$$"},
		{"language", &pg_query.DefElem{Defname: "language", Arg: strNode("sql")}, "language sql"},
		{"security", &pg_query.DefElem{Defname: "security", Arg: strNode("definer")}, "security definer"},
		{"volatility", &pg_query.DefElem{Defname: "volatility", Arg: strNode("immutable")}, "immutable"},
		{"unknown", &pg_query.DefElem{Defname: "cost", Arg: strNode("100")}, "cost"},
		{"no-arg", &pg_query.DefElem{Defname: "language"}, "language"},
		{"non-string-arg", &pg_query.DefElem{Defname: "as", Arg: intNode(1)}, "as"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, formatDefElem(c.elem), c.name)
	}
}

func TestFormatCreateTrigger(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create trigger trg before insert on t execute function fn()")
	assert.NoError(t, err)
	assert.Contains(t, out, "create trigger trg")
	assert.Contains(t, out, " on t execute function fn();")
}

func TestTriggerTiming(t *testing.T) {
	t.Parallel()
	assert.Equal(t, " before", triggerTiming(1))
	assert.Equal(t, " after", triggerTiming(2))
	assert.Equal(t, " instead of", triggerTiming(4))
	assert.Empty(t, triggerTiming(0))
}

func TestTriggerEvents(t *testing.T) {
	t.Parallel()
	assert.Equal(t, " insert or delete or update", triggerEvents(1|2|4))
	assert.Equal(t, " insert", triggerEvents(1))
	assert.Empty(t, triggerEvents(0))
}

func TestTriggerFunction(t *testing.T) {
	t.Parallel()
	assert.Empty(t, triggerFunction(nil))
	assert.Equal(t, " execute function fn()", triggerFunction([]*pg_query.Node{strNode("fn")}))
}

func TestFormatCreateCast(t *testing.T) {
	t.Parallel()
	withFunc, err := New().Format("create cast (int as text) with function f(int)")
	assert.NoError(t, err)
	assert.Equal(t, "create cast (pg_catalog.int4 as text) with function f;", withFunc)

	without, err := New().Format("create cast (int as text) without function")
	assert.NoError(t, err)
	assert.Equal(t, "create cast (pg_catalog.int4 as text) without function;", without)
}

func TestCastImplementationInout(t *testing.T) {
	t.Parallel()
	assert.Equal(t, " with inout", castImplementation(&pg_query.CreateCastStmt{Inout: true}))
}

func TestFormatCreateIndex(t *testing.T) {
	t.Parallel()
	out, err := New().Format("create unique index concurrently if not exists i on t using gin (a)")
	assert.NoError(t, err)
	assert.Equal(t, "create unique index concurrently if not exists i on t using gin (a);", out)
}

func TestIndexColumnsEmpty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, New().indexColumns(nil))
}

func TestFormatIndexElem(t *testing.T) {
	t.Parallel()
	f := New()
	assert.Equal(t, "a", f.formatIndexElem(&pg_query.IndexElem{Name: "a"}))
	assert.Equal(t, "c", f.formatIndexElem(&pg_query.IndexElem{Expr: columnRefNode("c")}))
	assert.Equal(t, "/* complex index element */", f.formatIndexElem(&pg_query.IndexElem{}))
}

func TestFormatAlterTableMultipleCommands(t *testing.T) {
	t.Parallel()
	out, err := New().Format("alter table t add column a int, drop column b")
	assert.NoError(t, err)
	want := "alter table t\n  add column a pg_catalog.int4,\n  drop column b;"
	assert.Equal(t, want, out)
}

func TestFormatAlterTableNonCommandNode(t *testing.T) {
	t.Parallel()
	out := New().formatAlterTable(&pg_query.AlterTableStmt{
		Relation: &pg_query.RangeVar{Relname: "t"},
		Cmds:     []*pg_query.Node{strNode("x")},
	})
	assert.Equal(t, "alter table t\n  ;", out)
}

func TestFormatAlterTableCmdKinds(t *testing.T) {
	t.Parallel()
	out, err := New().Format("alter table t alter column c type text, add constraint pk primary key (a), drop constraint ck")
	assert.NoError(t, err)
	assert.Contains(t, out, "alter column c type text")
	assert.Contains(t, out, "add constraint pk primary key (a)")
	assert.Contains(t, out, "drop constraint ck")
}

func TestFormatAlterTableCmdUnsupported(t *testing.T) {
	t.Parallel()
	out := New().formatAlterTableCmd(&pg_query.AlterTableCmd{Subtype: pg_query.AlterTableType_AT_SetStatistics})
	assert.Equal(t, "/* unsupported alter table command */", out)
}

func TestColumnNodeAccessorsNil(t *testing.T) {
	t.Parallel()
	f := New()
	assert.Empty(t, f.columnDefFromNode(nil))
	assert.Empty(t, f.columnDefFromNode(strNode("x")))
	assert.Empty(t, f.columnTypeFromNode(nil))
	assert.Empty(t, f.columnTypeFromNode(strNode("x")))
	assert.Empty(t, f.constraintFromNode(nil))
	assert.Empty(t, f.constraintFromNode(strNode("x")))
}
