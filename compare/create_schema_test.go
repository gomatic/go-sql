package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// astTestCase checks the decoded AST structure.
type astTestCase struct {
	expectedAST statementData
	name        testName
	sql         sqlStatement
}

// diffTestCase checks statement comparison.
type diffTestCase struct {
	name          testName
	sourceSQL     sqlStatement
	targetSQL     sqlStatement
	expectEqual   expectEqual
	expectedDiffs diffCount
}

// identityTestCase checks identity extraction.
type identityTestCase struct {
	name             testName
	sql              sqlStatement
	expectedIdentity identity
}

func TestCreateSchema_ASTStructure(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []astTestCase{
		{name: "simple_schema", sql: `create schema my_schema`, expectedAST: statementData{"schemaname": "my_schema"}},
		{name: "schema_if_not_exists", sql: `create schema if not exists v1_armada_cell_type`, expectedAST: statementData{"if_not_exists": true, "schemaname": "v1_armada_cell_type"}},
	}

	for _, tc := range tests {
		t.Run(string(tc.name), func(t *testing.T) {
			t.Parallel()
			data := parseAndExtractData(t, must, tc.sql, typeCreateSchema)
			assertASTContains(t, want, tc.expectedAST, data)
		})
	}
}

func TestCreateSchemaIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []identityTestCase{
		{name: "simple_schema", sql: `create schema my_schema`, expectedIdentity: "create.schema:my_schema"},
		{name: "schema_if_not_exists", sql: `create schema if not exists v1_armada_cell_type`, expectedIdentity: "create.schema:v1_armada_cell_type"},
	}

	for _, tc := range tests {
		t.Run(string(tc.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tc.expectedIdentity, createSchemaIdentity(parseStatement(t, must, tc.sql)))
		})
	}
}

func TestCreateSchemaIdentity_NilCases(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Equal(identity(""), createSchemaIdentity(nil))
	want.Equal(identity(""), createSchemaIdentity(statementData{"stmt": map[string]any{"data": map[string]any{}}}))
}

func TestCreateSchemaDiff(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []diffTestCase{
		{name: "identical_schemas", sourceSQL: `create schema my_schema`, targetSQL: `create schema my_schema`, expectEqual: true, expectedDiffs: 0},
		{name: "different_if_not_exists", sourceSQL: `create schema my_schema`, targetSQL: `create schema if not exists my_schema`, expectEqual: false, expectedDiffs: 1},
	}

	for _, tc := range tests {
		t.Run(string(tc.name), func(t *testing.T) {
			t.Parallel()
			diffs := genericDiff(parseStatement(t, must, tc.sourceSQL), parseStatement(t, must, tc.targetSQL))
			if tc.expectEqual {
				want.Nil(diffs)
			} else {
				want.Len(diffs, int(tc.expectedDiffs))
			}
		})
	}
}
