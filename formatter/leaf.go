package formatter

import (
	"strings"

	errs "github.com/gomatic/go-error"
	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/gomatic/go-sql"
)

// ErrDeparseLeaf means PostgreSQL's deparser couldn't render a leaf expression.
// It's effectively unreachable for a node that came from a successful parse, so
// the verification gate, not callers, is what ever surfaces it. Match it with
// [errors.Is].
const ErrDeparseLeaf errs.Const = "deparse leaf"

// selectPrefix is what [deparseNode] strips off the synthetic wrapper.
const selectPrefix = "SELECT "

// rawDeparser is [pg_query.Deparse]'s signature. We inject it so a test can drive
// the deparse-failure path, which a real parsed node never reaches.
type rawDeparser func(*pg_query.ParseResult) (string, error)

// deparseNode renders a single value-expression node as SQL text using
// PostgreSQL's own deparser — correct for every node kind, so the formatter never
// has to hand-render an expression. Keyword case is normalized later, in one pass
// over the whole statement.
func deparseNode(node *pg_query.Node) (string, error) {
	return deparseNodeWith(pg_query.Deparse, node)
}

// deparseNodeWith is [deparseNode] with its deparser injected. It wraps node in a
// throwaway SELECT, deparses that, and strips the leading "SELECT ".
func deparseNodeWith(deparse rawDeparser, node *pg_query.Node) (string, error) {
	out, err := deparse(selectWrap(node))
	if err != nil {
		return "", ErrDeparseLeaf.With(err)
	}
	return strings.TrimPrefix(out, selectPrefix), nil
}

// selectWrap builds the synthetic `SELECT <node>` parse tree that [deparseNodeWith]
// renders to get a node's text in isolation.
func selectWrap(node *pg_query.Node) *pg_query.ParseResult {
	target := &pg_query.Node{Node: &pg_query.Node_ResTarget{ResTarget: &pg_query.ResTarget{Val: node}}}
	sel := &pg_query.SelectStmt{TargetList: []*pg_query.Node{target}}
	stmt := &pg_query.RawStmt{Stmt: &pg_query.Node{Node: &pg_query.Node_SelectStmt{SelectStmt: sel}}}
	return &pg_query.ParseResult{Stmts: []*pg_query.RawStmt{stmt}}
}

// lowerFunc is [sql.LowerKeywords]' signature. We inject it so a test can drive
// the lowering-failure path, which deparse output (always valid SQL) never
// reaches in practice.
type lowerFunc func(string) (sql.SQL, error)

// canonicalStatement renders one statement through PostgreSQL's deparser and
// lowercases its keywords — correct for every statement kind, the fallback the
// gate leans on when a house renderer is missing or unfaithful. It reports false
// when the statement can't be deparsed, leaving the gate to fall back further.
func canonicalStatement(stmt *pg_query.RawStmt) (string, bool) {
	return canonicalStatementWith(pg_query.Deparse, sql.LowerKeywords, stmt)
}

// canonicalStatementWith is [canonicalStatement] with its deparser and lowercaser
// injected.
func canonicalStatementWith(deparse rawDeparser, lower lowerFunc, stmt *pg_query.RawStmt) (string, bool) {
	deparsed, err := deparse(&pg_query.ParseResult{Stmts: []*pg_query.RawStmt{stmt}})
	if err != nil {
		return "", false
	}
	lowered, err := lower(deparsed)
	if err != nil {
		return "", false
	}
	return string(lowered), true
}
