package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sql "github.com/gomatic/go-sql"
)

func TestFormatHouseSelectLeadingCommas(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a, b from t")
	require.NoError(t, err)
	assert.Equal(t, "select a\n     , b\n  from t", out)
}

func TestFormatHouseSelectWhereSplitsAnd(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t where x = 1 and y = 2")
	require.NoError(t, err)
	assert.Equal(t, "select a\n  from t\n where x = 1\n   and y = 2", out)
}

func TestFormatHouseSelectWithoutFrom(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select 1")
	require.NoError(t, err)
	assert.Equal(t, "select 1", out)
}

func TestFormatHouseSelectAliasesAndSchema(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a as x, b from s.t as u")
	require.NoError(t, err)
	assert.Equal(t, "select a as x\n     , b\n  from s.t as u", out)
}

func TestFormatHouseSelectKeepsExpressionsAndLowercases(t *testing.T) {
	t.Parallel()
	out, err := New().Format("SELECT a + b AS total FROM t WHERE x::int IN (1,2)")
	require.NoError(t, err)
	assert.Equal(t, "select a + b as total\n  from t\n where x::int in (1, 2)", out)
}

func TestFormatHouseBailsToCanonicalForOrderBy(t *testing.T) {
	t.Parallel()
	// order by isn't covered by the house renderer yet, so the gate falls back to
	// the canonical single-line form — still correct, just not house-styled.
	out, err := New().Format("select a, b from t order by a")
	require.NoError(t, err)
	assert.Equal(t, "select a, b from t order by a", out)
}

func TestFormatHouseBailsToCanonicalForJoin(t *testing.T) {
	t.Parallel()
	out, err := New().Format("select a from t1 join t2 on t1.id = t2.id")
	require.NoError(t, err)
	// A join in FROM isn't covered yet; the canonical deparse renders it on one line.
	assert.NotContains(t, out, "\n  from")
	assert.True(t, preservesMeaning("select a from t1 join t2 on t1.id = t2.id", out))
}

// The house output must always round-trip to the same statement.
func TestFormatHousePreservesMeaning(t *testing.T) {
	t.Parallel()
	for _, in := range []sql.SQL{
		"select a, b from t",
		"select a from t where x = 1 and y = 2",
		"select a as x, b from s.t as u",
	} {
		out, err := New().Format(in)
		require.NoError(t, err)
		assert.True(t, preservesMeaning(string(in), out), "input %q", in)
	}
}
