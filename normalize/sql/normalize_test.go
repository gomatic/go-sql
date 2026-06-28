package sqlnorm

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sql "github.com/gomatic/go-sql"
)

func TestNormalize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input SQL
		want  SQL
	}{
		{
			name:  "empty input",
			input: SQL(""),
			want:  SQL(""),
		},
		{
			name:  "basic whitespace",
			input: SQL("SELECT  *   FROM   table"),
			want:  SQL("SELECT * FROM table"),
		},
		{
			name:  "comments fall back to whitespace normalization",
			input: SQL("SELECT * FROM table; -- comment"),
			want:  SQL("SELECT * FROM table; -- comment"),
		},
		{
			name:  "trailing semicolon",
			input: SQL("SELECT * FROM table;"),
			want:  SQL("SELECT * FROM table"),
		},
		{
			name:  "newlines",
			input: SQL("SELECT\n*\nFROM\ntable"),
			want:  SQL("SELECT * FROM table"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.input.Normalize())
		})
	}
}

func TestNormalizeRoutineInsertSelect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input SQL
		want  SQL
	}{
		{
			name:  "simple insert select sorting",
			input: SQL("INSERT INTO t (b, a) SELECT y, x FROM s"),
			want:  SQL("INSERT INTO t (a, b) SELECT x, y FROM s"),
		},
		{
			name:  "insert select with expressions",
			input: SQL("INSERT INTO t (b, a) SELECT y + 1, x FROM s"),
			want:  SQL("INSERT INTO t (a, b) SELECT x, y + 1 FROM s"),
		},
		{
			name:  "insert select with aliases",
			input: SQL("INSERT INTO t (b, a) SELECT y AS col_y, x AS col_x FROM s"),
			want:  SQL("INSERT INTO t (a, b) SELECT x AS col_x, y AS col_y FROM s"),
		},
		{
			name:  "CTE with insert select",
			input: SQL("WITH d AS (DELETE FROM old RETURNING *) INSERT INTO t (b, a) SELECT y, x FROM d"),
			want:  SQL("WITH d AS (DELETE FROM old RETURNING *) INSERT INTO t (a, b) SELECT x, y FROM d"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.input.NormalizeRoutine())
		})
	}
}

func TestNormalizeStrict(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input SQL
		want  SQL
	}{
		{
			name:  "no sorting in strict mode",
			input: SQL("INSERT INTO t (b, a) SELECT y, x FROM s"),
			want:  SQL("INSERT INTO t (b, a) SELECT y, x FROM s"),
		},
		{
			name:  "no sorting simple select in strict mode",
			input: SQL("SELECT b, a FROM t"),
			want:  SQL("SELECT b, a FROM t"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.input.NormalizeStrict())
		})
	}
}

func TestNormalizeRoutineSelect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input SQL
		want  SQL
	}{
		{
			name:  "simple select sorting",
			input: SQL("SELECT b, a FROM t"),
			want:  SQL("SELECT a, b FROM t"),
		},
		{
			name:  "select with where clause sorting",
			input: SQL("SELECT b, a FROM t WHERE id > 10"),
			want:  SQL("SELECT a, b FROM t WHERE id > 10"),
		},
		{
			name:  "select with joins sorting",
			input: SQL("SELECT t2.a, t1.b FROM t1 JOIN t2 ON t1.id = t2.id"),
			want:  SQL("SELECT t1.b, t2.a FROM t1 JOIN t2 ON t1.id = t2.id"),
		},
		{
			name:  "complex select (group by) no sorting",
			input: SQL("SELECT b, a FROM t GROUP BY b, a"),
			want:  SQL("SELECT b, a FROM t GROUP BY b, a"),
		},
		{
			name:  "complex select (distinct) no sorting",
			input: SQL("SELECT DISTINCT b, a FROM t"),
			want:  SQL("SELECT DISTINCT b, a FROM t"),
		},
		{
			name:  "window function select sorting",
			input: SQL("SELECT b, a, row_number() OVER () FROM t"),
			want:  SQL("SELECT a, b, row_number() OVER () FROM t"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.input.NormalizeRoutine())
		})
	}
}

// TestNormalizeWithOptionsParseFailureFallback checks that SQL we can't parse
// falls back to whitespace normalization instead of erroring out.
func TestNormalizeWithOptionsParseFailureFallback(t *testing.T) {
	t.Parallel()
	// `table` is a reserved keyword, so this won't parse and has to fall back to
	// plain whitespace collapsing.
	got := normalizeWithOptions(SQL("SELECT  b   FROM   table"), sortColumnsEnabled(false), sql.Deparse)
	assert.Equal(t, SQL("SELECT b FROM table"), got)
}

// TestNormalizeWithOptionsDeparseFailureFallback drives the deparse-failure branch
// with an injected deparser, since with real input a successful parse always
// deparses cleanly.
func TestNormalizeWithOptionsDeparseFailureFallback(t *testing.T) {
	t.Parallel()
	failingDeparse := func(*pg_query.ParseResult) (sql.SQL, error) {
		return "", sql.ErrDeparse
	}
	got := normalizeWithOptions(SQL("SELECT  1  FROM  t;"), sortColumnsEnabled(true), failingDeparse)
	require.Equal(t, SQL("SELECT 1 FROM t"), got)
}
