package compare

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dropIdentityCase struct {
	name     testName
	sql      sqlStatement
	expected identity
}

func TestDropIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []dropIdentityCase{
		{name: "drop_table", sql: "DROP TABLE IF EXISTS my_schema.my_table", expected: "drop.table:my_schema.my_table"},
		{name: "drop_index", sql: "DROP INDEX IF EXISTS my_index", expected: "drop.index:my_index"},
		{name: "drop_view", sql: "DROP VIEW IF EXISTS my_schema.my_view", expected: "drop.view:my_schema.my_view"},
		{
			name:     "drop_function",
			sql:      "DROP FUNCTION IF EXISTS my_schema.my_func(int, text)",
			expected: "drop.function:my_schema.my_func(pg_catalog.int4,text)",
		},
		{name: "drop_schema", sql: "DROP SCHEMA IF EXISTS my_schema", expected: "drop.schema:my_schema"},
		{
			name:     "drop_trigger",
			sql:      "DROP TRIGGER IF EXISTS my_trigger ON my_table",
			expected: "drop.trigger:my_table.my_trigger",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, dropIdentity(parseTestSQL(t, must, tt.sql)))
		})
	}
}

func TestDropIdentity_NilCases(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Equal(identity(""), dropIdentity(nil))
	// A handled type with no remove_type gives back no identity.
	want.Equal(identity(""), dropIdentity(statementData{"stmt": map[string]any{"data": map[string]any{}}}))
}

func TestDropDiff(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	source := parseTestSQL(t, must, "DROP TABLE IF EXISTS my_table")
	want.Empty(genericDiff(source, source))
}

type objectTypeCase struct {
	name     testName
	expected dropObjectType
	objType  objectTypeInt
}

func TestMapObjectType(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []objectTypeCase{
		{name: "function", objType: objectTypeInt(pg_query.ObjectType_OBJECT_FUNCTION), expected: "function"},
		{name: "index", objType: objectTypeInt(pg_query.ObjectType_OBJECT_INDEX), expected: "index"},
		{name: "schema", objType: objectTypeInt(pg_query.ObjectType_OBJECT_SCHEMA), expected: "schema"},
		{name: "table", objType: objectTypeInt(pg_query.ObjectType_OBJECT_TABLE), expected: "table"},
		{name: "trigger", objType: objectTypeInt(pg_query.ObjectType_OBJECT_TRIGGER), expected: "trigger"},
		{name: "view", objType: objectTypeInt(pg_query.ObjectType_OBJECT_VIEW), expected: "view"},
		{
			name:     "unhandled_falls_back",
			objType:  objectTypeInt(pg_query.ObjectType_OBJECT_SEQUENCE),
			expected: "unknown_object__sequence",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, mapObjectType(tt.objType))
		})
	}
}

func TestExtractNodeStringParts_SkipsNonStrings(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	// A non-map element and a node with no String_ both add nothing.
	items := []any{
		"not a map",
		map[string]any{"node": map[string]any{"string_": map[string]any{"sval": "kept"}}},
		map[string]any{"node": map[string]any{}},
	}
	want.Equal([]string{"kept"}, extractNodeStringParts(items))
}

func TestExtractDropArgTypeName_NonMap(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Equal(typeName(""), extractDropArgTypeName(42))
}

func TestExtractDropObjectName_Fallback(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	want.Equal(qualifiedName(""), extractDropObjectName(statementData{}), "no objects yields no name")

	// A node whose child shape we don't recognize gives back no name.
	node := statementData{"objects": []any{map[string]any{"node": map[string]any{"unknown": map[string]any{}}}}}
	want.Equal(qualifiedName(""), extractDropObjectName(node))
}

func TestExtractFunctionDropName_EmptyObjname(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Equal(qualifiedName(""), extractFunctionDropName(statementData{}))
}

func TestFirstObjectNode(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	want.Nil(firstObjectNode(statementData{}, keyObjects), "no objects yields nil")
	want.Nil(
		firstObjectNode(statementData{"objects": []any{"not a map"}}, keyObjects),
		"non-map first object yields nil",
	)
}
