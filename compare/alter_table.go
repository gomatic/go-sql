package compare

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// alterSubtype is a pg_query AlterTableType value.
type alterSubtype int

// Alter-table subtypes whose target name we dig out of the command's def node.
const (
	alterAddColumn      alterSubtype = alterSubtype(pg_query.AlterTableType_AT_AddColumn)
	alterAddConstraint  alterSubtype = alterSubtype(pg_query.AlterTableType_AT_AddConstraint)
	alterDropConstraint alterSubtype = alterSubtype(pg_query.AlterTableType_AT_DropConstraint)
)

type (
	alterSubtypeName string // alterSubtypeName is the snake_case name of an alter-table subtype.
	alterTargetName  string // alterTargetName is the column or constraint an ALTER TABLE targets.
)

// alterTableIdentity identifies an ALTER TABLE by table, subtype, and target —
// for example, alter.table.add_column:schema.table:column.
func alterTableIdentity(stmt statementData) identity {
	data := extractStatementData(stmt)
	relation := extractMap(data, keyRelation)
	if relation == nil {
		return ""
	}
	qualified := formatQualifiedName(extractSchemaAndName(relation))

	cmd := firstAlterCmd(data)
	subtype, found := extractInt(cmd, "subtype")
	if cmd == nil || !found {
		return identity(identityPrefixAlterTable + "unknown:" + identityPrefix(qualified))
	}

	subtypeName := mapAlterSubtype(alterSubtype(subtype))
	targetName := extractAlterTarget(cmd, alterSubtype(subtype))
	return identity(identityPrefixAlterTable + identityPrefix(subtypeName) +
		":" + identityPrefix(qualified) + ":" + identityPrefix(targetName))
}

// firstAlterCmd returns the first AlterTableCmd node, or nil if there isn't one.
func firstAlterCmd(data statementData) statementData {
	cmds := extractSlice(data, "cmds")
	if len(cmds) == 0 {
		return nil
	}
	cmd, ok := cmds[0].(map[string]any)
	if !ok {
		return nil
	}
	return extractMap(extractMap(statementData(cmd), keyNode), keyAlterTableCmd)
}

// mapAlterSubtype turns a pg_query AlterTableType into a snake_case name.
func mapAlterSubtype(t alterSubtype) alterSubtypeName {
	name := strings.TrimPrefix(pg_query.AlterTableType(t).String(), "AT_")
	return alterSubtypeName(toSnakeCase(stringValue(name)))
}

// extractAlterTarget returns the column or constraint name an ALTER TABLE command
// is aimed at.
func extractAlterTarget(cmd statementData, subtype alterSubtype) alterTargetName {
	if name := extractString(cmd, "name"); name != "" {
		return alterTargetName(name)
	}
	switch subtype {
	case alterAddColumn:
		return extractAddColumnName(cmd)
	case alterAddConstraint, alterDropConstraint:
		return extractConstraintName(cmd)
	default:
		return ""
	}
}

// extractAddColumnName pulls the column name out of an ADD COLUMN command.
func extractAddColumnName(cmd statementData) alterTargetName {
	colDef := extractMap(extractMap(extractMap(cmd, keyDef), keyNode), keyColumnDef)
	return alterTargetName(extractString(colDef, "colname"))
}

// extractConstraintName pulls the constraint name out of an ADD/DROP CONSTRAINT
// command.
func extractConstraintName(cmd statementData) alterTargetName {
	constraint := extractMap(extractMap(extractMap(cmd, keyDef), keyNode), keyConstraint)
	return alterTargetName(extractString(constraint, "conname"))
}
