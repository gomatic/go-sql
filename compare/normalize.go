package compare

import (
	"reflect"
	"slices"
	"strconv"
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

// keyLocation is the per-node source-position key.
const keyLocation fieldKey = "location"

// positionalFields are the source-position keys we strip before comparing: they
// move around with formatting, not with meaning.
var positionalFields = map[fieldKey]struct{}{
	"colno":         {},
	"lineno":        {},
	keyLocation:     {},
	"stmt_len":      {},
	"stmt_location": {},
}

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
		if _, skip := positionalFields[fieldKey(k)]; skip {
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

// computeDiffs returns the field-level differences between two statements, or
// nil if they're equal.
func computeDiffs(source, target statementData) statementDiffs {
	if statementsAreEqual(source, target) {
		return nil
	}
	var diffs statementDiffs
	computeMapDiffs("", normalizeStatement(source), normalizeStatement(target), &diffs)
	return diffs
}

// computeMapDiffs records the differences between two maps, under prefix.
func computeMapDiffs(prefix fieldPath, source, target map[string]any, diffs *statementDiffs) {
	for _, key := range sortedKeys(source, target) {
		processKey(prefix, key, source, target, diffs)
	}
}

// sortedKeys returns the union of both maps' keys, in stable order.
func sortedKeys(source, target map[string]any) []mapKey {
	seen := make(map[mapKey]struct{}, len(source)+len(target))
	for k := range source {
		seen[mapKey(k)] = struct{}{}
	}
	for k := range target {
		seen[mapKey(k)] = struct{}{}
	}
	keys := make([]mapKey, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// processKey records the difference for one key — whether that's a presence
// difference or a value difference.
func processKey(prefix fieldPath, key mapKey, source, target map[string]any, diffs *statementDiffs) {
	path := buildFieldPath(prefix, key)
	sourceVal, sourceExists := source[string(key)]
	targetVal, targetExists := target[string(key)]
	if !sourceExists {
		*diffs = append(*diffs, fieldDiff{Field: path, Source: nil, Target: targetVal})
		return
	}
	if !targetExists {
		*diffs = append(*diffs, fieldDiff{Field: path, Source: sourceVal, Target: nil})
		return
	}
	computeValueDiffs(path, sourceVal, targetVal, diffs)
}

// buildFieldPath joins a prefix and a key into a dotted path.
func buildFieldPath(prefix fieldPath, key mapKey) fieldPath {
	if prefix == "" {
		return fieldPath(key)
	}
	return fieldPath(string(prefix) + "." + string(key))
}

// computeValueDiffs records the difference between two values sitting at path.
func computeValueDiffs(path fieldPath, source, target any, diffs *statementDiffs) {
	if source == nil || target == nil {
		appendPresenceDiff(path, source, target, diffs)
		return
	}
	if diffComposite(path, source, target, diffs) {
		return
	}
	if !reflect.DeepEqual(source, target) {
		*diffs = append(*diffs, fieldDiff{Field: path, Source: source, Target: target})
	}
}

// appendPresenceDiff records a diff when only one side actually has the value.
func appendPresenceDiff(path fieldPath, source, target any, diffs *statementDiffs) {
	if source == nil && target == nil {
		return
	}
	*diffs = append(*diffs, fieldDiff{Field: path, Source: source, Target: target})
}

// diffComposite handles map and slice values, and tells you whether it took
// care of them.
func diffComposite(path fieldPath, source, target any, diffs *statementDiffs) bool {
	if sourceMap, ok := source.(map[string]any); ok {
		if targetMap, ok := target.(map[string]any); ok {
			computeMapDiffs(path, sourceMap, targetMap, diffs)
			return true
		}
	}
	if sourceSlice, ok := source.([]any); ok {
		if targetSlice, ok := target.([]any); ok {
			computeSliceDiffs(path, sourceSlice, targetSlice, diffs)
			return true
		}
	}
	return false
}

// computeSliceDiffs records the difference between two slices. If the slices are
// the same length and hold the same elements but in a different order, we report
// that as an order change — SQL column order actually matters.
func computeSliceDiffs(path fieldPath, source, target []any, diffs *statementDiffs) {
	if len(source) != len(target) {
		*diffs = append(*diffs, fieldDiff{Field: path, Source: source, Target: target})
		return
	}
	if sameOrder(source, target) {
		return
	}
	if slicesHaveSameElements(source, target) {
		*diffs = append(*diffs, reorderDiff(path))
		return
	}
	diffSliceElements(path, source, target, diffs)
}

// sameOrder tells you whether two equal-length slices match element by element.
func sameOrder(source, target []any) bool {
	for i := range source {
		if !reflect.DeepEqual(source[i], target[i]) {
			return false
		}
	}
	return true
}

// reorderDiff is the synthetic diff we report when the elements are just
// reordered.
func reorderDiff(path fieldPath) fieldDiff {
	return fieldDiff{
		Field:  fieldPath(string(path) + ".order"),
		Source: "elements reordered",
		Target: "see source and target for details",
	}
}

// diffSliceElements records element-by-element differences for slices that don't
// match.
func diffSliceElements(path fieldPath, source, target []any, diffs *statementDiffs) {
	for i := range source {
		itemPath := fieldPath(string(path) + "[" + strconv.Itoa(i) + "]")
		computeValueDiffs(itemPath, source[i], target[i], diffs)
	}
}

// slicesHaveSameElements tells you whether two slices hold the same multiset of
// elements, order aside.
func slicesHaveSameElements(source, target []any) elementsMatch {
	if len(source) != len(target) {
		return false
	}
	matched := make([]bool, len(target))
	for _, srcElem := range source {
		if !matchElement(srcElem, target, matched) {
			return false
		}
	}
	return true
}

// matchElement finds the first not-yet-matched target equal to elem, marks it,
// and reports whether it found one.
func matchElement(elem any, target []any, matched []bool) bool {
	for j, tgtElem := range target {
		if !matched[j] && reflect.DeepEqual(elem, tgtElem) {
			matched[j] = true
			return true
		}
	}
	return false
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

// extractStatementData digs the data payload out of a decoded statement,
// handling both the stmt-wrapped and top-level forms.
func extractStatementData(stmt statementData) statementData {
	if stmtObj := extractMap(stmt, keyStmt); stmtObj != nil {
		return extractMap(stmtObj, keyData)
	}
	return extractMap(stmt, keyData)
}

// toSnakeCase turns PascalCase or camelCase into snake_case.
func toSnakeCase(input stringValue) stringValue {
	s := string(input)
	result := make([]rune, 0, len(s)+4)
	for i, r := range s {
		if i > 0 && isUpper(rParam(r)) && !isUpper(rParam(rune(s[i-1]))) {
			result = append(result, '_')
		}
		if isUpper(rParam(r)) {
			r += 'a' - 'A'
		}
		result = append(result, r)
	}
	return stringValue(result)
}

// rParam names the r parameter of isUpper; rename it to the real domain concept.
type rParam rune

// isUpper tells you whether r is an ASCII uppercase letter.
func isUpper(r rParam) bool { return rune(r) >= 'A' && rune(r) <= 'Z' }
