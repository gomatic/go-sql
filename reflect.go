package sql

import (
	"reflect"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// treeReflect hands back the reflect.Value we root the AST walk at.
func treeReflect(tree *pg_query.ParseResult) reflect.Value {
	return reflect.ValueOf(tree)
}

// traverseAndSort walks the AST, sorting the column lists of every INSERT and
// simple SELECT it runs into. It recurses through pointers, structs, slices, and
// interfaces.
// traversableKind is the subset of reflect.Kind that can CONTAIN a value worth
// sorting. reflect.Kind has two dozen members and all but these four are leaves
// — an int holds no sub-value to reorder — so a switch listing every one of
// them would carry twenty arms that do nothing. Naming the domain keeps the
// switch total over what it actually dispatches on.
type traversableKind reflect.Kind

const (
	traversablePointer   = traversableKind(reflect.Pointer)
	traversableStruct    = traversableKind(reflect.Struct)
	traversableSlice     = traversableKind(reflect.Slice)
	traversableInterface = traversableKind(reflect.Interface)
)

func traverseAndSort(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	switch traversableKind(v.Kind()) {
	case traversablePointer:
		traversePointer(v)
	case traversableStruct:
		traverseStruct(v)
	case traversableSlice:
		traverseSlice(v)
	case traversableInterface:
		traverseAndSort(v.Elem())
	default:
	}
}

func traversePointer(v reflect.Value) {
	if v.IsNil() {
		return
	}
	switch node := v.Interface().(type) {
	case *pg_query.InsertStmt:
		sortInsertStmt(node)
	case *pg_query.SelectStmt:
		sortSelectStmt(node)
	}
	traverseAndSort(v.Elem())
}

func traverseStruct(v reflect.Value) {
	for i := range v.NumField() {
		if field := v.Field(i); field.CanInterface() {
			traverseAndSort(field)
		}
	}
}

func traverseSlice(v reflect.Value) {
	for i := range v.Len() {
		traverseAndSort(v.Index(i))
	}
}
