package sql

import (
	"strings"

	errs "github.com/gomatic/go-error"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// ErrScan means we couldn't scan the SQL text into tokens. Match it with
// [errors.Is], not by string.
const ErrScan errs.Const = "scan SQL"

// Scan tokenizes SQL text into PostgreSQL's lexical tokens, each with its source
// offsets and keyword classification. Unlike [Parse] it doesn't require a
// well-formed statement, but a lexical error (an unterminated string, say) comes
// back wrapped in [ErrScan].
func Scan(sql SQL) (*pg_query.ScanResult, error) {
	result, err := pg_query.Scan(string(sql))
	if err != nil {
		return nil, ErrScan.With(err)
	}
	return result, nil
}

// LowerKeywords lowercases every SQL keyword in text and leaves everything else —
// string literals, quoted identifiers, dollar-quoted bodies, numbers, operators —
// exactly as written. A lexical error comes back wrapped in [ErrScan].
func LowerKeywords(text string) (SQL, error) {
	result, err := Scan(SQL(text))
	if err != nil {
		return "", err
	}
	var out strings.Builder
	cursor := 0
	for _, tok := range result.Tokens {
		cursor = writeToken(&out, text, tok, cursor)
	}
	_, _ = out.WriteString(text[cursor:])
	return SQL(out.String()), nil
}

// writeToken copies the gap before tok verbatim, then the token itself —
// lowercased when it's a keyword — and reports the new cursor.
func writeToken(out *strings.Builder, text string, tok *pg_query.ScanToken, cursor int) int {
	start, end := int(tok.Start), int(tok.End)
	_, _ = out.WriteString(text[cursor:start])
	span := text[start:end]
	if tok.KeywordKind != pg_query.KeywordKind_NO_KEYWORD {
		span = strings.ToLower(span)
	}
	_, _ = out.WriteString(span)
	return end
}
