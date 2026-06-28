package compare

import (
	"encoding/json"
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sql "github.com/gomatic/go-sql"
)

func TestCompare_AddedAndRemoved(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	source := sqlFromFile(t, must, "source.sql")
	target := sqlFromFile(t, must, "target.sql")

	result, err := Compare(source, target)
	must.NoError(err)

	want.NotEmpty(result.Added, "target adds statements")
	want.NotEmpty(result.Removed, "source removes statements")
	want.True(result.HasChanges())
}

func TestCompare_IdenticalHasNoChanges(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	source := sqlFromFile(t, must, "source.sql")

	result, err := Compare(source, source)
	must.NoError(err)

	want.Empty(result.Added)
	want.Empty(result.Changed)
	want.Empty(result.Removed)
	want.False(result.HasChanges())
}

func TestCompare_DetectsChangedStatement(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	result, err := Compare(
		"CREATE TABLE app.t (id int)",
		"CREATE TABLE app.t (id text)",
	)
	must.NoError(err)

	want.Len(result.Changed, 1)
	want.Empty(result.Added)
	want.Empty(result.Removed)
	want.NotEmpty(result.Changed[0].Diffs)
	want.Equal("create.table:app.t", result.Changed[0].Identity)
	want.Equal(string(typeCreateTable), result.Changed[0].Type)
}

func TestCompare_AddedCarriesStatement(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	result, err := Compare("", "CREATE TABLE app.t (id int)")
	must.NoError(err)

	must.Len(result.Added, 1)
	want.Equal("create.table:app.t", result.Added[0].Identity)
	want.NotNil(result.Added[0].Statement)
	want.Empty(result.Added[0].Diffs)
}

func TestCompare_ParseErrorSource(t *testing.T) {
	t.Parallel()
	_, must := assert.New(t), require.New(t)

	_, err := Compare("NOT VALID SQL ((", "SELECT 1")
	must.Error(err)
	must.ErrorIs(err, sql.ErrParse)
}

func TestCompare_ParseErrorTarget(t *testing.T) {
	t.Parallel()
	_, must := assert.New(t), require.New(t)

	_, err := Compare("SELECT 1", "NOT VALID SQL ((")
	must.Error(err)
	must.ErrorIs(err, sql.ErrParse)
}

func TestCompareWith_ConvertErrorSource(t *testing.T) {
	t.Parallel()
	_, must := assert.New(t), require.New(t)

	failingEncode := func(*pg_query.ParseResult) ([]json.RawMessage, error) {
		return nil, errSentinel
	}

	_, err := compareWith(failingEncode, json.Unmarshal, "SELECT 1", "SELECT 1")
	must.Error(err)
	must.ErrorIs(err, ErrConvert)
	must.ErrorIs(err, errSentinel)
}

func TestCompareWith_ConvertErrorTarget(t *testing.T) {
	t.Parallel()
	_, must := assert.New(t), require.New(t)

	calls := 0
	encodeFailsOnTarget := func(tree *pg_query.ParseResult) ([]json.RawMessage, error) {
		calls++
		if calls == 1 {
			return sql.ToJSON(tree)
		}
		return nil, errSentinel
	}

	_, err := compareWith(encodeFailsOnTarget, json.Unmarshal, "SELECT 1", "SELECT 2")
	must.Error(err)
	must.ErrorIs(err, ErrConvert)
}

func TestCompareWith_DecodeError(t *testing.T) {
	t.Parallel()
	_, must := assert.New(t), require.New(t)

	failingDecode := func([]byte, any) error { return errSentinel }

	_, err := compareWith(sql.ToJSON, failingDecode, "SELECT 1", "SELECT 1")
	must.Error(err)
	must.ErrorIs(err, ErrDecode)
	must.ErrorIs(err, errSentinel)
}

func TestIndexStatements_SkipsUnhandledType(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	// A plain SELECT has no handler, so it should get skipped.
	stmts := []statementData{parseStatement(t, must, "SELECT 1")}
	indexed := indexStatements(newRegistry(), stmts)
	want.Empty(indexed)
}

func TestIndexStatements_SkipsEmptyIdentity(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	// A handled type whose identity we can't derive gets skipped.
	stmts := []statementData{{
		"stmt": map[string]any{"type": string(typeCreateTable), "data": map[string]any{}},
	}}
	indexed := indexStatements(newRegistry(), stmts)
	want.Empty(indexed)
}

func TestIndexStatements_IndexesByIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	stmts := []statementData{parseStatement(t, must, "CREATE TABLE app.t (id int)")}
	indexed := indexStatements(newRegistry(), stmts)
	_, found := indexed["create.table:app.t"]
	want.True(found)
}

// errSentinel is a stand-in cause we feed into the injected failure paths.
const errSentinel sentinelError = "injected failure"

type sentinelError string

func (e sentinelError) Error() string { return string(e) }

var _ error = sentinelError("")
