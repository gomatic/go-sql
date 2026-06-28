package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type alterIdentityCase struct {
	name     testName
	sql      sqlStatement
	expected identity
}

func TestAlterTableIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []alterIdentityCase{
		{name: "add_column", sql: "ALTER TABLE my_schema.my_table ADD COLUMN new_col text", expected: "alter.table.add_column:my_schema.my_table:new_col"},
		{name: "add_constraint", sql: "ALTER TABLE my_table ADD CONSTRAINT my_pk PRIMARY KEY (id)", expected: "alter.table.add_constraint:my_table:my_pk"},
		{name: "drop_column", sql: "ALTER TABLE my_table DROP COLUMN old_col", expected: "alter.table.drop_column:my_table:old_col"},
		{name: "drop_constraint", sql: "ALTER TABLE my_table DROP CONSTRAINT my_fk", expected: "alter.table.drop_constraint:my_table:my_fk"},
		{name: "drop_not_null", sql: "ALTER TABLE my_table ALTER COLUMN col DROP NOT NULL", expected: "alter.table.drop_not_null:my_table:col"},
		{name: "set_not_null", sql: "ALTER TABLE my_table ALTER COLUMN col SET NOT NULL", expected: "alter.table.set_not_null:my_table:col"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, alterTableIdentity(parseTestSQL(t, must, tt.sql)))
		})
	}
}

type alterIdentityRawCase struct {
	name     testName
	stmt     statementData
	expected identity
}

func TestAlterTableIdentity_RawCases(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	wrap := func(data map[string]any) statementData {
		return statementData{"stmt": map[string]any{"data": data}}
	}

	tests := []alterIdentityRawCase{
		{name: "nil_statement", stmt: nil, expected: ""},
		{name: "missing_relation", stmt: wrap(map[string]any{}), expected: ""},
		{name: "no_cmds_is_unknown", stmt: wrap(map[string]any{"relation": map[string]any{"relname": "t"}}), expected: "alter.table.unknown:t"},
		{
			name:     "non_map_cmd_is_unknown",
			stmt:     wrap(map[string]any{"relation": map[string]any{"relname": "t"}, "cmds": []any{"notamap"}}),
			expected: "alter.table.unknown:t",
		},
		{
			name:     "cmd_without_subtype_is_unknown",
			stmt:     wrap(map[string]any{"relation": map[string]any{"relname": "t"}, "cmds": []any{map[string]any{"node": map[string]any{"alter_table_cmd": map[string]any{}}}}}),
			expected: "alter.table.unknown:t",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, alterTableIdentity(tt.stmt))
		})
	}
}

type alterTargetCase struct {
	cmd      statementData
	name     testName
	expected alterTargetName
	subtype  alterSubtype
}

func TestExtractAlterTarget(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []alterTargetCase{
		{name: "name_field_wins", cmd: statementData{"name": "col"}, subtype: alterAddColumn, expected: "col"},
		{name: "add_column_from_def", cmd: statementData{"def": map[string]any{"node": map[string]any{"column_def": map[string]any{"colname": "c"}}}}, subtype: alterAddColumn, expected: "c"},
		{name: "add_constraint_from_def", cmd: statementData{"def": map[string]any{"node": map[string]any{"constraint": map[string]any{"conname": "pk"}}}}, subtype: alterAddConstraint, expected: "pk"},
		{name: "unknown_subtype_empty", cmd: statementData{}, subtype: alterSubtype(9999), expected: ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, extractAlterTarget(tt.cmd, tt.subtype))
		})
	}
}

type alterCompareCase struct {
	name        testName
	sourceSQL   sqlStatement
	targetSQL   sqlStatement
	expectDiffs expectBool
}

func TestAlterTableDiff(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []alterCompareCase{
		{name: "different_column_type", sourceSQL: "ALTER TABLE t ADD COLUMN c text", targetSQL: "ALTER TABLE t ADD COLUMN c int", expectDiffs: true},
		{name: "identical_add_column", sourceSQL: "ALTER TABLE t ADD COLUMN c text", targetSQL: "ALTER TABLE t ADD COLUMN c text", expectDiffs: false},
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
