package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type registryGetCase struct {
	name        testName
	stmtType    pgQueryType
	expectKnown bool
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	reg := newRegistry()

	tests := []registryGetCase{
		{name: "create_table_found", stmtType: typeCreateTable, expectKnown: true},
		{name: "create_schema_found", stmtType: typeCreateSchema, expectKnown: true},
		{name: "drop_found", stmtType: typeDrop, expectKnown: true},
		{name: "grant_found", stmtType: typeGrant, expectKnown: true},
		{name: "unknown_not_found", stmtType: "unknown_type", expectKnown: false},
		{name: "empty_not_found", stmtType: "", expectKnown: false},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			_, known := reg.get(tt.stmtType)
			want.Equal(tt.expectKnown, known)
		})
	}
}

func TestRegistry_DefaultRegistryCoversAllTypes(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	reg := newRegistry()
	types := []pgQueryType{
		typeAlterTable, typeComment, typeCreateFunction, typeCreateSchema,
		typeCreateTable, typeCreateTrigger, typeIndex, typeView, typeDrop, typeGrant,
	}
	for _, ty := range types {
		_, known := reg.get(ty)
		want.True(known, "expected handler for %q", ty)
	}
	want.Len(reg, len(types))
}

type registryDiffCase struct {
	source    statementData
	target    statementData
	name      testName
	stmtType  pgQueryType
	expectNil bool
}

func TestRegistry_Diff(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	reg := newRegistry()

	tests := []registryDiffCase{
		{name: "unknown_type_returns_nil", stmtType: "unknown", source: statementData{}, target: statementData{}, expectNil: true},
		{name: "empty_type_returns_nil", stmtType: "", source: statementData{}, target: statementData{}, expectNil: true},
		{name: "known_equal_returns_nil", stmtType: typeCreateTable, source: statementData{"a": 1}, target: statementData{"a": 1}, expectNil: true},
		{name: "known_different_returns_diffs", stmtType: typeCreateTable, source: statementData{"a": 1}, target: statementData{"a": 2}, expectNil: false},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			diffs := reg.diff(tt.stmtType, tt.source, tt.target)
			if tt.expectNil {
				want.Nil(diffs)
			} else {
				want.NotEmpty(diffs)
			}
		})
	}
}

func TestGenericDiff(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	want.Nil(genericDiff(statementData{"a": 1}, statementData{"a": 1}))
	want.NotEmpty(genericDiff(statementData{"a": 1}, statementData{"a": 2}))
}
