package formatter

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// formatCreateSchema renders a CREATE SCHEMA statement.
func formatCreateSchema(stmt *pg_query.CreateSchemaStmt) string {
	var b builder
	b.write("create schema")
	if stmt.IfNotExists {
		b.write(ifNotExists)
	}
	if stmt.Schemaname != "" {
		b.write(" ")
		b.write(stmt.Schemaname)
	}
	b.write(";")
	return b.String()
}

// formatCreateView renders a CREATE [OR REPLACE] VIEW statement.
func (f Formatter) formatCreateView(stmt *pg_query.ViewStmt) string {
	var b builder
	b.write(createKw)
	if stmt.Replace {
		b.write(orReplace)
	}
	b.write(" view ")
	if stmt.View != nil {
		b.write(formatRangeVar(stmt.View))
	}
	b.write(" as\n")
	if stmt.Query != nil {
		if selectStmt, ok := stmt.Query.Node.(*pg_query.Node_SelectStmt); ok {
			b.write(f.formatSelect(selectStmt.SelectStmt, 0))
		}
	}
	b.write(";")
	return b.String()
}

// formatCreateTable renders a CREATE TABLE statement.
func (f Formatter) formatCreateTable(stmt *pg_query.CreateStmt) string {
	var b builder
	b.write("create table")
	if stmt.IfNotExists {
		b.write(ifNotExists)
	}
	b.write(" ")
	b.write(formatRangeVar(stmt.Relation))
	b.write(" (")
	if len(stmt.TableElts) > 0 {
		b.write("\n")
		for i, elt := range stmt.TableElts {
			if i > 0 {
				b.write(",\n")
			}
			b.write(pad(f.indentSize))
			b.write(f.formatTableElement(elt))
		}
		b.write("\n")
	}
	b.write(");")
	return b.String()
}

// formatTableElement renders one column definition or table constraint.
func (f Formatter) formatTableElement(elt *pg_query.Node) string {
	switch node := elt.Node.(type) {
	case *pg_query.Node_ColumnDef:
		return f.formatColumnDef(node.ColumnDef)
	case *pg_query.Node_Constraint:
		return f.formatConstraint(node.Constraint)
	default:
		return "/* unsupported table element */"
	}
}

// formatColumnDef renders a column name, its type, and any inline constraints.
func (f Formatter) formatColumnDef(col *pg_query.ColumnDef) string {
	var b builder
	b.write(col.Colname)
	b.write(" ")
	b.write(f.formatTypeName(col.TypeName))
	for _, constraint := range col.Constraints {
		if cons := constraint.GetConstraint(); cons != nil {
			b.write(" ")
			b.write(f.formatColumnConstraint(cons))
		}
	}
	return b.String()
}

// formatTypeName renders a type name plus any modifiers, falling back to text
// when there's no name to render.
func (f Formatter) formatTypeName(typeName *pg_query.TypeName) string {
	if typeName == nil || len(typeName.Names) == 0 {
		return "text"
	}

	result := joinStringNodes(typeName.Names)
	if len(typeName.Typmods) == 0 {
		return result
	}

	mods := make([]string, 0, len(typeName.Typmods))
	for _, mod := range typeName.Typmods {
		mods = append(mods, f.formatExpr(mod, 0))
	}
	return result + "(" + strings.Join(mods, ", ") + ")"
}

// formatConstraint renders a table constraint, named or anonymous.
func (f Formatter) formatConstraint(constraint *pg_query.Constraint) string {
	var b builder
	if constraint.Conname != "" {
		b.write("constraint ")
		b.write(constraint.Conname)
		b.write(" ")
	}
	b.write(f.constraintBody(constraint))
	b.write(constraintKeys(constraint.Keys))
	return b.String()
}

// constraintBody renders the type-specific part of a constraint.
func (f Formatter) constraintBody(constraint *pg_query.Constraint) string {
	switch constraint.Contype {
	case pg_query.ConstrType_CONSTR_PRIMARY:
		return "primary key"
	case pg_query.ConstrType_CONSTR_UNIQUE:
		return "unique"
	case pg_query.ConstrType_CONSTR_FOREIGN:
		return f.foreignKeyBody(constraint)
	case pg_query.ConstrType_CONSTR_CHECK:
		return "check (" + f.formatExpr(constraint.RawExpr, 0) + ")"
	default:
		return "/* unknown constraint */"
	}
}

// foreignKeyBody renders a FOREIGN KEY body along with the table it points at.
func (f Formatter) foreignKeyBody(constraint *pg_query.Constraint) string {
	if constraint.Pktable == nil {
		return "foreign key"
	}
	return "foreign key references " + formatRangeVar(constraint.Pktable)
}

// constraintKeys renders a parenthesized, comma-separated column-key list, or
// "" when there aren't any keys.
func constraintKeys(keys []*pg_query.Node) string {
	if len(keys) == 0 {
		return ""
	}
	return " (" + strings.Join(stringNodeValues(keys), ", ") + ")"
}

// formatColumnConstraint renders an inline column constraint.
func (f Formatter) formatColumnConstraint(constraint *pg_query.Constraint) string {
	switch constraint.Contype {
	case pg_query.ConstrType_CONSTR_NOTNULL:
		return "not null"
	case pg_query.ConstrType_CONSTR_NULL:
		return nullKw
	case pg_query.ConstrType_CONSTR_DEFAULT:
		return "default " + f.formatExpr(constraint.RawExpr, 0)
	case pg_query.ConstrType_CONSTR_PRIMARY:
		return "primary key"
	case pg_query.ConstrType_CONSTR_UNIQUE:
		return "unique"
	default:
		return "/* unknown column constraint */"
	}
}

// formatCreateFunction renders a CREATE [OR REPLACE] FUNCTION statement.
func (f Formatter) formatCreateFunction(stmt *pg_query.CreateFunctionStmt) string {
	var b builder
	b.write(createKw)
	if stmt.Replace {
		b.write(orReplace)
	}
	b.write(" function ")
	b.write(joinStringNodes(stmt.Funcname))
	b.write("(")
	b.write(f.functionParams(stmt.Parameters))
	b.write(")")
	f.writeFunctionReturns(&b, stmt.ReturnType)
	f.writeFunctionOptions(&b, stmt.Options)
	b.write(";")
	return b.String()
}

// functionParams renders a comma-separated function parameter list.
func (f Formatter) functionParams(params []*pg_query.Node) string {
	parts := make([]string, 0, len(params))
	for _, param := range params {
		parts = append(parts, f.formatFunctionParameter(param))
	}
	return strings.Join(parts, ", ")
}

// writeFunctionReturns renders the RETURNS clause when there is one.
func (f Formatter) writeFunctionReturns(b *builder, returnType *pg_query.TypeName) {
	if returnType == nil {
		return
	}
	b.write("\n")
	b.write(pad(f.indentSize))
	b.write("returns ")
	b.write(f.formatTypeName(returnType))
}

// writeFunctionOptions renders the function's option clauses.
func (f Formatter) writeFunctionOptions(b *builder, options []*pg_query.Node) {
	for _, option := range options {
		if defElem := option.GetDefElem(); defElem != nil {
			b.write("\n")
			b.write(pad(f.indentSize))
			b.write(formatDefElem(defElem))
		}
	}
}

// formatFunctionParameter renders a single function parameter.
func (f Formatter) formatFunctionParameter(param *pg_query.Node) string {
	fp := param.GetFunctionParameter()
	if fp == nil {
		return "/* unknown parameter */"
	}

	var b builder
	b.write(paramMode(fp.Mode))
	if fp.Name != "" {
		b.write(fp.Name)
		b.write(" ")
	}
	if fp.ArgType != nil {
		b.write(f.formatTypeName(fp.ArgType))
	}
	if fp.Defexpr != nil {
		b.write(" default ")
		b.write(f.formatExpr(fp.Defexpr, 0))
	}
	return b.String()
}

// paramMode renders a parameter's leading mode keyword.
func paramMode(mode pg_query.FunctionParameterMode) string {
	switch mode {
	case pg_query.FunctionParameterMode_FUNC_PARAM_OUT:
		return "out "
	case pg_query.FunctionParameterMode_FUNC_PARAM_INOUT:
		return "inout "
	default:
		return ""
	}
}

// The DefElem names we know about, lowercased.
const (
	defAs         = "as"
	defLanguage   = "language"
	defSecurity   = "security"
	defVolatility = "volatility"
)

// defElemFormat says how a recognized DefElem wraps its argument.
type defElemFormat struct {
	prefix string
	suffix string
}

// defElemFormats maps a lowercased DefElem name to the text that goes around it.
var defElemFormats = map[string]defElemFormat{
	defAs:         {prefix: "as\n$$\n", suffix: "\n$$"},
	defLanguage:   {prefix: "language ", suffix: ""},
	defSecurity:   {prefix: "security ", suffix: ""},
	defVolatility: {prefix: "", suffix: ""},
}

// formatDefElem renders a function definition element, falling back to the bare
// name when its argument is missing or we don't recognize the name.
func formatDefElem(elem *pg_query.DefElem) string {
	arg, ok := defElemArg(elem)
	spec, known := defElemFormats[strings.ToLower(elem.Defname)]
	if ok && known {
		return spec.prefix + arg + spec.suffix
	}
	return elem.Defname
}

// defElemArg pulls out a DefElem's string argument and tells you whether it had one.
func defElemArg(elem *pg_query.DefElem) (string, bool) {
	if elem.Arg == nil {
		return "", false
	}
	if s := elem.Arg.GetString_(); s != nil {
		return s.Sval, true
	}
	return "", false
}

// formatCreateTrigger renders a CREATE [OR REPLACE] TRIGGER statement.
func (f Formatter) formatCreateTrigger(stmt *pg_query.CreateTrigStmt) string {
	var b builder
	b.write(createKw)
	if stmt.Replace {
		b.write(orReplace)
	}
	b.write(" trigger ")
	b.write(stmt.Trigname)
	b.write(triggerTiming(stmt.Timing))
	b.write(triggerEvents(stmt.Events))
	b.write(onKw)
	b.write(formatRangeVar(stmt.Relation))
	b.write(triggerFunction(stmt.Funcname))
	b.write(";")
	return b.String()
}

// triggerTiming turns a trigger timing bit into its keyword.
func triggerTiming(timing int32) string {
	switch timing {
	case 1:
		return " before"
	case 2:
		return " after"
	case 4:
		return " instead of"
	default:
		return ""
	}
}

// triggerEvents renders a trigger's event list, OR-joined.
func triggerEvents(events int32) string {
	names := make([]string, 0, 3)
	if events&1 != 0 {
		names = append(names, "insert")
	}
	if events&2 != 0 {
		names = append(names, "delete")
	}
	if events&4 != 0 {
		names = append(names, "update")
	}
	if len(names) == 0 {
		return ""
	}
	return " " + strings.Join(names, " or ")
}

// triggerFunction renders a trigger's EXECUTE FUNCTION clause.
func triggerFunction(funcname []*pg_query.Node) string {
	if len(funcname) == 0 {
		return ""
	}
	return " execute function " + joinStringNodes(funcname) + "()"
}

// formatCreateCast renders a CREATE CAST statement.
func (f Formatter) formatCreateCast(stmt *pg_query.CreateCastStmt) string {
	var b builder
	b.write("create cast (")
	if stmt.Sourcetype != nil {
		b.write(f.formatTypeName(stmt.Sourcetype))
	}
	b.write(" as ")
	if stmt.Targettype != nil {
		b.write(f.formatTypeName(stmt.Targettype))
	}
	b.write(")")
	b.write(castImplementation(stmt))
	b.write(";")
	return b.String()
}

// castImplementation renders a cast's implementation clause.
func castImplementation(stmt *pg_query.CreateCastStmt) string {
	switch {
	case stmt.Func != nil:
		return " with function " + formatObjectWithArgs(stmt.Func)
	case stmt.Inout:
		return " with inout"
	default:
		return " without function"
	}
}

// formatCreateIndex renders a CREATE INDEX statement.
func (f Formatter) formatCreateIndex(stmt *pg_query.IndexStmt) string {
	var b builder
	b.write(createKw)
	if stmt.Unique {
		b.write(" unique")
	}
	b.write(" index")
	if stmt.Concurrent {
		b.write(" concurrently")
	}
	if stmt.IfNotExists {
		b.write(ifNotExists)
	}
	if stmt.Idxname != "" {
		b.write(" ")
		b.write(stmt.Idxname)
	}
	b.write(onKw)
	b.write(formatRangeVar(stmt.Relation))
	if stmt.AccessMethod != "" {
		b.write(" using ")
		b.write(stmt.AccessMethod)
	}
	b.write(f.indexColumns(stmt.IndexParams))
	b.write(";")
	return b.String()
}

// indexColumns renders a parenthesized index-parameter list, or "" when there
// aren't any.
func (f Formatter) indexColumns(params []*pg_query.Node) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, 0, len(params))
	for _, param := range params {
		if elem := param.GetIndexElem(); elem != nil {
			parts = append(parts, f.formatIndexElem(elem))
		}
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// formatIndexElem renders one index element — a column name or an expression.
func (f Formatter) formatIndexElem(elem *pg_query.IndexElem) string {
	if elem.Name != "" {
		return elem.Name
	}
	if elem.Expr != nil {
		return f.formatExpr(elem.Expr, 0)
	}
	return "/* complex index element */"
}

// formatAlterTable renders an ALTER TABLE statement and its commands.
func (f Formatter) formatAlterTable(stmt *pg_query.AlterTableStmt) string {
	var b builder
	b.write("alter table ")
	b.write(formatRangeVar(stmt.Relation))
	for i, cmd := range stmt.Cmds {
		if i > 0 {
			b.write(",")
		}
		b.write("\n")
		b.write(pad(f.indentSize))
		if alterCmd := cmd.GetAlterTableCmd(); alterCmd != nil {
			b.write(f.formatAlterTableCmd(alterCmd))
		}
	}
	b.write(";")
	return b.String()
}

// formatAlterTableCmd renders a single ALTER TABLE subcommand.
func (f Formatter) formatAlterTableCmd(cmd *pg_query.AlterTableCmd) string {
	switch cmd.Subtype {
	case pg_query.AlterTableType_AT_AddColumn:
		return "add column " + f.columnDefFromNode(cmd.Def)
	case pg_query.AlterTableType_AT_DropColumn:
		return "drop column " + cmd.Name
	case pg_query.AlterTableType_AT_AlterColumnType:
		return "alter column " + cmd.Name + " type " + f.columnTypeFromNode(cmd.Def)
	case pg_query.AlterTableType_AT_AddConstraint:
		return "add " + f.constraintFromNode(cmd.Def)
	case pg_query.AlterTableType_AT_DropConstraint:
		return "drop constraint " + cmd.Name
	default:
		return "/* unsupported alter table command */"
	}
}

// columnDefFromNode renders the column definition a command node carries.
func (f Formatter) columnDefFromNode(def *pg_query.Node) string {
	if def == nil {
		return ""
	}
	if col := def.GetColumnDef(); col != nil {
		return f.formatColumnDef(col)
	}
	return ""
}

// columnTypeFromNode renders the column type a command node carries.
func (f Formatter) columnTypeFromNode(def *pg_query.Node) string {
	if def == nil {
		return ""
	}
	if col := def.GetColumnDef(); col != nil {
		return f.formatTypeName(col.TypeName)
	}
	return ""
}

// constraintFromNode renders the constraint a command node carries.
func (f Formatter) constraintFromNode(def *pg_query.Node) string {
	if def == nil {
		return ""
	}
	if constraint := def.GetConstraint(); constraint != nil {
		return f.formatConstraint(constraint)
	}
	return ""
}
