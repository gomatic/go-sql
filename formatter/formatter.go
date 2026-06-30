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

// statementSeparator joins consecutive statements: a terminating semicolon plus
// a blank line. The semicolon is what keeps multi-statement output valid SQL —
// without it `select 1\n\nselect 2` is a syntax error — so the whole rendering
// re-parses as the same statements and reformatting is idempotent.
const statementSeparator = ";\n\n"

// Format parses query and renders each statement, separating consecutive ones
// with a semicolon and a blank line so the whole output re-parses as the same
// statements. A parse failure comes back unwrapped (it already carries
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

	return strings.Join(formatted, statementSeparator), nil
}

// formatStatement renders one statement past the verification gate, best layout
// first: the house style when it covers the statement, then the canonical
// deparse, then the verbatim source when neither can be proven faithful. An empty
// statement renders empty.
func formatStatement(query sql.SQL, stmt *pg_query.RawStmt) string {
	if stmt.Stmt == nil {
		return ""
	}
	return chooseFormatted(statementSource(query, stmt), candidates(stmt)...)
}

// candidates lists the renderings to try, best first: the house style if it
// covers this statement, then the canonical deparse.
func candidates(stmt *pg_query.RawStmt) []string {
	candidates := make([]string, 0, 2)
	if house, ok := houseStatement(stmt); ok {
		candidates = append(candidates, house)
	}
	if canonical, ok := canonicalStatement(stmt); ok {
		candidates = append(candidates, canonical)
	}
	return candidates
}

// pgSpace is exactly the whitespace PostgreSQL's lexer ignores: space, tab,
// newline, carriage return, form feed, and vertical tab. statementSource trims
// against this set rather than Go's wider unicode notion ([strings.TrimSpace]),
// because a character pg_query treats as significant — a non-ASCII space such as
// U+00A0 or U+2000 is part of the statement, not boundary whitespace — must stay
// in the verbatim slice. Trimming it with Go's notion would yield a fallback that
// no longer means the same statement and let the gate emit it, silently changing
// what the statement does.
const pgSpace = " \t\n\r\f\v"

// statementSource slices the verbatim source of one statement out of query,
// dropping the surrounding PostgreSQL whitespace and trailing semicolon the
// boundary carries. It's both the gate's reference and the last-resort output.
func statementSource(query sql.SQL, stmt *pg_query.RawStmt) string {
	text := string(query)
	start := int(stmt.StmtLocation)
	end := start + int(stmt.StmtLen)
	if stmt.StmtLen == 0 || end > len(text) {
		end = len(text)
	}
	return strings.Trim(strings.TrimSuffix(strings.Trim(text[start:end], pgSpace), ";"), pgSpace)
}
