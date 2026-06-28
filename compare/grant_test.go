package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type grantIdentityCase struct {
	name     testName
	sql      sqlStatement
	expected identity
}

func TestGrantIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []grantIdentityCase{
		{name: "grant_on_table", sql: "GRANT SELECT ON my_table TO my_role", expected: "grant.table:my_table:my_role"},
		{name: "grant_on_schema_table", sql: "GRANT SELECT ON my_schema.my_table TO my_role", expected: "grant.table:my_schema.my_table:my_role"},
		{name: "revoke_on_table", sql: "REVOKE SELECT ON my_table FROM my_role", expected: "revoke.table:my_table:my_role"},
		{name: "grant_on_schema", sql: "GRANT USAGE ON SCHEMA my_schema TO my_role", expected: "grant.schema:my_schema:my_role"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, grantIdentity(parseTestSQL(t, must, tt.sql)))
		})
	}
}

func TestGrantIdentity_NilCases(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Equal(identity(""), grantIdentity(nil))
	want.Equal(identity(""), grantIdentity(statementData{}))
}

func TestGrantDiff(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []struct {
		name        testName
		sourceSQL   sqlStatement
		targetSQL   sqlStatement
		expectDiffs expectBool
	}{
		{name: "identical_grants", sourceSQL: "GRANT SELECT ON t TO r", targetSQL: "GRANT SELECT ON t TO r", expectDiffs: false},
		{name: "different_privilege", sourceSQL: "GRANT SELECT ON t TO r", targetSQL: "GRANT INSERT ON t TO r", expectDiffs: true},
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

type extractGrantTypeCase struct {
	name     testName
	data     statementData
	expected grantType
}

func TestExtractGrantType(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []extractGrantTypeCase{
		{name: "is_grant_true", data: statementData{"is_grant": true}, expected: "grant"},
		{name: "is_grant_false", data: statementData{"is_grant": false}, expected: "revoke"},
		{name: "is_grant_missing", data: statementData{}, expected: "revoke"},
		{name: "is_grant_wrong_type", data: statementData{"is_grant": "true"}, expected: "revoke"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, extractGrantType(tt.data))
		})
	}
}

func TestExtractGrantObjectName_Fallbacks(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	// Nothing in objects at all.
	want.Equal(qualifiedName(""), extractGrantObjectName(statementData{}))

	// A node that's neither a range_var nor a string_ gives back no name.
	node := statementData{"objects": []any{map[string]any{"node": map[string]any{"role_spec": map[string]any{}}}}}
	want.Equal(qualifiedName(""), extractGrantObjectName(node))

	// A string_ node (a schema grant) gives back the bare name.
	strNode := statementData{"objects": []any{map[string]any{"node": map[string]any{"string_": map[string]any{"sval": "app"}}}}}
	want.Equal(qualifiedName("app"), extractGrantObjectName(strNode))
}

func TestExtractGrantee_NoGrantees(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Equal(granteeName(""), extractGrantee(statementData{}))
}
