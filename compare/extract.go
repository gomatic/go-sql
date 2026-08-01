package compare

// Pulling typed values out of a decoded statement map. Every reader here
// answers "absent" rather than a zero value where the distinction matters, so a
// missing key and a key holding zero stay distinguishable to the comparison.

// extractStatementData digs the data payload out of a decoded statement,
// handling both the stmt-wrapped and top-level forms.
func extractStatementData(stmt statementData) statementData {
	if stmtObj := extractMap(stmt, keyStmt); stmtObj != nil {
		return extractMap(stmtObj, keyData)
	}
	return extractMap(stmt, keyData)
}

// extractString reads a string field, handing back "" if it's missing or isn't
// a string.
func extractString(data statementData, key fieldKey) stringValue {
	if val, ok := data[string(key)].(string); ok {
		return stringValue(val)
	}
	return ""
}

// extractInt reads an integer field, taking JSON's float64 and int64 forms in
// stride.
func extractInt(data statementData, key fieldKey) (intValue, intFound) {
	switch val := data[string(key)].(type) {
	case int:
		return intValue(val), true
	case int64:
		return intValue(val), true
	case float64:
		return intValue(val), true
	default:
		return 0, false
	}
}

// extractMap reads a nested map field, handing back nil if it's missing or isn't
// a map.
func extractMap(data statementData, key fieldKey) statementData {
	if val, ok := data[string(key)].(map[string]any); ok {
		return val
	}
	return nil
}

// extractSlice reads a slice field, handing back nil if it's missing or isn't a
// slice.
func extractSlice(data statementData, key fieldKey) []any {
	if val, ok := data[string(key)].([]any); ok {
		return val
	}
	return nil
}

// toSnakeCase turns PascalCase or camelCase into snake_case.
func toSnakeCase(input stringValue) stringValue {
	s := string(input)
	result := make([]rune, 0, len(s)+4)
	for i, r := range s {
		if i > 0 && isUpper(nameRune(r)) && !isUpper(nameRune(rune(s[i-1]))) {
			result = append(result, '_')
		}
		if isUpper(nameRune(r)) {
			r += 'a' - 'A'
		}
		result = append(result, r)
	}
	return stringValue(result)
}

// extractSchemaAndName pulls the schema and object name out of a relation node.
func extractSchemaAndName(data statementData) (schemaName, objectName) {
	return schemaName(extractString(data, keySchemaname)), objectName(extractString(data, keyRelname))
}

// formatQualifiedName renders schema.name, or just name when there's no schema.
func formatQualifiedName(schema schemaName, name objectName) qualifiedName {
	if schema == "" {
		return qualifiedName(name)
	}
	return qualifiedName(string(schema) + "." + string(name))
}
