package formatter

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// formatSelect renders a SELECT statement — with its optional WITH, FROM, WHERE,
// ORDER BY, and set-operation clauses — at the given base indentation.
func (f Formatter) formatSelect(stmt *pg_query.SelectStmt, baseIndent int) string {
	var b builder
	f.writeWithClause(&b, stmt, baseIndent)
	f.writeSelectList(&b, stmt, baseIndent)
	f.writeFromClause(&b, stmt, baseIndent)
	f.writeWhereClause(&b, stmt, baseIndent)
	f.writeOrderByClause(&b, stmt, baseIndent)
	f.writeSetOp(&b, stmt, baseIndent)
	return b.String()
}

// writeWithClause renders the leading WITH (CTE) block when there is one.
func (f Formatter) writeWithClause(b *builder, stmt *pg_query.SelectStmt, baseIndent int) {
	ctes := stmt.WithClause.GetCtes()
	if len(ctes) == 0 {
		return
	}

	indent := pad(baseIndent)
	b.write(indent)
	b.write("with ")
	for i, cte := range ctes {
		if i > 0 {
			b.write(",\n")
			b.write(indent)
			b.write("     ")
		}
		f.formatCTE(cte, b, indent, baseIndent)
	}
	b.write("\n")
	b.write(indent)
}

// writeSelectList renders the SELECT keyword and its target list.
func (f Formatter) writeSelectList(b *builder, stmt *pg_query.SelectStmt, baseIndent int) {
	b.write(pad(baseIndent))
	b.write("select ")
	for i, target := range stmt.TargetList {
		if i > 0 {
			b.write("\n")
			b.write(pad(baseIndent + f.indentSize + 1))
			b.write(", ")
		}
		b.write(f.formatResTarget(target.GetResTarget()))
	}
}

// writeFromClause renders the FROM clause when there is one.
func (f Formatter) writeFromClause(b *builder, stmt *pg_query.SelectStmt, baseIndent int) {
	if len(stmt.FromClause) == 0 {
		return
	}

	b.write("\n")
	b.write(pad(baseIndent))
	b.write("from ")
	for i, from := range stmt.FromClause {
		if i > 0 {
			b.write(", ")
		}
		b.write(f.formatFromExpr(from))
	}
}

// writeWhereClause renders the WHERE clause when there is one.
func (f Formatter) writeWhereClause(b *builder, stmt *pg_query.SelectStmt, baseIndent int) {
	if stmt.WhereClause == nil {
		return
	}

	b.write("\n")
	b.write(pad(baseIndent))
	b.write(whereKw)
	b.write(f.formatExpr(stmt.WhereClause, baseIndent+f.indentSize))
}

// writeOrderByClause renders the ORDER BY clause when there is one.
func (f Formatter) writeOrderByClause(b *builder, stmt *pg_query.SelectStmt, baseIndent int) {
	if len(stmt.SortClause) == 0 {
		return
	}

	b.write("\n")
	b.write(pad(baseIndent))
	b.write("order by ")
	for i, sort := range stmt.SortClause {
		if i > 0 {
			b.write(", ")
		}
		b.write(f.formatSortBy(sort.GetSortBy()))
	}
}

// writeSetOp renders a trailing UNION/INTERSECT/EXCEPT and its right operand.
func (f Formatter) writeSetOp(b *builder, stmt *pg_query.SelectStmt, baseIndent int) {
	if stmt.Op == pg_query.SetOperation_SETOP_NONE || stmt.Rarg == nil {
		return
	}

	b.write("\n")
	b.write(pad(baseIndent))
	b.write(setOpKeyword(stmt.Op))
	if stmt.All {
		b.write(" all")
	}
	b.write("\n")
	b.write(f.formatSelect(stmt.Rarg, baseIndent))
}

// setOpKeyword turns a set operation into its keyword.
func setOpKeyword(op pg_query.SetOperation) string {
	switch op {
	case pg_query.SetOperation_SETOP_UNION:
		return "union"
	case pg_query.SetOperation_SETOP_INTERSECT:
		return "intersect"
	case pg_query.SetOperation_SETOP_EXCEPT:
		return "except"
	default:
		return ""
	}
}

// formatCTE renders a single common table expression, fencing its body with
// formatter directives so the nested query stays verbatim.
func (f Formatter) formatCTE(cte *pg_query.Node, b *builder, indent string, baseIndent int) {
	cteExpr := cte.GetCommonTableExpr()
	if cteExpr == nil {
		return
	}

	b.write(cteExpr.GetCtename())
	b.write(" as (\n")
	b.write(indent)
	b.write("  --@formatter:off\n")
	if cteQuery := cteExpr.GetCtequery(); cteQuery != nil {
		b.write(f.formatCTEQuery(cteQuery, baseIndent))
	}
	b.write("\n")
	b.write(indent)
	b.write("  --@formatter:on\n")
	b.write(indent)
	b.write(")")
}

// formatCTEQuery renders the SELECT body of a CTE, one step further in.
func (f Formatter) formatCTEQuery(cteQuery *pg_query.Node, baseIndent int) string {
	if selectStmt := cteQuery.GetSelectStmt(); selectStmt != nil {
		return f.formatSelect(selectStmt, baseIndent+f.indentSize)
	}
	return ""
}

// formatFromExpr renders a single FROM-clause entry.
func (f Formatter) formatFromExpr(from *pg_query.Node) string {
	switch node := from.Node.(type) {
	case *pg_query.Node_RangeVar:
		return rangeVarWithAlias(node.RangeVar)
	case *pg_query.Node_JoinExpr:
		return f.formatJoinExpr(node.JoinExpr)
	default:
		return "/* complex from expression */"
	}
}

// rangeVarWithAlias renders a relation reference with its optional alias.
func rangeVarWithAlias(rv *pg_query.RangeVar) string {
	result := formatRangeVar(rv)
	if rv.Alias != nil {
		return result + " as " + rv.Alias.Aliasname
	}
	return result
}

// formatJoinExpr renders a join between two FROM entries.
func (f Formatter) formatJoinExpr(join *pg_query.JoinExpr) string {
	var b builder
	b.write(f.formatFromExpr(join.Larg))
	b.write("\n")
	b.write(joinKeyword(join.Jointype))
	b.write(f.formatFromExpr(join.Rarg))
	if join.Quals != nil {
		b.write("\n")
		b.write(pad(f.indentSize))
		b.write("on ")
		b.write(f.formatExpr(join.Quals, f.indentSize))
	}
	return b.String()
}

// joinKeyword turns a join type into its leading keyword.
func joinKeyword(jt pg_query.JoinType) string {
	switch jt {
	case pg_query.JoinType_JOIN_LEFT:
		return "left join "
	case pg_query.JoinType_JOIN_RIGHT:
		return "right join "
	case pg_query.JoinType_JOIN_FULL:
		return "full join "
	default:
		return "join "
	}
}
