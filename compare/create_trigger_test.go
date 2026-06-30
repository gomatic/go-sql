package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type triggerIdentityCase struct {
	name     testName
	sql      sqlStatement
	expected identity
}

func TestCreateTriggerIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []triggerIdentityCase{
		{
			name:     "simple_trigger",
			sql:      "CREATE TRIGGER my_trigger BEFORE INSERT ON my_table FOR EACH ROW EXECUTE FUNCTION my_func()",
			expected: "create.trigger:my_trigger:my_table",
		},
		{
			name:     "schema_qualified_table",
			sql:      "CREATE TRIGGER my_trigger BEFORE INSERT ON my_schema.my_table FOR EACH ROW EXECUTE FUNCTION my_func()",
			expected: "create.trigger:my_trigger:my_schema.my_table",
		},
		{
			name:     "after_update",
			sql:      "CREATE TRIGGER my_trigger AFTER UPDATE ON my_table FOR EACH ROW EXECUTE FUNCTION my_func()",
			expected: "create.trigger:my_trigger:my_table",
		},
		{
			name:     "or_replace",
			sql:      "CREATE OR REPLACE TRIGGER my_trigger BEFORE INSERT ON my_table FOR EACH ROW EXECUTE FUNCTION my_func()",
			expected: "create.trigger:my_trigger:my_table",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, createTriggerIdentity(parseTestSQL(t, must, tt.sql)))
		})
	}
}

type triggerIdentityNilCase struct {
	name     testName
	stmt     statementData
	expected identity
}

func TestCreateTriggerIdentity_NilCases(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []triggerIdentityNilCase{
		{name: "nil_statement", stmt: nil, expected: ""},
		{name: "empty_statement", stmt: statementData{}, expected: ""},
		{name: "missing_data", stmt: statementData{"stmt": map[string]any{}}, expected: ""},
		{name: "missing_trigname", stmt: statementData{"stmt": map[string]any{"data": map[string]any{}}}, expected: ""},
		{
			name:     "trigname_without_relation",
			stmt:     statementData{"stmt": map[string]any{"data": map[string]any{"trigname": "tg"}}},
			expected: "create.trigger:tg:",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, createTriggerIdentity(tt.stmt))
		})
	}
}

type triggerCompareCase struct {
	name        testName
	sourceSQL   sqlStatement
	targetSQL   sqlStatement
	expectDiffs expectBool
}

func TestCreateTriggerDiff(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []triggerCompareCase{
		{
			name:        "identical_triggers",
			sourceSQL:   "CREATE TRIGGER t BEFORE INSERT ON tbl FOR EACH ROW EXECUTE FUNCTION f()",
			targetSQL:   "CREATE TRIGGER t BEFORE INSERT ON tbl FOR EACH ROW EXECUTE FUNCTION f()",
			expectDiffs: false,
		},
		{
			name:        "different_timing",
			sourceSQL:   "CREATE TRIGGER t BEFORE INSERT ON tbl FOR EACH ROW EXECUTE FUNCTION f()",
			targetSQL:   "CREATE TRIGGER t AFTER INSERT ON tbl FOR EACH ROW EXECUTE FUNCTION f()",
			expectDiffs: true,
		},
		{
			name:        "different_event",
			sourceSQL:   "CREATE TRIGGER t BEFORE INSERT ON tbl FOR EACH ROW EXECUTE FUNCTION f()",
			targetSQL:   "CREATE TRIGGER t BEFORE UPDATE ON tbl FOR EACH ROW EXECUTE FUNCTION f()",
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
