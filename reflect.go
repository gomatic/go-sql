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
func traverseAndSort(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		traversePointer(v)
	case reflect.Struct:
		traverseStruct(v)
	case reflect.Slice:
		traverseSlice(v)
	case reflect.Interface:
		traverseAndSort(v.Elem())
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
