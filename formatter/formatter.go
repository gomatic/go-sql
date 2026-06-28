// Package formatter renders PostgreSQL statements as canonically-styled SQL
// text. It runs the input through the root [sql] package, walks each statement's
// AST, and emits a lowercase, leading-comma layout. Anything it can't render
// comes back as [ErrUnsupportedStatement].
package formatter

import (
	"fmt"
	"strings"

	errs "github.com/gomatic/go-error"
	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/gomatic/go-sql"
)

// ErrUnsupportedStatement is what you get when the formatter hits a statement
// kind it can't render. Match it with [errors.Is], not by string.
const ErrUnsupportedStatement errs.Const = "unsupported statement"

// Keyword fragments the statement formatters share. We keep them here so the
// same literal doesn't get repeated across files.
const (
	createKw    = "create"
	ifNotExists = " if not exists"
	nullKw      = "null"
	onKw        = " on "
	orReplace   = " or replace"
	whereKw     = "where "
)

// defaultIndent is how many columns one indentation step is worth.
const defaultIndent = 2

// Formatter renders SQL with a fixed indentation step. It's an immutable value:
// the methods take a value receiver and [New] hands back a value, so you can
// copy and share a Formatter freely.
type Formatter struct {
	indentSize int
}

// New returns a Formatter that indents two spaces per step.
func New() Formatter {
	return Formatter{indentSize: defaultIndent}
}

// Format parses query and renders each statement, joining multiple ones with a
// blank line. A parse failure comes back unwrapped (it already carries
// [sql.ErrParse]); a statement we can't render yields [ErrUnsupportedStatement].
func (f Formatter) Format(query string) (string, error) {
	result, err := sql.Parse(sql.SQL(query))
	if err != nil {
		return "", err
	}

	formatted := make([]string, 0, len(result.Stmts))
	for _, stmt := range result.Stmts {
		out, err := f.formatStatement(stmt)
		if err != nil {
			return "", err
		}
		formatted = append(formatted, out)
	}

	return strings.Join(formatted, "\n\n"), nil
}

// formatStatement hands a raw statement to the first group handler that knows
// its node kind. An empty statement renders empty; a kind nobody recognizes
// yields [ErrUnsupportedStatement].
func (f Formatter) formatStatement(stmt *pg_query.RawStmt) (string, error) {
	if stmt.Stmt == nil {
		return "", nil
	}

	node := stmt.Stmt.Node
	groups := []func(any) (string, bool){
		f.formatCreateStatement,
		f.formatModifyStatement,
		f.formatQueryStatement,
	}
	for _, group := range groups {
		if out, ok := group(node); ok {
			return out, nil
		}
	}

	return "", ErrUnsupportedStatement.With(nil, "type", fmt.Sprintf("%T", node))
}

// formatCreateStatement covers the CREATE family of statements.
func (f Formatter) formatCreateStatement(node any) (string, bool) {
	switch n := node.(type) {
	case *pg_query.Node_CreateSchemaStmt:
		return formatCreateSchema(n.CreateSchemaStmt), true
	case *pg_query.Node_CreateStmt:
		return f.formatCreateTable(n.CreateStmt), true
	case *pg_query.Node_ViewStmt:
		return f.formatCreateView(n.ViewStmt), true
	case *pg_query.Node_CreateFunctionStmt:
		return f.formatCreateFunction(n.CreateFunctionStmt), true
	case *pg_query.Node_CreateTrigStmt:
		return f.formatCreateTrigger(n.CreateTrigStmt), true
	case *pg_query.Node_CreateCastStmt:
		return f.formatCreateCast(n.CreateCastStmt), true
	case *pg_query.Node_IndexStmt:
		return f.formatCreateIndex(n.IndexStmt), true
	default:
		return "", false
	}
}

// formatModifyStatement covers the statements that alter or drop objects.
func (f Formatter) formatModifyStatement(node any) (string, bool) {
	switch n := node.(type) {
	case *pg_query.Node_AlterTableStmt:
		return f.formatAlterTable(n.AlterTableStmt), true
	case *pg_query.Node_CommentStmt:
		return formatComment(n.CommentStmt), true
	case *pg_query.Node_GrantStmt:
		return formatGrant(n.GrantStmt), true
	case *pg_query.Node_DropStmt:
		return formatDrop(n.DropStmt), true
	case *pg_query.Node_DoStmt:
		return formatDo(n.DoStmt), true
	default:
		return "", false
	}
}

// formatQueryStatement covers the data-query and data-modification statements.
func (f Formatter) formatQueryStatement(node any) (string, bool) {
	switch n := node.(type) {
	case *pg_query.Node_SelectStmt:
		return f.formatSelect(n.SelectStmt, 0), true
	case *pg_query.Node_InsertStmt:
		return f.formatInsert(n.InsertStmt), true
	case *pg_query.Node_UpdateStmt:
		return f.formatUpdate(n.UpdateStmt), true
	case *pg_query.Node_DeleteStmt:
		return f.formatDelete(n.DeleteStmt), true
	default:
		return "", false
	}
}

// stringNodeValues pulls the string values out of a node list, skipping any
// node that isn't a string (a star, say).
func stringNodeValues(nodes []*pg_query.Node) []string {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if s := n.GetString_(); s != nil {
			parts = append(parts, s.Sval)
		}
	}
	return parts
}

// joinStringNodes joins a list's string-valued nodes with a dot.
func joinStringNodes(nodes []*pg_query.Node) string {
	return strings.Join(stringNodeValues(nodes), ".")
}

// pad returns width spaces.
func pad(width int) string {
	return strings.Repeat(" ", width)
}

// builder piles up rendered SQL text. It wraps [strings.Builder] so writes
// return nothing: strings.Builder.WriteString never fails, so handing back its
// always-nil error would just be noise — we swallow that one error right here.
// The methods need a pointer receiver because a strings.Builder mustn't be
// copied once it's been written to.
type builder struct {
	inner strings.Builder
}

// write tacks each part onto the builder, in order.
func (b *builder) write(parts ...string) {
	for _, part := range parts {
		_, _ = b.inner.WriteString(part)
	}
}

// String hands back everything we've piled up so far.
func (b *builder) String() string {
	return b.inner.String()
}
