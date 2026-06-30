// Package sqlnorm canonicalizes SQL text so you can compare it for meaning rather
// than formatting. It parses the SQL into PostgreSQL's AST (through the parent
// [sql] package), optionally reorders column lists, and deparses back to a form
// with whitespace normalized and the trailing semicolon trimmed. If the input
// won't parse, we fall back to plain whitespace collapsing, so every input still
// gives you a deterministic result.
//
// We call the package sqlnorm, not sql, so it doesn't clash with the parent
// [github.com/gomatic/go-sql] package it imports.
package sqlnorm

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	sql "github.com/gomatic/go-sql"
)

// sortColumnsEnabled says whether we canonically reorder column lists.
type sortColumnsEnabled bool

// deparseFunc renders a parsed AST back to SQL text. We inject it so a test can
// reach the deparse-failure fallback; with real input a successful parse always
// deparses cleanly, so that branch is otherwise impossible to hit.
type deparseFunc func(*pg_query.ParseResult) (sql.SQL, error)

// SQL is normalized SQL text.
type SQL string

// Normalize reformats the SQL to canonical form without touching column order,
// so it's safe where that order actually matters.
func (p SQL) Normalize() SQL { return normalize(p) }

// NormalizeRoutine reformats the SQL and also sorts the column lists of simple
// SELECT and INSERT … SELECT statements. That's what you want for routine bodies,
// where column order means nothing and you're after a deterministic comparison.
func (p SQL) NormalizeRoutine() SQL { return normalizeRoutine(p) }

// NormalizeStrict reformats the SQL without reordering, just like [SQL.Normalize];
// it's here to spell out the intent where column order matters.
func (p SQL) NormalizeStrict() SQL { return normalizeStrict(p) }

func normalize(code SQL) SQL {
	return normalizeWithOptions(code, sortColumnsEnabled(false), sql.Deparse)
}

func normalizeRoutine(code SQL) SQL {
	return normalizeWithOptions(code, sortColumnsEnabled(true), sql.Deparse)
}

func normalizeStrict(code SQL) SQL {
	return normalizeWithOptions(code, sortColumnsEnabled(false), sql.Deparse)
}

// normalizeWithOptions parses, optionally sorts, and deparses code, falling back
// to whitespace normalization if either the parse or the deparse fails.
func normalizeWithOptions(code SQL, sortColumns sortColumnsEnabled, deparse deparseFunc) SQL {
	if code == "" {
		return ""
	}

	tree, err := sql.Parse(sql.SQL(code))
	if err != nil {
		return normalizeWhitespace(code)
	}

	if bool(sortColumns) {
		sql.SortColumnLists(tree)
	}

	deparsed, err := deparse(tree)
	if err != nil {
		return normalizeWhitespace(code)
	}

	return trimCanonical(string(deparsed))
}

// normalizeWhitespace squashes runs of whitespace down to single spaces and trims
// trailing semicolons. It's the fallback we use when parsing fails.
func normalizeWhitespace(code SQL) SQL {
	return trimCanonical(strings.Join(strings.Fields(string(code)), " "))
}

// trimCanonical strips surrounding whitespace and every trailing semicolon —
// each a meaningless empty statement — along with any whitespace padding them.
// Right-trimming semicolons and whitespace together (rather than space-then-one-
// semicolon) keeps the result canonical and idempotent: inputs like "0 ;" and
// "0 ;;" both settle on "0" in a single pass instead of leaving a stray space or
// a residual trailing semicolon for a second pass to clean up.
func trimCanonical(s string) SQL {
	return SQL(strings.TrimSpace(strings.TrimRight(strings.TrimSpace(s), "; \t\n\r\f\v")))
}
