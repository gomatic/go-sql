// Package formatter renders PostgreSQL statements as canonically-styled SQL
// text. It parses the input through the root [sql] package, renders each
// statement, and then checks every rendering against the original: a statement
// it can't prove it reformatted faithfully — same meaning, same comments — is
// emitted verbatim instead. So the formatter never changes what a statement does
// and never drops a comment.
//
// Rendering today is PostgreSQL's own canonical deparse with keywords
// lowercased, which is correct for every statement kind. The multi-line house
// layout (leading commas, a clause per line) is built on top of the [doc]
// layout engine and replaces the canonical fallback statement kind by statement
// kind.
package formatter

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/gomatic/go-sql"
)

// Formatter renders SQL. It's an immutable value with nothing to configure yet,
// so [New] hands one back by value and you can copy and share it freely.
type Formatter struct{}

// New returns a Formatter.
func New() Formatter {
	return Formatter{}
}

// Format parses query and renders each statement, joining multiple ones with a
// blank line. A parse failure comes back unwrapped (it already carries
// [sql.ErrParse]). No statement kind is ever an error: one that can't be
// deparsed, or can't be rendered faithfully, falls back to its verbatim source.
func (Formatter) Format(query sql.SQL) (string, error) {
	result, err := sql.Parse(query)
	if err != nil {
		return "", err
	}

	formatted := make([]string, 0, len(result.Stmts))
	for _, stmt := range result.Stmts {
		formatted = append(formatted, formatStatement(query, stmt))
	}

	return strings.Join(formatted, "\n\n"), nil
}

// formatStatement renders one statement past the verification gate: the
// canonical deparse when it holds up, and the verbatim source when it can't be
// proven faithful. An empty statement renders empty.
func formatStatement(query sql.SQL, stmt *pg_query.RawStmt) string {
	if stmt.Stmt == nil {
		return ""
	}
	original := statementSource(query, stmt)
	if canonical, ok := canonicalStatement(stmt); ok {
		return chooseFormatted(original, canonical)
	}
	return chooseFormatted(original)
}

// statementSource slices the verbatim source of one statement out of query,
// dropping the surrounding whitespace and trailing semicolon the boundary
// carries. It's both the gate's reference and the last-resort output.
func statementSource(query sql.SQL, stmt *pg_query.RawStmt) string {
	text := string(query)
	start := int(stmt.StmtLocation)
	end := start + int(stmt.StmtLen)
	if stmt.StmtLen == 0 || end > len(text) {
		end = len(text)
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text[start:end]), ";"))
}
