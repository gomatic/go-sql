package sql

import (
	"reflect"
	"sort"
)

// Normalizing a decoded AST value into a canonical Go value. Every reflect.Kind
// dispatch here is over a NAMED subset of the kinds, so each switch stays total
// over its own domain rather than carrying twenty arms for kinds it can never
// meaningfully receive.

// compositeKind is the subset of reflect.Kind that CONTAINS other values and so
// is normalized structurally rather than read as a leaf.
type compositeKind reflect.Kind

const (
	compositeSlice  = compositeKind(reflect.Slice)
	compositeArray  = compositeKind(reflect.Array)
	compositeMap    = compositeKind(reflect.Map)
	compositeStruct = compositeKind(reflect.Struct)
)

func normalizeValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		return normalizeValue(v.Elem())
	}

	switch compositeKind(v.Kind()) {
	case compositeSlice, compositeArray:
		return normalizeSlice(v)
	case compositeMap:
		return normalizeMap(v)
	case compositeStruct:
		return normalizeStructValue(v)
	default:
		return scalarValue(v)
	}
}

// scalarKind is the subset of reflect.Kind this package widens to a canonical
// Go value: every integer width collapses to int64, every float to float64, so
// two ASTs that differ only in the width a field was decoded at compare equal.
// Anything else is handed through as-is.
type scalarKind reflect.Kind

const (
	scalarBool    = scalarKind(reflect.Bool)
	scalarInt     = scalarKind(reflect.Int)
	scalarInt8    = scalarKind(reflect.Int8)
	scalarInt16   = scalarKind(reflect.Int16)
	scalarInt32   = scalarKind(reflect.Int32)
	scalarInt64   = scalarKind(reflect.Int64)
	scalarUint    = scalarKind(reflect.Uint)
	scalarUint8   = scalarKind(reflect.Uint8)
	scalarUint16  = scalarKind(reflect.Uint16)
	scalarUint32  = scalarKind(reflect.Uint32)
	scalarUint64  = scalarKind(reflect.Uint64)
	scalarFloat32 = scalarKind(reflect.Float32)
	scalarFloat64 = scalarKind(reflect.Float64)
	scalarString  = scalarKind(reflect.String)
)

// scalarValue pulls a leaf (non-composite) reflect.Value out as a plain Go value.
func scalarValue(v reflect.Value) any {
	switch scalarKind(v.Kind()) {
	case scalarBool:
		return v.Bool()
	case scalarInt, scalarInt8, scalarInt16, scalarInt32, scalarInt64:
		return v.Int()
	case scalarUint, scalarUint8, scalarUint16, scalarUint32, scalarUint64:
		return v.Uint()
	case scalarFloat32, scalarFloat64:
		return v.Float()
	case scalarString:
		return v.String()
	default:
		return v.Interface()
	}
}

func normalizeSlice(v reflect.Value) []any {
	if v.Kind() == reflect.Slice && v.IsNil() {
		return nil
	}
	result := make([]any, 0, v.Len())
	for i := range v.Len() {
		if normalized := normalizeValue(v.Index(i)); !isZeroValue(normalized) {
			result = append(result, normalized)
		}
	}
	return result
}

func normalizeMap(v reflect.Value) map[string]any {
	if v.IsNil() {
		return nil
	}
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })

	result := make(map[string]any, len(keys))
	for _, key := range keys {
		if normalized := normalizeValue(v.MapIndex(key)); !isZeroValue(normalized) {
			result[key.String()] = normalized
		}
	}
	return result
}

func normalizePrimitiveValue(v reflect.Value) map[string]any {
	if !v.IsValid() || !v.CanInterface() {
		return nil
	}
	return map[string]any{"value": v.Interface()}
}

// emptiableKind is the subset of reflect.Kind whose emptiness CANNOT be decided
// by comparing against the zero value. An empty non-nil slice is not DeepEqual
// to a nil one, and a nil pointer inside a non-nil interface is not DeepEqual to
// an untyped nil, so each of these needs its own test — Len for the containers,
// IsNil for the references.
//
// Every other kind falls to the default, where DeepEqual against the zero value
// is exactly right: an int 0, a false bool and a "" string are all their type's
// zero and are dropped as absent. Naming the subset keeps this switch total over
// the kinds that need special handling rather than over all two dozen.
type emptiableKind reflect.Kind

const (
	emptiableSlice     = emptiableKind(reflect.Slice)
	emptiableMap       = emptiableKind(reflect.Map)
	emptiableArray     = emptiableKind(reflect.Array)
	emptiableString    = emptiableKind(reflect.String)
	emptiablePointer   = emptiableKind(reflect.Pointer)
	emptiableInterface = emptiableKind(reflect.Interface)
)

func isZeroValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch emptiableKind(rv.Kind()) {
	case emptiableSlice, emptiableMap, emptiableArray:
		return rv.Len() == 0
	case emptiableString:
		return rv.String() == ""
	case emptiablePointer, emptiableInterface:
		return rv.IsNil()
	default:
		return reflect.DeepEqual(v, reflect.Zero(rv.Type()).Interface())
	}
}

// goName is a PascalCase Go type or field name to convert to a snake_case JSON key.
type goName string

func toSnakeCase(s goName) string {
	var result []rune
	for i, r := range string(s) {
		if i > 0 && isUpper(nameRune(r)) && !isUpper(nameRune(rune(string(s)[i-1]))) {
			result = append(result, '_')
		}
		result = append(result, toLower(nameRune(r)))
	}
	return string(result)
}

// nameRune is one rune of a Go name being converted to snake_case.
type nameRune rune

func isUpper(r nameRune) bool { return rune(r) >= 'A' && rune(r) <= 'Z' }

func toLower(r nameRune) rune {
	if isUpper(nameRune(rune(r))) {
		return rune(r) + ('a' - 'A')
	}
	return rune(r)
}
