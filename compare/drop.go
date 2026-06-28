package compare

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

type (
	dropObjectType string // dropObjectType is the snake_case kind of a dropped object.
	objectTypeInt  int    // objectTypeInt is a pg_query ObjectType value.
)

// Object kinds DROP recognizes, lined up with the pg_query ObjectType values.
const (
	dropObjectFunction objectTypeInt = objectTypeInt(pg_query.ObjectType_OBJECT_FUNCTION)
	dropObjectIndex    objectTypeInt = objectTypeInt(pg_query.ObjectType_OBJECT_INDEX)
	dropObjectSchema   objectTypeInt = objectTypeInt(pg_query.ObjectType_OBJECT_SCHEMA)
	dropObjectTable    objectTypeInt = objectTypeInt(pg_query.ObjectType_OBJECT_TABLE)
	dropObjectTrigger  objectTypeInt = objectTypeInt(pg_query.ObjectType_OBJECT_TRIGGER)
	dropObjectView     objectTypeInt = objectTypeInt(pg_query.ObjectType_OBJECT_VIEW)
)

// Snake_case kind names for the object kinds DROP knows about.
const (
	kindFunction dropObjectType = "function"
	kindIndex    dropObjectType = "index"
	kindSchema   dropObjectType = "schema"
	kindTable    dropObjectType = "table"
	kindTrigger  dropObjectType = "trigger"
	kindView     dropObjectType = "view"
)

// dropIdentity identifies a DROP by object kind and name — for example,
// drop.table:schema.table.
func dropIdentity(stmt statementData) identity {
	data := extractStatementData(stmt)
	if data == nil {
		return ""
	}
	removeType, found := extractInt(data, "remove_type")
	if !found {
		return ""
	}
	kind := mapObjectType(objectTypeInt(removeType))
	name := extractDropObjectName(data)
	return identity(identityPrefixDrop + identityPrefix(kind) + ":" + identityPrefix(name))
}

// mapObjectType names a dropped object's kind. For kinds we don't handle, it
// falls back to a snake_case rendering of the pg_query enum.
func mapObjectType(t objectTypeInt) dropObjectType {
	switch t {
	case dropObjectTable:
		return kindTable
	case dropObjectView:
		return kindView
	case dropObjectFunction:
		return kindFunction
	case dropObjectSchema:
		return kindSchema
	case dropObjectIndex:
		return kindIndex
	case dropObjectTrigger:
		return kindTrigger
	default:
		return dropObjectType("unknown_" + string(toSnakeCase(stringValue(pg_query.ObjectType(t).String()))))
	}
}

// extractDropObjectName reads the dropped object's qualified name from the first
// listed object, switching on its node shape.
func extractDropObjectName(data statementData) qualifiedName {
	node := firstObjectNode(data, keyObjects)
	if node == nil {
		return ""
	}
	if owa := extractMap(node, keyObjWithArgs); owa != nil {
		return extractFunctionDropName(owa)
	}
	if str := extractMap(node, keyStringNode); str != nil {
		return qualifiedName(extractString(str, keySval))
	}
	if list := extractMap(node, keyList); list != nil {
		return extractListName(list)
	}
	return ""
}

// firstObjectNode returns the node wrapper of the first element in a list field.
func firstObjectNode(data statementData, key fieldKey) statementData {
	items := extractSlice(data, key)
	if len(items) == 0 {
		return nil
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		return nil
	}
	return extractMap(statementData(first), keyNode)
}

// extractFunctionDropName reads a function's name and argument types out of an
// ObjectWithArgs node — for example, schema.fn(int4,text).
func extractFunctionDropName(objWithArgs statementData) qualifiedName {
	parts := extractNodeStringParts(extractSlice(objWithArgs, keyObjname))
	if len(parts) == 0 {
		return ""
	}
	name := strings.Join(parts, ".")
	if argTypes := extractDropArgTypes(objWithArgs); len(argTypes) > 0 {
		name += "(" + strings.Join(argTypes, ",") + ")"
	}
	return qualifiedName(name)
}

// extractNodeStringParts collects the string values out of node-wrapped String_
// items.
func extractNodeStringParts(items []any) []string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if sval := extractNodeStringValue(item); sval != "" {
			parts = append(parts, string(sval))
		}
	}
	return parts
}

// extractNodeStringValue reads the sval off a node-wrapped String_ item.
func extractNodeStringValue(item any) stringValue {
	m, ok := item.(map[string]any)
	if !ok {
		return ""
	}
	return extractString(extractMap(extractMap(statementData(m), keyNode), keyStringNode), keySval)
}

// extractDropArgTypes collects the argument type names from a DROP FUNCTION.
func extractDropArgTypes(objWithArgs statementData) []string {
	args := extractSlice(objWithArgs, keyObjargs)
	types := make([]string, 0, len(args))
	for _, arg := range args {
		if tn := extractDropArgTypeName(arg); tn != "" {
			types = append(types, string(tn))
		}
	}
	return types
}

// extractDropArgTypeName reads the type name of a single argument.
func extractDropArgTypeName(arg any) typeName {
	m, ok := arg.(map[string]any)
	if !ok {
		return ""
	}
	return extractTypeName(extractMap(extractMap(statementData(m), keyNode), keyTypeName))
}

// extractListName builds a dotted name from a List node's items.
func extractListName(listNode statementData) qualifiedName {
	return qualifiedName(strings.Join(extractNodeStringParts(extractSlice(listNode, "items")), "."))
}
