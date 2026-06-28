package formatter

import (
	"strconv"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// formatExpr renders a value expression. Node kinds we don't handle come out as
// a placeholder comment. baseIndent is where the continuation of a boolean chain
// lines up.
func (f Formatter) formatExpr(expr *pg_query.Node, baseIndent int) string {
	if expr == nil {
		return ""
	}

	switch node := expr.Node.(type) {
	case *pg_query.Node_ColumnRef:
		return formatColumnRef(node.ColumnRef)
	case *pg_query.Node_AConst:
		return formatAConst(node.AConst)
	case *pg_query.Node_BoolExpr:
		return f.formatBoolExpr(node.BoolExpr, baseIndent)
	case *pg_query.Node_FuncCall:
		return f.formatFuncCall(node.FuncCall)
	default:
		return "/* complex expression */"
	}
}

// formatColumnRef renders a column reference, qualified or not.
func formatColumnRef(col *pg_query.ColumnRef) string {
	return joinStringNodes(col.Fields)
}

// formatAConst renders a literal constant. The getters return nil unless their
// oneof arm is set, so a null marker or an empty constant falls through to null.
func formatAConst(c *pg_query.A_Const) string {
	switch {
	case c.Isnull:
		return nullKw
	case c.GetSval() != nil:
		return "'" + c.GetSval().Sval + "'"
	case c.GetIval() != nil:
		return strconv.FormatInt(int64(c.GetIval().Ival), 10)
	case c.GetFval() != nil:
		return c.GetFval().Fval
	case c.GetBoolval() != nil:
		return boolText(c.GetBoolval().Boolval)
	case c.GetBsval() != nil:
		return c.GetBsval().Bsval
	default:
		return nullKw
	}
}

// boolText renders a boolean as its SQL keyword.
func boolText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// formatBoolExpr renders an AND/OR chain, one operand to a line.
func (f Formatter) formatBoolExpr(expr *pg_query.BoolExpr, baseIndent int) string {
	var b builder
	for i, arg := range expr.Args {
		if i > 0 {
			b.write("\n")
			b.write(pad(baseIndent))
			b.write(boolOpKeyword(expr.Boolop))
		}
		b.write(f.formatExpr(arg, baseIndent))
	}
	return b.String()
}

// boolOpKeyword turns a boolean operator into its leading keyword.
func boolOpKeyword(op pg_query.BoolExprType) string {
	if op == pg_query.BoolExprType_OR_EXPR {
		return "or "
	}
	return "and "
}

// formatFuncCall renders a function call and its argument list.
func (f Formatter) formatFuncCall(call *pg_query.FuncCall) string {
	var b builder
	b.write(joinStringNodes(call.Funcname))
	b.write("(")
	for i, arg := range call.Args {
		if i > 0 {
			b.write(", ")
		}
		b.write(f.formatExpr(arg, 0))
	}
	b.write(")")
	return b.String()
}

// formatSortBy renders an ORDER BY term with its optional direction.
func (f Formatter) formatSortBy(sort *pg_query.SortBy) string {
	return f.formatExpr(sort.Node, 0) + sortDirection(sort.SortbyDir)
}

// sortDirection turns a sort direction into its trailing keyword.
func sortDirection(dir pg_query.SortByDir) string {
	switch dir {
	case pg_query.SortByDir_SORTBY_DESC:
		return " desc"
	case pg_query.SortByDir_SORTBY_ASC:
		return " asc"
	default:
		return ""
	}
}

// formatResTarget renders a result target with its optional alias.
func (f Formatter) formatResTarget(target *pg_query.ResTarget) string {
	result := f.formatExpr(target.Val, 0)
	if target.Name != "" {
		return result + " as " + target.Name
	}
	return result
}

// formatRangeVar renders a relation name, schema-qualified or not.
func formatRangeVar(rv *pg_query.RangeVar) string {
	parts := make([]string, 0, 2)
	if rv.Schemaname != "" {
		parts = append(parts, rv.Schemaname)
	}
	if rv.Relname != "" {
		parts = append(parts, rv.Relname)
	}
	return strings.Join(parts, ".")
}
