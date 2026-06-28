package formatter

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// formatComment renders a COMMENT ON statement.
func formatComment(stmt *pg_query.CommentStmt) string {
	var b builder
	b.write("comment on ")
	b.write(commentObjectType(stmt.Objtype))
	if stmt.Object != nil {
		b.write(formatObjectAddress(stmt.Object))
	}
	b.write(" is ")
	b.write(commentBody(stmt.Comment))
	b.write(";")
	return b.String()
}

// commentObjectType turns a comment's object type into its keyword.
func commentObjectType(objtype pg_query.ObjectType) string {
	switch objtype {
	case pg_query.ObjectType_OBJECT_TABLE:
		return "table "
	case pg_query.ObjectType_OBJECT_COLUMN:
		return "column "
	case pg_query.ObjectType_OBJECT_FUNCTION:
		return "function "
	case pg_query.ObjectType_OBJECT_VIEW:
		return "view "
	case pg_query.ObjectType_OBJECT_SCHEMA:
		return "schema "
	default:
		return "/* unknown object type */ "
	}
}

// commentBody renders a comment's text as a dollar-quoted literal, or null.
func commentBody(comment string) string {
	if comment == "" {
		return nullKw
	}
	return "$$" + comment + "$$"
}

// formatGrant renders a GRANT or REVOKE statement.
func formatGrant(stmt *pg_query.GrantStmt) string {
	var b builder
	b.write(grantVerb(stmt.IsGrant))
	b.write(grantPrivileges(stmt.Privileges))
	b.write(onKw)
	b.write(grantTargetType(stmt.Targtype))
	b.write(objectAddressList(stmt.Objects))
	b.write(grantDirection(stmt.IsGrant))
	b.write(grantees(stmt.Grantees))
	b.write(";")
	return b.String()
}

// grantVerb turns the grant flag into its leading verb.
func grantVerb(isGrant bool) string {
	if isGrant {
		return "grant "
	}
	return "revoke "
}

// grantDirection turns the grant flag into its role-direction keyword.
func grantDirection(isGrant bool) string {
	if isGrant {
		return " to "
	}
	return " from "
}

// grantPrivileges renders the comma-separated privilege list.
func grantPrivileges(privileges []*pg_query.Node) string {
	names := make([]string, 0, len(privileges))
	for _, priv := range privileges {
		if access := priv.GetAccessPriv(); access != nil {
			names = append(names, access.PrivName)
		}
	}
	return strings.Join(names, ", ")
}

// grantTargetType turns a grant target type into its keyword.
func grantTargetType(targtype pg_query.GrantTargetType) string {
	switch targtype {
	case pg_query.GrantTargetType_ACL_TARGET_OBJECT:
		return "table "
	case pg_query.GrantTargetType_ACL_TARGET_ALL_IN_SCHEMA:
		return "all tables in schema "
	default:
		return ""
	}
}

// grantees renders the comma-separated list of grantee roles.
func grantees(roles []*pg_query.Node) string {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		if spec := role.GetRoleSpec(); spec != nil {
			names = append(names, spec.Rolename)
		}
	}
	return strings.Join(names, ", ")
}

// formatDrop renders a DROP statement.
func formatDrop(stmt *pg_query.DropStmt) string {
	var b builder
	b.write("drop ")
	b.write(dropObjectType(stmt.RemoveType))
	if stmt.MissingOk {
		b.write(" if exists")
	}
	if len(stmt.Objects) > 0 {
		b.write(" ")
		b.write(objectAddressList(stmt.Objects))
	}
	b.write(dropBehavior(stmt.Behavior))
	b.write(";")
	return b.String()
}

// dropObjectType turns a dropped object type into its keyword.
func dropObjectType(objtype pg_query.ObjectType) string {
	switch objtype {
	case pg_query.ObjectType_OBJECT_TABLE:
		return "table"
	case pg_query.ObjectType_OBJECT_VIEW:
		return "view"
	case pg_query.ObjectType_OBJECT_FUNCTION:
		return "function"
	case pg_query.ObjectType_OBJECT_INDEX:
		return "index"
	case pg_query.ObjectType_OBJECT_SCHEMA:
		return "schema"
	case pg_query.ObjectType_OBJECT_TRIGGER:
		return "trigger"
	default:
		return "/* unknown object type */"
	}
}

// dropBehavior turns a drop behavior into its trailing keyword.
func dropBehavior(behavior pg_query.DropBehavior) string {
	switch behavior {
	case pg_query.DropBehavior_DROP_CASCADE:
		return " cascade"
	case pg_query.DropBehavior_DROP_RESTRICT:
		return " restrict"
	default:
		return ""
	}
}

// formatDo renders a DO statement, dropping each code block in verbatim.
func formatDo(stmt *pg_query.DoStmt) string {
	var b builder
	b.write("do\n$$\n")
	for _, code := range stmt.Args {
		b.write(doCodeBlock(code))
	}
	b.write("\n$$;")
	return b.String()
}

// doCodeBlock pulls the body text out of a DO statement's AS element.
func doCodeBlock(code *pg_query.Node) string {
	defElem := code.GetDefElem()
	if defElem == nil || defElem.Defname != defAs || defElem.Arg == nil {
		return ""
	}
	if s := defElem.Arg.GetString_(); s != nil {
		return s.Sval
	}
	return ""
}

// objectAddressList renders comma-separated object addresses.
func objectAddressList(objects []*pg_query.Node) string {
	names := make([]string, 0, len(objects))
	for _, obj := range objects {
		names = append(names, formatObjectAddress(obj))
	}
	return strings.Join(names, ", ")
}

// formatObjectAddress renders a dotted object address out of a list node.
func formatObjectAddress(obj *pg_query.Node) string {
	if list := obj.GetList(); list != nil {
		return joinStringNodes(list.Items)
	}
	return "/* complex object address */"
}

// formatObjectWithArgs renders a dotted function-or-operator object name.
func formatObjectWithArgs(obj *pg_query.ObjectWithArgs) string {
	return joinStringNodes(obj.Objname)
}
