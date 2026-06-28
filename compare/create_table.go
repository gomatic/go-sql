package compare

// createTableIdentity identifies a CREATE TABLE by its qualified name — for
// example, create.table:schema.table.
func createTableIdentity(stmt statementData) identity {
	data := extractStatementData(stmt)
	if data == nil {
		return ""
	}
	relation := extractMap(data, keyRelation)
	if relation == nil {
		return ""
	}
	return identity(identityPrefixCreateTable + identityPrefix(formatQualifiedName(extractSchemaAndName(relation))))
}
