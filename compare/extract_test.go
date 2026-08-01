package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestExtractInt(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []extractIntCase{
		{name: "nil_data", data: nil, key: "count", expectedValue: 0, expectedFound: false},
		{name: "empty_data", data: statementData{}, key: "count", expectedValue: 0, expectedFound: false},
		{name: "key_not_found", data: statementData{"other": 42}, key: "count", expectedValue: 0, expectedFound: false},
		{name: "key_not_int", data: statementData{"count": "42"}, key: "count", expectedValue: 0, expectedFound: false},
		{name: "key_is_int", data: statementData{"count": 42}, key: "count", expectedValue: 42, expectedFound: true},
		{
			name:          "key_is_int64",
			data:          statementData{"count": int64(42)},
			key:           "count",
			expectedValue: 42,
			expectedFound: true,
		},
		{
			name:          "key_is_float64",
			data:          statementData{"count": float64(42)},
			key:           "count",
			expectedValue: 42,
			expectedFound: true,
		},
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

func TestExtractMap(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []extractMapCase{
		{name: "nil_data", data: nil, key: "nested", expected: nil},
		{name: "empty_data", data: statementData{}, key: "nested", expected: nil},
		{name: "key_not_found", data: statementData{"other": map[string]any{}}, key: "nested", expected: nil},
		{name: "key_not_map", data: statementData{"nested": "not a map"}, key: "nested", expected: nil},
		{
			name:     "key_found",
			data:     statementData{"nested": map[string]any{"inner": "value"}},
			key:      "nested",
			expected: statementData{"inner": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, extractMap(tt.data, tt.key))
		})
	}
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

func TestExtractStatementData(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []extractStatementDataCase{
		{name: "nil_stmt", stmt: nil, expected: nil},
		{
			name:     "stmt_wrapped",
			stmt:     statementData{"stmt": map[string]any{"data": map[string]any{"a": "b"}}},
			expected: statementData{"a": "b"},
		},
		{
			name:     "top_level_data",
			stmt:     statementData{"data": map[string]any{"a": "b"}},
			expected: statementData{"a": "b"},
		},
		{name: "missing_data", stmt: statementData{"stmt": map[string]any{}}, expected: nil},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, extractStatementData(tt.stmt))
		})
	}
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

func TestExtractSchemaAndName(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []schemaAndNameCase{
		{name: "empty_data", data: statementData{}, expectedSchema: "", expectedName: ""},
		{
			name:           "only_relname",
			data:           statementData{"relname": "my_table"},
			expectedSchema: "",
			expectedName:   "my_table",
		},
		{
			name:           "only_schemaname",
			data:           statementData{"schemaname": "my_schema"},
			expectedSchema: "my_schema",
			expectedName:   "",
		},
		{
			name:           "both",
			data:           statementData{"schemaname": "my_schema", "relname": "my_table"},
			expectedSchema: "my_schema",
			expectedName:   "my_table",
		},
		{
			name:           "wrong_types",
			data:           statementData{"schemaname": 123, "relname": 456},
			expectedSchema: "",
			expectedName:   "",
		},
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
