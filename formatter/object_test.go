package formatter

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
)

func TestFormatComment(t *testing.T) {
	t.Parallel()
	out, err := New().Format("comment on table t is 'hi'")
	assert.NoError(t, err)
	assert.Equal(t, "comment on table t is $$hi$$;", out)
}

func TestFormatCommentNull(t *testing.T) {
	t.Parallel()
	out, err := New().Format("comment on table t is null")
	assert.NoError(t, err)
	assert.Equal(t, "comment on table t is null;", out)
}

func TestCommentObjectType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "table ", commentObjectType(pg_query.ObjectType_OBJECT_TABLE))
	assert.Equal(t, "column ", commentObjectType(pg_query.ObjectType_OBJECT_COLUMN))
	assert.Equal(t, "function ", commentObjectType(pg_query.ObjectType_OBJECT_FUNCTION))
	assert.Equal(t, "view ", commentObjectType(pg_query.ObjectType_OBJECT_VIEW))
	assert.Equal(t, "schema ", commentObjectType(pg_query.ObjectType_OBJECT_SCHEMA))
	assert.Equal(t, "/* unknown object type */ ", commentObjectType(pg_query.ObjectType_OBJECT_INDEX))
}

func TestCommentBody(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "null", commentBody(""))
	assert.Equal(t, "$$x$$", commentBody("x"))
}

func TestFormatGrant(t *testing.T) {
	t.Parallel()
	out, err := New().Format("grant select, update on all tables in schema s to r")
	assert.NoError(t, err)
	assert.Equal(t, "grant select, update on all tables in schema /* complex object address */ to r;", out)
}

func TestFormatRevoke(t *testing.T) {
	t.Parallel()
	out, err := New().Format("revoke select on table t from r")
	assert.NoError(t, err)
	assert.Equal(t, "revoke select on table /* complex object address */ from r;", out)
}

func TestGrantVerbAndDirection(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "grant ", grantVerb(true))
	assert.Equal(t, "revoke ", grantVerb(false))
	assert.Equal(t, " to ", grantDirection(true))
	assert.Equal(t, " from ", grantDirection(false))
}

func TestGrantPrivilegesSkipsNonAccessPriv(t *testing.T) {
	t.Parallel()
	assert.Empty(t, grantPrivileges([]*pg_query.Node{strNode("x")}))
}

func TestGrantTargetType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "table ", grantTargetType(pg_query.GrantTargetType_ACL_TARGET_OBJECT))
	assert.Equal(t, "all tables in schema ", grantTargetType(pg_query.GrantTargetType_ACL_TARGET_ALL_IN_SCHEMA))
	assert.Empty(t, grantTargetType(pg_query.GrantTargetType_ACL_TARGET_DEFAULTS))
}

func TestGranteesSkipsNonRoleSpec(t *testing.T) {
	t.Parallel()
	assert.Empty(t, grantees([]*pg_query.Node{strNode("x")}))
}

func TestFormatDrop(t *testing.T) {
	t.Parallel()
	out, err := New().Format("drop table if exists t cascade")
	assert.NoError(t, err)
	assert.Equal(t, "drop table if exists t cascade;", out)
}

func TestDropObjectType(t *testing.T) {
	t.Parallel()
	cases := map[pg_query.ObjectType]string{
		pg_query.ObjectType_OBJECT_TABLE:    "table",
		pg_query.ObjectType_OBJECT_VIEW:     "view",
		pg_query.ObjectType_OBJECT_FUNCTION: "function",
		pg_query.ObjectType_OBJECT_INDEX:    "index",
		pg_query.ObjectType_OBJECT_SCHEMA:   "schema",
		pg_query.ObjectType_OBJECT_TRIGGER:  "trigger",
		pg_query.ObjectType_OBJECT_COLUMN:   "/* unknown object type */",
	}
	for objtype, want := range cases {
		assert.Equal(t, want, dropObjectType(objtype))
	}
}

func TestDropBehavior(t *testing.T) {
	t.Parallel()
	assert.Equal(t, " cascade", dropBehavior(pg_query.DropBehavior_DROP_CASCADE))
	assert.Equal(t, " restrict", dropBehavior(pg_query.DropBehavior_DROP_RESTRICT))
	assert.Empty(t, dropBehavior(pg_query.DropBehavior(99)))
}

func TestFormatDropNoObjects(t *testing.T) {
	t.Parallel()
	out := formatDrop(&pg_query.DropStmt{RemoveType: pg_query.ObjectType_OBJECT_TABLE})
	assert.Equal(t, "drop table;", out)
}

func TestFormatDo(t *testing.T) {
	t.Parallel()
	out, err := New().Format("do $$ begin end $$")
	assert.NoError(t, err)
	assert.Equal(t, "do\n$$\n begin end \n$$;", out)
}

func TestDoCodeBlockVariants(t *testing.T) {
	t.Parallel()
	assert.Empty(t, doCodeBlock(strNode("x")))
	assert.Empty(t, doCodeBlock(defElemNode(&pg_query.DefElem{Defname: "other", Arg: strNode("y")})))
	assert.Empty(t, doCodeBlock(defElemNode(&pg_query.DefElem{Defname: "as"})))
	assert.Empty(t, doCodeBlock(defElemNode(&pg_query.DefElem{Defname: "as", Arg: intNode(1)})))
	assert.Equal(t, "body", doCodeBlock(defElemNode(&pg_query.DefElem{Defname: "as", Arg: strNode("body")})))
}

func TestObjectAddressListJoins(t *testing.T) {
	t.Parallel()
	listNode := &pg_query.Node{
		Node: &pg_query.Node_List{List: &pg_query.List{Items: []*pg_query.Node{strNode("a"), strNode("b")}}},
	}
	assert.Equal(t, "a.b", objectAddressList([]*pg_query.Node{listNode}))
}

func TestFormatObjectAddressComplex(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/* complex object address */", formatObjectAddress(strNode("x")))
}

func TestFormatObjectWithArgs(t *testing.T) {
	t.Parallel()
	obj := &pg_query.ObjectWithArgs{Objname: []*pg_query.Node{strNode("schema"), strNode("fn")}}
	assert.Equal(t, "schema.fn", formatObjectWithArgs(obj))
}
