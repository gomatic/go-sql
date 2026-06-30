package sql

import pg_query "github.com/pganalyze/pg_query_go/v6"

// Comments returns the text of every comment in sql, both line (`-- …`) and
// block (`/* … */`), in source order. It's the basis for checking that a
// reformat preserved every comment. A lexical error comes back wrapped in
// [ErrScan].
func Comments(sql SQL) ([]string, error) {
	result, err := Scan(sql)
	if err != nil {
		return nil, err
	}
	text := string(sql)
	comments := make([]string, 0)
	for _, tok := range result.Tokens {
		if isComment(tok.Token) {
			comments = append(comments, text[tok.Start:tok.End])
		}
	}
	return comments, nil
}

// isComment reports whether a token kind is one of the two comment kinds.
func isComment(tok pg_query.Token) bool {
	return tok == pg_query.Token_SQL_COMMENT || tok == pg_query.Token_C_COMMENT
}
