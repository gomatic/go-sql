package compare

// createViewIdentity identifies a CREATE VIEW by its qualified name — for
// example, create.view:schema.view.
func createViewIdentity(stmt statementData) identity {
	data := extractStatementData(stmt)
	if data == nil {
		return ""
	}
	view := extractMap(data, "view")
	if view == nil {
		return ""
	}
	return identity(identityPrefixCreateView + identityPrefix(formatQualifiedName(extractSchemaAndName(view))))
}
