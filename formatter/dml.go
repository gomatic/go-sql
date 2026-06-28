package formatter

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// formatInsert renders an INSERT statement.
func (f Formatter) formatInsert(stmt *pg_query.InsertStmt) string {
	var b builder
	b.write("insert into ")
	b.write(formatRangeVar(stmt.Relation))
	b.write(insertColumns(stmt.Cols))
	f.writeInsertSource(&b, stmt.SelectStmt)
	f.writeOnConflict(&b, stmt.OnConflictClause)
	b.write(";")
	return b.String()
}

// insertColumns renders the parenthesized target-column list, or "" when there
// aren't any columns.
func insertColumns(cols []*pg_query.Node) string {
	if len(cols) == 0 {
		return ""
	}
	names := make([]string, 0, len(cols))
	for _, col := range cols {
		if resTarget := col.GetResTarget(); resTarget != nil {
			names = append(names, resTarget.Name)
		}
	}
	return " (" + strings.Join(names, ", ") + ")"
}

// writeInsertSource renders the SELECT (or VALUES) that feeds the INSERT.
func (f Formatter) writeInsertSource(b *builder, source *pg_query.Node) {
	if source == nil {
		return
	}
	b.write("\n")
	if selectStmt := source.GetSelectStmt(); selectStmt != nil {
		b.write(f.formatSelect(selectStmt, 0))
	}
}

// writeOnConflict renders the ON CONFLICT clause when there is one.
func (f Formatter) writeOnConflict(b *builder, clause *pg_query.OnConflictClause) {
	if clause == nil {
		return
	}
	b.write("\n")
	b.write(pad(f.indentSize))
	b.write(f.formatOnConflict(clause))
}

// formatOnConflict renders an ON CONFLICT target and its action.
func (f Formatter) formatOnConflict(clause *pg_query.OnConflictClause) string {
	return "on conflict" + f.conflictTarget(clause.Infer) + f.conflictAction(clause)
}

// conflictTarget renders an ON CONFLICT clause's optional inference target.
func (f Formatter) conflictTarget(infer *pg_query.InferClause) string {
	if infer == nil || len(infer.IndexElems) == 0 {
		return ""
	}
	parts := make([]string, 0, len(infer.IndexElems))
	for _, elem := range infer.IndexElems {
		if indexElem := elem.GetIndexElem(); indexElem != nil {
			parts = append(parts, f.formatIndexElem(indexElem))
		}
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// conflictAction renders an ON CONFLICT's DO NOTHING / DO UPDATE action.
func (f Formatter) conflictAction(clause *pg_query.OnConflictClause) string {
	switch clause.Action {
	case pg_query.OnConflictAction_ONCONFLICT_NOTHING:
		return " do nothing"
	case pg_query.OnConflictAction_ONCONFLICT_UPDATE:
		return " do update set " + f.assignmentList(clause.TargetList, ", ")
	default:
		return ""
	}
}

// formatUpdate renders an UPDATE statement.
func (f Formatter) formatUpdate(stmt *pg_query.UpdateStmt) string {
	var b builder
	b.write("update ")
	b.write(formatRangeVar(stmt.Relation))
	if len(stmt.TargetList) > 0 {
		b.write("\n")
		b.write(pad(f.indentSize))
		b.write("set ")
		b.write(f.assignmentList(stmt.TargetList, ",\n"+pad(f.indentSize+2)))
	}
	f.writeRowFilter(&b, stmt.WhereClause)
	b.write(";")
	return b.String()
}

// formatDelete renders a DELETE statement.
func (f Formatter) formatDelete(stmt *pg_query.DeleteStmt) string {
	var b builder
	b.write("delete from ")
	b.write(formatRangeVar(stmt.Relation))
	f.writeRowFilter(&b, stmt.WhereClause)
	b.write(";")
	return b.String()
}

// writeRowFilter renders the WHERE clause that UPDATE and DELETE both use.
func (f Formatter) writeRowFilter(b *builder, where *pg_query.Node) {
	if where == nil {
		return
	}
	b.write("\n")
	b.write(whereKw)
	b.write(f.formatExpr(where, f.indentSize))
}

// assignmentList renders "col = expr" pairs, joined by sep.
func (f Formatter) assignmentList(targets []*pg_query.Node, sep string) string {
	parts := make([]string, 0, len(targets))
	for _, target := range targets {
		if resTarget := target.GetResTarget(); resTarget != nil {
			parts = append(parts, resTarget.Name+" = "+f.formatExpr(resTarget.Val, 0))
		}
	}
	return strings.Join(parts, sep)
}
