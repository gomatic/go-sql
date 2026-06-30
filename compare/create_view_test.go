package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type viewIdentityCase struct {
	name     testName
	sql      sqlStatement
	expected identity
}

func TestCreateViewIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []viewIdentityCase{
		{name: "simple_view", sql: "CREATE VIEW my_view AS SELECT 1", expected: "create.view:my_view"},
		{
			name:     "schema_qualified",
			sql:      "CREATE VIEW my_schema.my_view AS SELECT 1",
			expected: "create.view:my_schema.my_view",
		},
		{name: "or_replace", sql: "CREATE OR REPLACE VIEW my_view AS SELECT 1", expected: "create.view:my_view"},
		{name: "with_columns", sql: "CREATE VIEW my_view (col1, col2) AS SELECT 1, 2", expected: "create.view:my_view"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, createViewIdentity(parseTestSQL(t, must, tt.sql)))
		})
	}
}

type viewIdentityNilCase struct {
	name     testName
	stmt     statementData
	expected identity
}

func TestCreateViewIdentity_NilCases(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []viewIdentityNilCase{
		{name: "nil_statement", stmt: nil, expected: ""},
		{name: "empty_statement", stmt: statementData{}, expected: ""},
		{name: "missing_data", stmt: statementData{"stmt": map[string]any{}}, expected: ""},
		{name: "missing_view", stmt: statementData{"stmt": map[string]any{"data": map[string]any{}}}, expected: ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, createViewIdentity(tt.stmt))
		})
	}
}

type viewCompareCase struct {
	name        testName
	sourceSQL   sqlStatement
	targetSQL   sqlStatement
	expectDiffs expectBool
}

func TestCreateViewDiff(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []viewCompareCase{
		{
			name:        "identical_views",
			sourceSQL:   "CREATE VIEW v AS SELECT 1",
			targetSQL:   "CREATE VIEW v AS SELECT 1",
			expectDiffs: false,
		},
		{
			name:        "different_query",
			sourceSQL:   "CREATE VIEW v AS SELECT 1",
			targetSQL:   "CREATE VIEW v AS SELECT 2",
			expectDiffs: true,
		},
		{
			name:        "or_replace_vs_not",
			sourceSQL:   "CREATE VIEW v AS SELECT 1",
			targetSQL:   "CREATE OR REPLACE VIEW v AS SELECT 1",
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
