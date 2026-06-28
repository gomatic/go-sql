// Package compare tells you what changed between two SQL scripts, at the AST
// level. It parses both inputs with the root [sql] package, normalizes each
// statement to a canonical JSON form (with positional metadata stripped), pairs
// statements up by a type-specific identity, and reports which statements were
// added, removed, and changed going from source to target.
//
// Identity is what makes two statements "the same object" across the two
// scripts: a CREATE TABLE is keyed by its qualified name, an ALTER TABLE by its
// table plus the column or constraint it touches, a GRANT by object and
// grantee, and so on. If a statement's type has no handler, or we can't derive
// its identity, we skip it.
//
// [sql]: github.com/gomatic/go-sql
package compare
