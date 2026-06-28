package compare

// createIndexIdentity identifies a CREATE INDEX by table and index name — for
// example, create.index:schema.table:index. We fold in the table so two
// identically named indexes on different tables don't collide.
func createIndexIdentity(stmt statementData) identity {
	data := extractStatementData(stmt)
	if data == nil {
		return ""
	}
	idxname := extractString(data, "idxname")
	if idxname == "" {
		return ""
	}
	relation := extractMap(data, keyRelation)
	if relation == nil {
		return ""
	}
	qualified := formatQualifiedName(extractSchemaAndName(relation))
	return identity(identityPrefixCreateIndex + identityPrefix(qualified) + ":" + identityPrefix(idxname))
}
