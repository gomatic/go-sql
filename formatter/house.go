package formatter

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/gomatic/go-sql"
)

// riverWidth is the column the clause keywords right-align to: every keyword is
// padded into this field so the clause contents line up one space past it (the
// "river"). It's the width of the widest top-level keyword the house style emits,
// "select".
const riverWidth = 6

// leafDeparse and houseLower are the leaf renderer and keyword lowercaser the
// house path uses. They're indirected so a test can drive the failure branches,
// which a real parsed statement never reaches: PostgreSQL's deparser renders every
// value node, and the lowercaser scans deparser output that is always valid SQL.
var (
	leafDeparse = deparseNode
	houseLower  = sql.LowerKeywords
)

// houseStatement renders a statement in the project's leading-comma river style,
// or reports false when the statement's shape isn't one the house renderer covers
// yet — in which case the gate falls back to the canonical deparse. Keyword case
// is normalized in one final pass, so leaf text from PostgreSQL's deparser (which
// uppercases keywords) comes out lowercase too.
func houseStatement(stmt *pg_query.RawStmt) (string, bool) {
	sel := stmt.Stmt.GetSelectStmt()
	if sel == nil {
		return "", false
	}
	lines, ok := selectLines(sel)
	if !ok {
		return "", false
	}
	lowered, err := houseLower(strings.Join(lines, "\n"))
	if err != nil {
		return "", false
	}
	return string(lowered), true
}

// selectLines builds the house-style lines for a plain SELECT, or false when the
// select carries a clause the renderer doesn't lay out yet.
func selectLines(sel *pg_query.SelectStmt) ([]string, bool) {
	if !isPlainSelect(sel) {
		return nil, false
	}
	targets, ok := targetTexts(sel.TargetList)
	if !ok {
		return nil, false
	}
	from, ok := fromTexts(sel.FromClause)
	if !ok {
		return nil, false
	}
	where, ok := whereLines(sel.WhereClause)
	if !ok {
		return nil, false
	}
	lines := leadingComma(clauseKeyword("select"), targets)
	lines = append(lines, leadingComma(clauseKeyword("from"), from)...)
	return append(lines, where...), true
}

// isPlainSelect reports whether sel is the simple SELECT shape the house renderer
// covers: a target list, optional FROM, optional WHERE, and none of the clauses
// (CTEs, set operations, DISTINCT, GROUP/HAVING/WINDOW, ORDER BY, LIMIT/OFFSET,
// VALUES, locking) that still fall back to the canonical deparse.
func isPlainSelect(sel *pg_query.SelectStmt) bool {
	return len(sel.TargetList) > 0 && !hasComplexClause(sel)
}

// hasComplexClause reports whether sel carries any clause beyond target/FROM/WHERE.
func hasComplexClause(sel *pg_query.SelectStmt) bool {
	return len(sel.WithClause.GetCtes()) > 0 ||
		sel.Op != pg_query.SetOperation_SETOP_NONE ||
		len(sel.DistinctClause) > 0 ||
		len(sel.GroupClause) > 0 ||
		sel.HavingClause != nil ||
		len(sel.WindowClause) > 0 ||
		len(sel.SortClause) > 0 ||
		sel.LimitCount != nil ||
		sel.LimitOffset != nil ||
		len(sel.ValuesLists) > 0 ||
		len(sel.LockingClause) > 0
}

// targetTexts renders each select-list target as "<expr>" or "<expr> as <alias>",
// or false when a target isn't a result target or its value won't deparse.
func targetTexts(targets []*pg_query.Node) ([]string, bool) {
	texts := make([]string, 0, len(targets))
	for _, node := range targets {
		target := node.GetResTarget()
		if target == nil {
			return nil, false
		}
		val, err := leafDeparse(target.Val)
		if err != nil {
			return nil, false
		}
		if target.Name != "" {
			val += " as " + target.Name
		}
		texts = append(texts, val)
	}
	return texts, true
}

// fromTexts renders each FROM item, covering only plain table references for now;
// a join or subquery yields false so the whole statement falls back.
func fromTexts(items []*pg_query.Node) ([]string, bool) {
	texts := make([]string, 0, len(items))
	for _, item := range items {
		rv := item.GetRangeVar()
		if rv == nil {
			return nil, false
		}
		texts = append(texts, rangeVarText(rv))
	}
	return texts, true
}

// rangeVarText renders a table reference, schema-qualified and aliased as written.
func rangeVarText(rv *pg_query.RangeVar) string {
	name := rv.Relname
	if rv.Schemaname != "" {
		name = rv.Schemaname + "." + name
	}
	if rv.Alias.GetAliasname() != "" {
		name += " as " + rv.Alias.GetAliasname()
	}
	return name
}

// whereLines renders the WHERE clause: nothing when absent, a top-level AND chain
// split one condition per line (`where` then `and`s), or a single `where` line.
func whereLines(where *pg_query.Node) ([]string, bool) {
	if where == nil {
		return nil, true
	}
	if conditions, ok := andConditions(where); ok {
		return clauseLines("where", "and", conditions)
	}
	text, err := leafDeparse(where)
	if err != nil {
		return nil, false
	}
	return []string{riverLine("where", text)}, true
}

// andConditions returns the operands of a top-level AND, or false for any other
// expression (a single condition, an OR chain, …) which renders on one line.
func andConditions(where *pg_query.Node) ([]*pg_query.Node, bool) {
	expr := where.GetBoolExpr()
	if expr == nil || expr.Boolop != pg_query.BoolExprType_AND_EXPR {
		return nil, false
	}
	return expr.Args, true
}

// clauseLines renders conditions with first on a `first` line and the rest on
// `rest` lines, or false if any condition won't deparse.
func clauseLines(first, rest string, conditions []*pg_query.Node) ([]string, bool) {
	lines := make([]string, 0, len(conditions))
	for i, condition := range conditions {
		text, err := leafDeparse(condition)
		if err != nil {
			return nil, false
		}
		keyword := rest
		if i == 0 {
			keyword = first
		}
		lines = append(lines, riverLine(keyword, text))
	}
	return lines, true
}

// clauseKeyword is the SQL clause keyword that leads a river-aligned block (select, from, …).
type clauseKeyword string

// leadingComma renders items under a clause keyword: the first on the keyword's
// line, each subsequent one on its own line led by a comma aligned to the river.
func leadingComma(keyword clauseKeyword, items []string) []string {
	if len(items) == 0 {
		return nil
	}
	lines := []string{riverLine(string(keyword), items[0])}
	for _, item := range items[1:] {
		lines = append(lines, riverLine(",", item))
	}
	return lines
}

// riverLine right-aligns keyword into the river field and follows it with content.
func riverLine(keyword, content string) string {
	return fmt.Sprintf("%*s %s", riverWidth, keyword, content)
}
