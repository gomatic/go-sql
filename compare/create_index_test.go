package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type indexIdentityCase struct {
	name     testName
	sql      sqlStatement
	expected identity
}

func TestCreateIndexIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []indexIdentityCase{
		{
			name:     "simple_index",
			sql:      "CREATE INDEX idx_name ON my_table (col)",
			expected: "create.index:my_table:idx_name",
		},
		{
			name:     "schema_qualified_table",
			sql:      "CREATE INDEX idx_name ON my_schema.my_table (col)",
			expected: "create.index:my_schema.my_table:idx_name",
		},
		{
			name:     "unique_index",
			sql:      "CREATE UNIQUE INDEX idx_name ON my_table (col)",
			expected: "create.index:my_table:idx_name",
		},
		{
			name:     "if_not_exists",
			sql:      "CREATE INDEX IF NOT EXISTS idx_name ON my_table (col)",
			expected: "create.index:my_table:idx_name",
		},
		{
			name:     "multi_column",
			sql:      "CREATE INDEX idx_name ON my_table (col1, col2)",
			expected: "create.index:my_table:idx_name",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, createIndexIdentity(parseTestSQL(t, must, tt.sql)))
		})
	}
}

type indexIdentityNilCase struct {
	name     testName
	stmt     statementData
	expected identity
}

func TestCreateIndexIdentity_NilCases(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []indexIdentityNilCase{
		{name: "nil_statement", stmt: nil, expected: ""},
		{name: "empty_statement", stmt: statementData{}, expected: ""},
		{name: "missing_data", stmt: statementData{"stmt": map[string]any{}}, expected: ""},
		{name: "missing_idxname", stmt: statementData{"stmt": map[string]any{"data": map[string]any{}}}, expected: ""},
		{
			name:     "missing_relation",
			stmt:     statementData{"stmt": map[string]any{"data": map[string]any{"idxname": "idx"}}},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, createIndexIdentity(tt.stmt))
		})
	}
}

type indexCompareCase struct {
	name        testName
	sourceSQL   sqlStatement
	targetSQL   sqlStatement
	expectDiffs expectBool
}

func TestCreateIndexDiff(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []indexCompareCase{
		{
			name:        "identical_indexes",
			sourceSQL:   "CREATE INDEX idx ON t (col)",
			targetSQL:   "CREATE INDEX idx ON t (col)",
			expectDiffs: false,
		},
		{
			name:        "different_column",
			sourceSQL:   "CREATE INDEX idx ON t (col1)",
			targetSQL:   "CREATE INDEX idx ON t (col2)",
			expectDiffs: true,
		},
		{
			name:        "unique_vs_not",
			sourceSQL:   "CREATE INDEX idx ON t (col)",
			targetSQL:   "CREATE UNIQUE INDEX idx ON t (col)",
			expectDiffs: true,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			diffs := genericDiff(parseTestSQL(t, must, tt.sourceSQL), parseTestSQL(t, must, tt.targetSQL))
			if tt.expectDiffs {
				want.NotEmpty(diffs)
			} else {
				want.Empty(diffs)
			}
		})
	}
}
