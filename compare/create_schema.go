package compare

// createSchemaIdentity identifies a CREATE SCHEMA by name — for example,
// create.schema:app.
func createSchemaIdentity(stmt statementData) identity {
	data := extractStatementData(stmt)
	if data == nil {
		return ""
	}
	schema := extractString(data, keySchemaname)
	if schema == "" {
		return ""
	}
	return identity(identityPrefixCreateSchema + identityPrefix(schema))
}
