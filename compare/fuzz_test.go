package compare

import (
	"testing"

	sql "github.com/gomatic/go-sql"
)

// compareSeeds walks the edge matrix for AST diffing, leaning on the DDL kinds
// the engine recognizes: empty, whitespace, malformed, multi-statement scripts,
// every handled statement kind, comments, unicode, and unbalanced input.
var compareSeeds = []string{
	"",
	"   \t\n ",
	"not valid sql ((",
	"select 'unterminated",
	"create table t (a int primary key, b text)",
	"create schema s",
	"create index i on t (a, b)",
	"create view v as select a, b from t",
	"alter table t add column c int",
	"drop table t",
	"grant select on t to r",
	"comment on table t is 'note 你好'",
	"create table a (x int); create table b (y text);",
	"create function f() returns trigger as $$ begin return new; end $$ language plpgsql",
}

// FuzzCompareReflexive asserts comparison never panics and is reflexive: a
// script compared against itself reports no added, removed, or changed
// statements. That is the engine's defining promise — equal inputs are equal.
func FuzzCompareReflexive(f *testing.F) {
	for _, s := range compareSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		result, err := Compare(sql.SQL(in), sql.SQL(in))
		if err != nil {
			return
		}
		if result.HasChanges() {
			t.Fatalf("self-comparison reported changes for %q: %+v", in, result)
		}
	})
}
