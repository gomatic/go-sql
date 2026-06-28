package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatementType(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	want.Equal(pgQueryType(""), statementType(statementData{}), "no stmt wrapper yields empty type")
	want.Equal(typeCreateTable, statementType(statementData{
		"stmt": map[string]any{"type": string(typeCreateTable)},
	}))
}

func TestResultHasChanges(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	want.False(Result{}.HasChanges())
	want.True(Result{Added: []StatementResult{{}}}.HasChanges())
	want.True(Result{Changed: []StatementResult{{}}}.HasChanges())
	want.True(Result{Removed: []StatementResult{{}}}.HasChanges())
}

func TestPublicDiffs(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	in := statementDiffs{{Field: "a.b", Source: 1, Target: 2}}
	out := publicDiffs(in)
	want.Equal([]Diff{{Field: "a.b", Source: 1, Target: 2}}, out)
}
