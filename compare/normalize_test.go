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
		{
			name:     "removes_location",
			input:    statementData{"location": 42, "name": "test"},
			expected: statementData{"name": "test"},
		},
		{
			name:     "removes_lineno",
			input:    statementData{"lineno": 1, "name": "test"},
			expected: statementData{"name": "test"},
		},
		{
			name:     "removes_colno",
			input:    statementData{"colno": 5, "name": "test"},
			expected: statementData{"name": "test"},
		},
		{
			name:     "removes_stmt_len",
			input:    statementData{"stmt_len": 100, "name": "test"},
			expected: statementData{"name": "test"},
		},
		{
			name:     "removes_stmt_location",
			input:    statementData{"stmt_location": 0, "name": "test"},
			expected: statementData{"name": "test"},
		},
		{
			name:     "nested_map",
			input:    statementData{"outer": map[string]any{"location": 1, "inner": "value"}},
			expected: statementData{"outer": map[string]any{"inner": "value"}},
		},
		{
			name:     "nested_slice",
			input:    statementData{"items": []any{map[string]any{"location": 1, "name": "a"}}},
			expected: statementData{"items": []any{map[string]any{"name": "a"}}},
		},
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
		{
			name:     "map_value",
			input:    map[string]any{"location": 1, "name": "test"},
			expected: map[string]any{"name": "test"},
		},
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
		{
			name:     "equal_content",
			source:   statementData{"name": "test"},
			target:   statementData{"name": "test"},
			expected: true,
		},
		{
			name:     "different_content",
			source:   statementData{"name": "a"},
			target:   statementData{"name": "b"},
			expected: false,
		},
		{
			name:     "location_ignored",
			source:   statementData{"location": 1, "name": "test"},
			target:   statementData{"location": 2, "name": "test"},
			expected: true,
		},
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

type extractStringCase struct {
	name     testName
	data     statementData
	key      fieldKey
	expected stringValue
}

type extractIntCase struct {
	name          testName
	data          statementData
	key           fieldKey
	expectedValue intValue
	expectedFound intFound
}

type extractMapCase struct {
	data     statementData
	expected statementData
	name     testName
	key      fieldKey
}

type extractSliceCase struct {
	name     testName
	data     statementData
	key      fieldKey
	expected []any
}

type extractStatementDataCase struct {
	stmt     statementData
	expected statementData
	name     testName
}

type toSnakeCaseCase struct {
	name     testName
	input    stringValue
	expected stringValue
}

type slicesHaveSameElementsCase struct {
	name     testName
	source   []any
	target   []any
	expected elementsMatch
}

type schemaAndNameCase struct {
	name           testName
	data           statementData
	expectedSchema schemaName
	expectedName   objectName
}

type qualifiedNameCase struct {
	name     testName
	schema   schemaName
	objName  objectName
	expected qualifiedName
}

type computeValueDiffsCase struct {
	source      any
	target      any
	name        testName
	expectDiffs expectBool
}

type computeSliceDiffsCase struct {
	name        testName
	source      []any
	target      []any
	expectDiffs expectBool
	expectOrder bool
}
