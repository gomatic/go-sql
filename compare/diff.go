package compare

import (
	"reflect"
	"slices"
	"strconv"
)

// Computing the differences between two normalized statements. Normalization
// decides WHAT is compared; this decides how two answers differ and how that
// difference is described to a caller.

// computeDiffs returns the field-level differences between two statements, or
// nil if they're equal.
func computeDiffs(source, target statementData) statementDiffs {
	if statementsAreEqual(source, target) {
		return nil
	}
	return computeMapDiffs("", normalizeStatement(source), normalizeStatement(target), nil)
}

// computeMapDiffs appends the differences between two maps, under prefix, and
// returns the grown diff list.
func computeMapDiffs(prefix fieldPath, source, target map[string]any, diffs statementDiffs) statementDiffs {
	for _, key := range sortedKeys(source, target) {
		diffs = processKey(prefix, key, source, target, diffs)
	}
	return diffs
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

// processKey appends the difference for one key — whether that's a presence
// difference or a value difference — and returns the grown diff list.
func processKey(prefix fieldPath, key mapKey, source, target map[string]any, diffs statementDiffs) statementDiffs {
	path := buildFieldPath(prefix, key)
	sourceVal, sourceExists := source[string(key)]
	targetVal, targetExists := target[string(key)]
	if !sourceExists {
		return append(diffs, fieldDiff{Field: path, Source: nil, Target: targetVal})
	}
	if !targetExists {
		return append(diffs, fieldDiff{Field: path, Source: sourceVal, Target: nil})
	}
	return computeValueDiffs(path, sourceVal, targetVal, diffs)
}

// buildFieldPath joins a prefix and a key into a dotted path.
func buildFieldPath(prefix fieldPath, key mapKey) fieldPath {
	if prefix == "" {
		return fieldPath(key)
	}
	return fieldPath(string(prefix) + "." + string(key))
}

// computeValueDiffs appends the difference between two values sitting at path
// and returns the grown diff list.
func computeValueDiffs(path fieldPath, source, target any, diffs statementDiffs) statementDiffs {
	if source == nil || target == nil {
		return appendPresenceDiff(path, source, target, diffs)
	}
	if grown, handled := diffComposite(path, source, target, diffs); handled {
		return grown
	}
	if !reflect.DeepEqual(source, target) {
		return append(diffs, fieldDiff{Field: path, Source: source, Target: target})
	}
	return diffs
}

// appendPresenceDiff appends a diff when only one side actually has the value.
func appendPresenceDiff(path fieldPath, source, target any, diffs statementDiffs) statementDiffs {
	if source == nil && target == nil {
		return diffs
	}
	return append(diffs, fieldDiff{Field: path, Source: source, Target: target})
}

// diffComposite handles map and slice values: the grown diff list plus whether
// it took care of them.
func diffComposite(path fieldPath, source, target any, diffs statementDiffs) (statementDiffs, bool) {
	if sourceMap, ok := source.(map[string]any); ok {
		if targetMap, ok := target.(map[string]any); ok {
			return computeMapDiffs(path, sourceMap, targetMap, diffs), true
		}
	}
	if sourceSlice, ok := source.([]any); ok {
		if targetSlice, ok := target.([]any); ok {
			return computeSliceDiffs(path, sourceSlice, targetSlice, diffs), true
		}
	}
	return diffs, false
}

// computeSliceDiffs appends the difference between two slices and returns the
// grown diff list. If the slices are the same length and hold the same elements
// but in a different order, we report that as an order change — SQL column
// order actually matters.
func computeSliceDiffs(path fieldPath, source, target []any, diffs statementDiffs) statementDiffs {
	if len(source) != len(target) {
		return append(diffs, fieldDiff{Field: path, Source: source, Target: target})
	}
	if sameOrder(source, target) {
		return diffs
	}
	if slicesHaveSameElements(source, target) {
		return append(diffs, reorderDiff(path))
	}
	return diffSliceElements(path, source, target, diffs)
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

// diffSliceElements appends element-by-element differences for slices that don't
// match and returns the grown diff list.
func diffSliceElements(path fieldPath, source, target []any, diffs statementDiffs) statementDiffs {
	for i := range source {
		itemPath := fieldPath(string(path) + "[" + strconv.Itoa(i) + "]")
		diffs = computeValueDiffs(itemPath, source[i], target[i], diffs)
	}
	return diffs
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

// nameRune is one rune of a PascalCase or camelCase name being case-converted.
type nameRune rune

// isUpper tells you whether r is an ASCII uppercase letter.
func isUpper(r nameRune) bool { return rune(r) >= 'A' && rune(r) <= 'Z' }
