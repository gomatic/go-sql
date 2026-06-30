package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type funcIdentityCase struct {
	name     testName
	sql      sqlStatement
	expected identity
}

func TestCreateFunctionIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []funcIdentityCase{
		{
			name:     "function_overload",
			sql:      "CREATE FUNCTION my_func(a text) RETURNS int AS $$ SELECT 1; $$ LANGUAGE sql",
			expected: "create.function:my_func(text)",
		},
		{
			name:     "function_with_args",
			sql:      "CREATE FUNCTION my_func(a int, b text) RETURNS int AS $$ SELECT 1; $$ LANGUAGE sql",
			expected: "create.function:my_func(pg_catalog.int4,text)",
		},
		{
			name:     "function_with_schema",
			sql:      "CREATE FUNCTION my_schema.my_func() RETURNS int AS $$ SELECT 1; $$ LANGUAGE sql",
			expected: "create.function:my_schema.my_func()",
		},
		{
			name:     "simple_function",
			sql:      "CREATE FUNCTION my_func() RETURNS int AS $$ SELECT 1; $$ LANGUAGE sql",
			expected: "create.function:my_func()",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, createFunctionIdentity(parseTestSQL(t, must, tt.sql)))
		})
	}
}

func TestCreateFunctionIdentity_NilCases(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Equal(identity(""), createFunctionIdentity(nil))
	want.Equal(identity(""), createFunctionIdentity(statementData{"stmt": map[string]any{"data": map[string]any{}}}))
}

func TestCreateFunctionDiff_SameFunctionIgnoresBody(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	source := parseTestSQL(t, must, "CREATE FUNCTION foo() RETURNS int AS $$ SELECT 1; $$ LANGUAGE sql")
	target := parseTestSQL(t, must, "CREATE FUNCTION foo() RETURNS int AS $$ SELECT 2; $$ LANGUAGE sql")

	want.Empty(createFunctionDiff(source, source))
	// We ignore the body, so a difference that's only in the body gives no diffs.
	want.Empty(createFunctionDiff(source, target))
}

func TestExtractFunctionName(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	schema, name := extractFunctionName(nil)
	want.Equal(schemaName(""), schema)
	want.Equal(objectName(""), name)
}

func TestExtractParamTypeName_NonMap(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Equal(typeName(""), extractParamTypeName(42))
}

type deepCopyCase struct {
	original statementData
	name     testName
}

func TestDeepCopyMap(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []deepCopyCase{
		{
			name: "nested_map_copy",
			original: statementData{
				"key1":   "value1",
				"list":   []any{"a", "b"},
				"nested": map[string]any{"inner": "data"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			copied := deepCopyMap(tt.original)

			copied["key1"] = "modified"
			nestedCopy, ok := copied["nested"].(map[string]any)
			want.True(ok, "nested should remain map[string]any after deep copy")
			nestedCopy["inner"] = "modified"
			listCopy, ok := copied["list"].([]any)
			want.True(ok, "list should be []any after deep copy")
			listCopy[0] = "modified"

			want.Equal("value1", tt.original["key1"])
			want.Equal("data", tt.original["nested"].(map[string]any)["inner"])
			want.Equal("a", tt.original["list"].([]any)[0])
		})
	}
}

func TestDeepCopyMap_Nil(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Nil(deepCopyMap(nil))
}

func TestRemoveFunctionBody_StripsOptions(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	original := parseTestSQL(t, must, "CREATE FUNCTION foo() RETURNS int AS $$ SELECT 1; $$ LANGUAGE sql")
	result := removeFunctionBody(original)

	data := extractMap(extractMap(result, keyStmt), keyData)
	must.NotNil(data)
	_, hasOptions := data["options"]
	want.False(hasOptions, "options should be stripped")

	// The original is untouched.
	origData := extractMap(extractMap(original, keyStmt), keyData)
	_, origHasOptions := origData["options"]
	want.True(origHasOptions, "original should retain options")
}
