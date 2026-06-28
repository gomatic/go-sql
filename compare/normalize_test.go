package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type normalizeStatementCase struct {
	input    statementData
	expected statementData
	name     testName
}

func TestNormalizeStatement(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []normalizeStatementCase{
		{name: "nil_input", input: nil, expected: nil},
		{name: "empty_map", input: statementData{}, expected: statementData{}},
		{name: "removes_location", input: statementData{"location": 42, "name": "test"}, expected: statementData{"name": "test"}},
		{name: "removes_lineno", input: statementData{"lineno": 1, "name": "test"}, expected: statementData{"name": "test"}},
		{name: "removes_colno", input: statementData{"colno": 5, "name": "test"}, expected: statementData{"name": "test"}},
		{name: "removes_stmt_len", input: statementData{"stmt_len": 100, "name": "test"}, expected: statementData{"name": "test"}},
		{name: "removes_stmt_location", input: statementData{"stmt_location": 0, "name": "test"}, expected: statementData{"name": "test"}},
		{name: "nested_map", input: statementData{"outer": map[string]any{"location": 1, "inner": "value"}}, expected: statementData{"outer": map[string]any{"inner": "value"}}},
		{name: "nested_slice", input: statementData{"items": []any{map[string]any{"location": 1, "name": "a"}}}, expected: statementData{"items": []any{map[string]any{"name": "a"}}}},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, normalizeStatement(tt.input))
		})
	}
}

type normalizeValueCase struct {
	input    any
	expected any
	name     testName
}

func TestNormalizeValue(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []normalizeValueCase{
		{name: "nil_value", input: nil, expected: nil},
		{name: "string_value", input: "test", expected: "test"},
		{name: "int_value", input: 42, expected: 42},
		{name: "bool_value", input: true, expected: true},
		{name: "float_value", input: 3.14, expected: 3.14},
		{name: "map_value", input: map[string]any{"location": 1, "name": "test"}, expected: map[string]any{"name": "test"}},
		{name: "slice_value", input: []any{"a", "b"}, expected: []any{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, normalizeValue(tt.input))
		})
	}
}

type statementsEqualCase struct {
	source   statementData
	target   statementData
	name     testName
	expected areEqual
}

func TestStatementsAreEqual(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []statementsEqualCase{
		{name: "both_nil", source: nil, target: nil, expected: true},
		{name: "source_nil", source: nil, target: statementData{}, expected: false},
		{name: "target_nil", source: statementData{}, target: nil, expected: false},
		{name: "both_empty", source: statementData{}, target: statementData{}, expected: true},
		{name: "equal_content", source: statementData{"name": "test"}, target: statementData{"name": "test"}, expected: true},
		{name: "different_content", source: statementData{"name": "a"}, target: statementData{"name": "b"}, expected: false},
		{name: "location_ignored", source: statementData{"location": 1, "name": "test"}, target: statementData{"location": 2, "name": "test"}, expected: true},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, statementsAreEqual(tt.source, tt.target))
		})
	}
}

type computeDiffsCase struct {
	source      statementData
	target      statementData
	name        testName
	expectDiffs expectBool
}

func TestComputeDiffs(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []computeDiffsCase{
		{name: "equal_statements", source: statementData{"name": "test"}, target: statementData{"name": "test"}, expectDiffs: false},
		{name: "different_value", source: statementData{"name": "a"}, target: statementData{"name": "b"}, expectDiffs: true},
		{name: "missing_key_in_target", source: statementData{"name": "test"}, target: statementData{}, expectDiffs: true},
		{name: "extra_key_in_target", source: statementData{}, target: statementData{"name": "test"}, expectDiffs: true},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			diffs := computeDiffs(tt.source, tt.target)
			if tt.expectDiffs {
				want.NotEmpty(diffs)
			} else {
				want.Empty(diffs)
			}
		})
	}
}

type extractStringCase struct {
	name     testName
	data     statementData
	key      fieldKey
	expected stringValue
}

func TestExtractString(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []extractStringCase{
		{name: "nil_data", data: nil, key: "name", expected: ""},
		{name: "empty_data", data: statementData{}, key: "name", expected: ""},
		{name: "key_not_found", data: statementData{"other": "value"}, key: "name", expected: ""},
		{name: "key_not_string", data: statementData{"name": 42}, key: "name", expected: ""},
		{name: "key_found", data: statementData{"name": "test"}, key: "name", expected: "test"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, extractString(tt.data, tt.key))
		})
	}
}

type extractIntCase struct {
	name          testName
	data          statementData
	key           fieldKey
	expectedValue intValue
	expectedFound intFound
}

func TestExtractInt(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []extractIntCase{
		{name: "nil_data", data: nil, key: "count", expectedValue: 0, expectedFound: false},
		{name: "empty_data", data: statementData{}, key: "count", expectedValue: 0, expectedFound: false},
		{name: "key_not_found", data: statementData{"other": 42}, key: "count", expectedValue: 0, expectedFound: false},
		{name: "key_not_int", data: statementData{"count": "42"}, key: "count", expectedValue: 0, expectedFound: false},
		{name: "key_is_int", data: statementData{"count": 42}, key: "count", expectedValue: 42, expectedFound: true},
		{name: "key_is_int64", data: statementData{"count": int64(42)}, key: "count", expectedValue: 42, expectedFound: true},
		{name: "key_is_float64", data: statementData{"count": float64(42)}, key: "count", expectedValue: 42, expectedFound: true},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			value, found := extractInt(tt.data, tt.key)
			want.Equal(tt.expectedValue, value)
			want.Equal(tt.expectedFound, found)
		})
	}
}

type extractMapCase struct {
	data     statementData
	expected statementData
	name     testName
	key      fieldKey
}

func TestExtractMap(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []extractMapCase{
		{name: "nil_data", data: nil, key: "nested", expected: nil},
		{name: "empty_data", data: statementData{}, key: "nested", expected: nil},
		{name: "key_not_found", data: statementData{"other": map[string]any{}}, key: "nested", expected: nil},
		{name: "key_not_map", data: statementData{"nested": "not a map"}, key: "nested", expected: nil},
		{name: "key_found", data: statementData{"nested": map[string]any{"inner": "value"}}, key: "nested", expected: statementData{"inner": "value"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, extractMap(tt.data, tt.key))
		})
	}
}

type extractSliceCase struct {
	name     testName
	data     statementData
	key      fieldKey
	expected []any
}

func TestExtractSlice(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []extractSliceCase{
		{name: "nil_data", data: nil, key: "items", expected: nil},
		{name: "empty_data", data: statementData{}, key: "items", expected: nil},
		{name: "key_not_found", data: statementData{"other": []any{}}, key: "items", expected: nil},
		{name: "key_not_slice", data: statementData{"items": "not a slice"}, key: "items", expected: nil},
		{name: "key_found", data: statementData{"items": []any{"a", "b"}}, key: "items", expected: []any{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, extractSlice(tt.data, tt.key))
		})
	}
}

type extractStatementDataCase struct {
	stmt     statementData
	expected statementData
	name     testName
}

func TestExtractStatementData(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []extractStatementDataCase{
		{name: "nil_stmt", stmt: nil, expected: nil},
		{name: "stmt_wrapped", stmt: statementData{"stmt": map[string]any{"data": map[string]any{"a": "b"}}}, expected: statementData{"a": "b"}},
		{name: "top_level_data", stmt: statementData{"data": map[string]any{"a": "b"}}, expected: statementData{"a": "b"}},
		{name: "missing_data", stmt: statementData{"stmt": map[string]any{}}, expected: nil},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, extractStatementData(tt.stmt))
		})
	}
}

type toSnakeCaseCase struct {
	name     testName
	input    stringValue
	expected stringValue
}

func TestToSnakeCase(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []toSnakeCaseCase{
		{name: "empty_string", input: "", expected: ""},
		{name: "lowercase", input: "lowercase", expected: "lowercase"},
		{name: "pascal_case", input: "PascalCase", expected: "pascal_case"},
		{name: "camel_case", input: "camelCase", expected: "camel_case"},
		{name: "all_caps", input: "ABC", expected: "abc"},
		{name: "mixed", input: "XMLHttpRequest", expected: "xmlhttp_request"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, toSnakeCase(tt.input))
		})
	}
}

type slicesHaveSameElementsCase struct {
	name     testName
	source   []any
	target   []any
	expected elementsMatch
}

func TestSlicesHaveSameElements(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []slicesHaveSameElementsCase{
		{name: "empty_slices", source: []any{}, target: []any{}, expected: true},
		{name: "same_order", source: []any{"a", "b", "c"}, target: []any{"a", "b", "c"}, expected: true},
		{name: "different_order", source: []any{"a", "b", "c"}, target: []any{"c", "b", "a"}, expected: true},
		{name: "different_length", source: []any{"a", "b"}, target: []any{"a", "b", "c"}, expected: false},
		{name: "different_elements", source: []any{"a", "b"}, target: []any{"a", "c"}, expected: false},
		{name: "duplicate_elements", source: []any{"a", "a", "b"}, target: []any{"a", "b", "a"}, expected: true},
		{name: "maps_same_order", source: []any{map[string]any{"name": "a"}, map[string]any{"name": "b"}}, target: []any{map[string]any{"name": "a"}, map[string]any{"name": "b"}}, expected: true},
		{name: "maps_different_order", source: []any{map[string]any{"name": "a"}, map[string]any{"name": "b"}}, target: []any{map[string]any{"name": "b"}, map[string]any{"name": "a"}}, expected: true},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, slicesHaveSameElements(tt.source, tt.target))
		})
	}
}

type schemaAndNameCase struct {
	name           testName
	data           statementData
	expectedSchema schemaName
	expectedName   objectName
}

func TestExtractSchemaAndName(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []schemaAndNameCase{
		{name: "empty_data", data: statementData{}, expectedSchema: "", expectedName: ""},
		{name: "only_relname", data: statementData{"relname": "my_table"}, expectedSchema: "", expectedName: "my_table"},
		{name: "only_schemaname", data: statementData{"schemaname": "my_schema"}, expectedSchema: "my_schema", expectedName: ""},
		{name: "both", data: statementData{"schemaname": "my_schema", "relname": "my_table"}, expectedSchema: "my_schema", expectedName: "my_table"},
		{name: "wrong_types", data: statementData{"schemaname": 123, "relname": 456}, expectedSchema: "", expectedName: ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			schema, name := extractSchemaAndName(tt.data)
			want.Equal(tt.expectedSchema, schema)
			want.Equal(tt.expectedName, name)
		})
	}
}

type qualifiedNameCase struct {
	name     testName
	schema   schemaName
	objName  objectName
	expected qualifiedName
}

func TestFormatQualifiedName(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []qualifiedNameCase{
		{name: "no_schema", schema: "", objName: "my_table", expected: "my_table"},
		{name: "with_schema", schema: "my_schema", objName: "my_table", expected: "my_schema.my_table"},
		{name: "empty_name", schema: "my_schema", objName: "", expected: "my_schema."},
		{name: "both_empty", schema: "", objName: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, formatQualifiedName(tt.schema, tt.objName))
		})
	}
}

type computeValueDiffsCase struct {
	source      any
	target      any
	name        testName
	expectDiffs expectBool
}

func TestComputeValueDiffs(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []computeValueDiffsCase{
		{name: "both_nil", source: nil, target: nil, expectDiffs: false},
		{name: "source_nil", source: nil, target: "value", expectDiffs: true},
		{name: "target_nil", source: "value", target: nil, expectDiffs: true},
		{name: "equal_strings", source: "test", target: "test", expectDiffs: false},
		{name: "different_strings", source: "a", target: "b", expectDiffs: true},
		{name: "equal_maps", source: map[string]any{"key": "value"}, target: map[string]any{"key": "value"}, expectDiffs: false},
		{name: "different_maps", source: map[string]any{"key": "a"}, target: map[string]any{"key": "b"}, expectDiffs: true},
		{name: "equal_slices", source: []any{"a", "b"}, target: []any{"a", "b"}, expectDiffs: false},
		{name: "different_slices", source: []any{"a"}, target: []any{"b"}, expectDiffs: true},
		{name: "type_mismatch_map_slice", source: map[string]any{}, target: []any{}, expectDiffs: true},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			var diffs statementDiffs
			computeValueDiffs("test", tt.source, tt.target, &diffs)
			if tt.expectDiffs {
				want.NotEmpty(diffs)
			} else {
				want.Empty(diffs)
			}
		})
	}
}

type computeSliceDiffsCase struct {
	name        testName
	source      []any
	target      []any
	expectDiffs expectBool
	expectOrder bool
}

func TestComputeSliceDiffs(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []computeSliceDiffsCase{
		{name: "identical", source: []any{"a", "b"}, target: []any{"a", "b"}, expectDiffs: false, expectOrder: false},
		{name: "different_length", source: []any{"a"}, target: []any{"a", "b"}, expectDiffs: true, expectOrder: false},
		{name: "reordered_same_elements", source: []any{"a", "b"}, target: []any{"b", "a"}, expectDiffs: true, expectOrder: true},
		{name: "different_elements", source: []any{"a", "b"}, target: []any{"c", "d"}, expectDiffs: true, expectOrder: false},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			var diffs statementDiffs
			computeSliceDiffs("test", tt.source, tt.target, &diffs)
			if !tt.expectDiffs {
				want.Empty(diffs)
				return
			}
			want.NotEmpty(diffs)
			if tt.expectOrder {
				want.Len(diffs, 1)
				want.Contains(string(diffs[0].Field), "order")
			}
		})
	}
}
