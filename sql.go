// Package sql parses SQL text into PostgreSQL's syntax tree, puts it in a
// canonical order, and deparses it back to SQL. It's a thin, allocation-light
// layer over [pg_query] that adds sentinel-error wrapping ([ErrParse],
// [ErrDeparse]) plus the column-ordering normalization we use for canonical
// comparison.
//
// The JSON normalization ([ToJSON]) lives right here too; SQL formatting, AST
// diffing, and text normalization live over in the formatter, compare, and
// normalize subpackages.
//
// [pg_query]: https://github.com/pganalyze/pg_query_go
package sql

import (
	"sort"
	"strconv"
	"strings"

	errs "github.com/gomatic/go-error"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// Sentinel errors this package can return. Match them with [errors.Is], not by
// string.
const (
	// ErrParse means we couldn't parse the SQL text into an AST.
	ErrParse errs.Const = "parse SQL"
	// ErrDeparse means we couldn't turn an AST back into SQL.
	ErrDeparse errs.Const = "deparse SQL"
)

// SQL is raw SQL text.
type SQL string

// Parse turns SQL text into PostgreSQL's parse-result AST. If parsing fails, you
// get back an error wrapped in [ErrParse].
func Parse(sql SQL) (*pg_query.ParseResult, error) {
	tree, err := pg_query.Parse(string(sql))
	if err != nil {
		return nil, ErrParse.With(err)
	}
	return tree, nil
}

// Deparse turns an AST back into SQL text. If it fails, you get back an error
// wrapped in [ErrDeparse].
func Deparse(tree *pg_query.ParseResult) (SQL, error) {
	result, err := pg_query.Deparse(tree)
	if err != nil {
		return "", ErrDeparse.With(err)
	}
	return SQL(result), nil
}

// SortColumnLists puts the column lists of INSERT … SELECT and simple SELECT
// target lists into a canonical order, in place, so two trees that mean the same
// thing compare equal.
//
// It edits tree in place, so it's not safe to run concurrently on the same
// *pg_query.ParseResult — serialize the calls, or work on separate trees.
func SortColumnLists(tree *pg_query.ParseResult) {
	if tree == nil {
		return
	}
	traverseAndSort(treeReflect(tree))
}

type pair struct {
	col    *pg_query.Node
	target *pg_query.Node
	name   string
}

func sortInsertStmt(stmt *pg_query.InsertStmt) {
	selStmt := insertSelect(stmt)
	if selStmt == nil || len(stmt.Cols) != len(selStmt.TargetList) {
		return
	}

	pairs := make([]pair, len(stmt.Cols))
	for i, col := range stmt.Cols {
		pairs[i] = pair{col: col, name: resTargetName(col), target: selStmt.TargetList[i]}
	}

	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].name < pairs[j].name })

	for i, p := range pairs {
		stmt.Cols[i] = p.col
		selStmt.TargetList[i] = p.target
	}
}

// insertSelect hands back the SELECT feeding an INSERT, or nil if the INSERT has
// no column list or no SELECT source.
func insertSelect(stmt *pg_query.InsertStmt) *pg_query.SelectStmt {
	if len(stmt.Cols) == 0 || stmt.SelectStmt == nil {
		return nil
	}
	return stmt.SelectStmt.GetSelectStmt()
}

// resTargetName pulls the column name out of a ResTarget node, or "" if it
// doesn't have one.
func resTargetName(col *pg_query.Node) string {
	if rt := col.GetResTarget(); rt != nil {
		return rt.Name
	}
	return ""
}

func sortSelectStmt(stmt *pg_query.SelectStmt) {
	if !isSimpleSelect(stmt) {
		return
	}
	sort.SliceStable(stmt.TargetList, func(i, j int) bool {
		return getTargetName(stmt.TargetList[i]) < getTargetName(stmt.TargetList[j])
	})
}

func isSimpleSelect(stmt *pg_query.SelectStmt) bool {
	return len(stmt.DistinctClause) == 0 &&
		stmt.IntoClause == nil &&
		len(stmt.GroupClause) == 0 &&
		stmt.HavingClause == nil &&
		len(stmt.WindowClause) == 0 &&
		len(stmt.ValuesLists) == 0 &&
		len(stmt.LockingClause) == 0
}

func getTargetName(target *pg_query.Node) string {
	rt := target.GetResTarget()
	if rt == nil {
		return ""
	}
	if rt.Name != "" {
		return rt.Name
	}
	return getNodeName(rt.Val)
}

// getNodeName renders the leading identifier-ish name of an expression node. It
// hands off to a small handler depending on the node's shape.
func getNodeName(node *pg_query.Node) string {
	if node == nil {
		return ""
	}
	switch {
	case node.GetColumnRef() != nil:
		return columnRefName(node.GetColumnRef())
	case node.GetAConst() != nil:
		return constName(node.GetAConst())
	case node.GetFuncCall() != nil:
		return funcCallName(node.GetFuncCall())
	case node.GetAExpr() != nil:
		return aExprName(node.GetAExpr())
	default:
		return ""
	}
}

func columnRefName(ref *pg_query.ColumnRef) string {
	parts := make([]string, 0, len(ref.Fields))
	for _, f := range ref.Fields {
		switch {
		case f.GetString_() != nil:
			parts = append(parts, f.GetString_().Sval)
		case f.GetAStar() != nil:
			parts = append(parts, "*")
		}
	}
	return strings.Join(parts, ".")
}

func constName(c *pg_query.A_Const) string {
	switch {
	case c.GetSval() != nil:
		return c.GetSval().Sval
	case c.GetIval() != nil:
		return strconv.FormatInt(int64(c.GetIval().Ival), 10)
	case c.GetFval() != nil:
		return c.GetFval().Fval
	case c.GetBsval() != nil:
		return c.GetBsval().Bsval
	case c.GetBoolval() != nil:
		return strconv.FormatBool(c.GetBoolval().Boolval)
	default:
		return ""
	}
}

func funcCallName(f *pg_query.FuncCall) string {
	parts := make([]string, 0, len(f.Funcname))
	for _, n := range f.Funcname {
		if s := n.GetString_(); s != nil {
			parts = append(parts, s.Sval)
		}
	}
	return strings.Join(parts, ".")
}

func aExprName(expr *pg_query.A_Expr) string {
	op := ""
	if len(expr.Name) > 0 {
		if s := expr.Name[0].GetString_(); s != nil {
			op = s.Sval
		}
	}
	return getNodeName(expr.Lexpr) + " " + op + " " + getNodeName(expr.Rexpr)
}
