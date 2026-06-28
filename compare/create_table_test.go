package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tableIdentityCase struct {
	name     testName
	sql      sqlStatement
	expected identity
}

func TestCreateTableIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []tableIdentityCase{
		{name: "simple_table", sql: "CREATE TABLE my_table (id int)", expected: "create.table:my_table"},
		{name: "schema_qualified", sql: "CREATE TABLE my_schema.my_table (id int)", expected: "create.table:my_schema.my_table"},
		{name: "if_not_exists", sql: "CREATE TABLE IF NOT EXISTS my_table (id int)", expected: "create.table:my_table"},
		{name: "with_columns", sql: "CREATE TABLE users (id int, name text, email text)", expected: "create.table:users"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, createTableIdentity(parseTestSQL(t, must, tt.sql)))
		})
	}
}

type tableIdentityNilCase struct {
	name     testName
	stmt     statementData
	expected identity
}

func TestCreateTableIdentity_NilCases(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []tableIdentityNilCase{
		{name: "nil_statement", stmt: nil, expected: ""},
		{name: "empty_statement", stmt: statementData{}, expected: ""},
		{name: "missing_stmt_wrapper", stmt: statementData{"data": map[string]any{}}, expected: ""},
		{name: "missing_data", stmt: statementData{"stmt": map[string]any{}}, expected: ""},
		{name: "missing_relation", stmt: statementData{"stmt": map[string]any{"data": map[string]any{}}}, expected: ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, createTableIdentity(tt.stmt))
		})
	}
}

type tableCompareCase struct {
	name        testName
	sourceSQL   sqlStatement
	targetSQL   sqlStatement
	expectDiffs expectBool
}

func TestCreateTableDiff(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []tableCompareCase{
		{name: "identical_tables", sourceSQL: "CREATE TABLE t (id int)", targetSQL: "CREATE TABLE t (id int)", expectDiffs: false},
		{name: "different_column_type", sourceSQL: "CREATE TABLE t (id int)", targetSQL: "CREATE TABLE t (id text)", expectDiffs: true},
		{name: "different_column_name", sourceSQL: "CREATE TABLE t (id int)", targetSQL: "CREATE TABLE t (user_id int)", expectDiffs: true},
		{name: "additional_column", sourceSQL: "CREATE TABLE t (id int)", targetSQL: "CREATE TABLE t (id int, name text)", expectDiffs: true},
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
