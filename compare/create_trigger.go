package compare

// createTriggerIdentity identifies a CREATE TRIGGER by name and table — for
// example, create.trigger:name:schema.table.
func createTriggerIdentity(stmt statementData) identity {
	data := extractStatementData(stmt)
	if data == nil {
		return ""
	}
	trigname := extractString(data, "trigname")
	if trigname == "" {
		return ""
	}
	tableName := formatQualifiedName(extractSchemaAndName(extractMap(data, keyRelation)))
	return identity(identityPrefixCreateTrigger + identityPrefix(trigname) + ":" + identityPrefix(tableName))
}
