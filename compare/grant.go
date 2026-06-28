package compare

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

type (
	grantObjectType string // grantObjectType is the snake_case kind of object a GRANT targets.
	grantType       string // grantType is "grant" or "revoke".
	granteeName     string // granteeName is the role a privilege is granted to.
)

// Grant directions.
const (
	grantDirectionGrant  grantType = "grant"
	grantDirectionRevoke grantType = "revoke"
)

// grantIdentity identifies a GRANT or REVOKE by direction, object, and grantee —
// for example, grant.table:schema.table:role.
func grantIdentity(stmt statementData) identity {
	data := extractStatementData(stmt)
	if data == nil {
		return ""
	}
	objType, _ := extractInt(data, "objtype")
	prefix := grantPrefix(extractGrantType(data))
	kind := mapGrantObjectType(objectTypeInt(objType))
	name := extractGrantObjectName(data)
	grantee := extractGrantee(data)
	return identity(prefix + identityPrefix(kind) + ":" + identityPrefix(name) + ":" + identityPrefix(grantee))
}

// grantPrefix picks the identity prefix that goes with the grant direction.
func grantPrefix(gt grantType) identityPrefix {
	if gt == grantDirectionGrant {
		return identityPrefixGrant
	}
	return identityPrefixRevoke
}

// extractGrantType tells you whether a statement grants or revokes.
func extractGrantType(data statementData) grantType {
	if isGrant, ok := data["is_grant"].(bool); ok && isGrant {
		return grantDirectionGrant
	}
	return grantDirectionRevoke
}

// mapGrantObjectType turns a pg_query ObjectType into a snake_case kind.
func mapGrantObjectType(t objectTypeInt) grantObjectType {
	name := strings.TrimPrefix(pg_query.ObjectType(t).String(), "OBJECT_")
	return grantObjectType(toSnakeCase(stringValue(name)))
}

// extractGrantObjectName reads the qualified name of the granted object.
func extractGrantObjectName(data statementData) qualifiedName {
	node := firstObjectNode(data, keyObjects)
	if node == nil {
		return ""
	}
	if rv := extractMap(node, keyRangeVar); rv != nil {
		return formatQualifiedName(extractSchemaAndName(rv))
	}
	if str := extractMap(node, keyStringNode); str != nil {
		return qualifiedName(extractString(str, keySval))
	}
	return ""
}

// extractGrantee reads the name of the first grantee role.
func extractGrantee(data statementData) granteeName {
	roleSpec := extractMap(firstObjectNode(data, "grantees"), keyRoleSpec)
	return granteeName(extractString(roleSpec, "rolename"))
}
