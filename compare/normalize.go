package compare

import (
	"reflect"
)

// Named types for the normalization and diff engine.
type (
	areEqual      bool   // areEqual tells you whether two statements are equal.
	elementsMatch bool   // elementsMatch tells you whether two slices share the same multiset of elements.
	intFound      bool   // intFound tells you whether an int was actually found.
	intValue      int    // intValue is an integer pulled out of a statement.
	mapKey        string // mapKey is a key encountered while diffing two maps.
	objectName    string // objectName is a database object name.
	qualifiedName string // qualifiedName is a schema-qualified database object name.
	schemaName    string // schemaName is a database schema name.
	stringValue   string // stringValue is a string pulled out of a statement.
)

// normalizeStatement returns a copy of a statement with positional metadata
// stripped out everywhere.
func normalizeStatement(stmt statementData) statementData {
	if stmt == nil {
		return nil
	}
	return normalizeMap(stmt)
}

// normalizeMap copies a map recursively, dropping positional fields as it goes.
func normalizeMap(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		if _, skip := positionalFields[positionalField(k)]; skip {
			continue
		}
		result[k] = normalizeValue(v)
	}
	return result
}

// normalizeValue normalizes a decoded JSON value, recursing into maps and slices.
func normalizeValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return normalizeMap(val)
	case []any:
		return normalizeSlice(val)
	default:
		return v
	}
}

// normalizeSlice normalizes each element of a slice, recursing as needed.
func normalizeSlice(s []any) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = normalizeValue(v)
	}
	return result
}

// statementsAreEqual tells you whether two statements are equal once they're
// normalized.
func statementsAreEqual(source, target statementData) areEqual {
	if source == nil || target == nil {
		return areEqual(source == nil && target == nil)
	}
	return areEqual(reflect.DeepEqual(normalizeStatement(source), normalizeStatement(target)))
}
