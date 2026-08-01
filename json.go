package sql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	errs "github.com/gomatic/go-error"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// ErrMarshal means we couldn't marshal a normalized statement to JSON.
const ErrMarshal errs.Const = "marshal statement"

// marshaler is [json.Marshal]'s signature. We inject it so a test can force a
// marshal failure without reaching for a package-level global.
type marshaler func(any) ([]byte, error)

// ToJSON converts a parse result to normalized JSON, one statement per entry.
// Within each statement the field order is canonical (keys sorted, zero values
// dropped) so two equivalent statements come out as identical JSON. A nil result
// gives you back a nil slice.
func ToJSON(result *pg_query.ParseResult) ([]json.RawMessage, error) {
	return toJSON(result, json.Marshal)
}

func toJSON(result *pg_query.ParseResult, marshal marshaler) ([]json.RawMessage, error) {
	if result == nil {
		return nil, nil
	}

	statements := make([]json.RawMessage, 0, len(result.Stmts))
	for _, stmt := range result.Stmts {
		jsonBytes, err := marshal(normalizeStatement(stmt))
		if err != nil {
			return nil, ErrMarshal.With(err)
		}
		statements = append(statements, json.RawMessage(jsonBytes))
	}

	return statements, nil
}

func normalizeStatement(stmt *pg_query.RawStmt) map[string]any {
	if stmt == nil || stmt.Stmt == nil {
		return map[string]any{}
	}
	return map[string]any{"stmt": normalizeNode(stmt.Stmt)}
}

// normalizeNode renders a statement node as {type, data}. If we recognize the
// statement kind, we unwrap it to its inner message; anything else falls back to
// a reflective rendering keyed by its derived snake_case type name.
func normalizeNode(node *pg_query.Node) map[string]any {
	if node == nil {
		return nil
	}
	if typeName, inner, ok := unwrapNode(node); ok {
		return map[string]any{"type": typeName, "data": normalizeStructValue(reflect.ValueOf(inner))}
	}
	return normalizeDefault(node.Node)
}

// recognizedStmts is the set of statement type names we unwrap straight to their
// inner message rather than running through the reflective fallback. It's
// read-only after init, so it's safe for concurrent use.
var recognizedStmts = map[string]struct{}{
	"create_schema_stmt":   {},
	"create_stmt":          {},
	"view_stmt":            {},
	"create_function_stmt": {},
	"create_trig_stmt":     {},
	"create_cast_stmt":     {},
	"alter_table_stmt":     {},
	"select_stmt":          {},
	"insert_stmt":          {},
	"update_stmt":          {},
	"delete_stmt":          {},
	"comment_stmt":         {},
	"grant_stmt":           {},
	"index_stmt":           {},
	"drop_stmt":            {},
	"do_stmt":              {},
}

// unwrapNode takes a recognized statement wrapper and hands back its snake_case
// name plus the inner message, dug out of the wrapper's single oneof field. The
// boolean is false for kinds we don't recognize, which take the reflective
// fallback instead.
func unwrapNode(node *pg_query.Node) (string, any, bool) {
	name := nodeTypeName(node.Node)
	if _, ok := recognizedStmts[name]; !ok {
		return "", nil, false
	}
	return name, reflect.ValueOf(node.Node).Elem().Field(0).Interface(), true
}

// normalizeDefault renders a node we don't recognize reflectively, keyed by the
// snake_case form of its concrete type name.
func normalizeDefault(n any) map[string]any {
	result := map[string]any{"type": nodeTypeName(n)}
	if rv := reflect.ValueOf(n); rv.Kind() == reflect.Pointer && !rv.IsNil() {
		result["data"] = normalizeStructValue(rv.Elem())
	}
	return result
}

// nodeTypeName builds a snake_case type name from a node's concrete Go type,
// dropping the package qualifier and pg_query's "Node_" wrapper prefix.
func nodeTypeName(n any) string {
	typeName := fmt.Sprintf("%T", n)
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		typeName = typeName[idx+1:]
	}
	return toSnakeCase(goName(strings.TrimPrefix(typeName, "Node_")))
}

type fieldInfo struct {
	value any
	name  string
}

func normalizeStructValue(v reflect.Value) map[string]any {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return normalizePrimitiveValue(v)
	}

	fields := structFields(v)
	sort.Slice(fields, func(i, j int) bool { return fields[i].name < fields[j].name })

	result := make(map[string]any, len(fields))
	for _, field := range fields {
		result[field.name] = field.value
	}
	return result
}

// structFields gathers a struct's non-zero, exported fields as snake_cased
// name/value pairs.
func structFields(v reflect.Value) []fieldInfo {
	typ := v.Type()
	fields := make([]fieldInfo, 0, typ.NumField())
	for i := range typ.NumField() {
		fieldValue := v.Field(i)
		if !fieldValue.CanInterface() {
			continue
		}
		if normalized := normalizeValue(fieldValue); !isZeroValue(normalized) {
			fields = append(fields, fieldInfo{name: toSnakeCase(goName(typ.Field(i).Name)), value: normalized})
		}
	}
	return fields
}
