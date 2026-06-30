package sql

import (
	errs "github.com/gomatic/go-error"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// ErrFingerprint means we couldn't fingerprint the SQL text. Match it with
// [errors.Is], not by string.
const ErrFingerprint errs.Const = "fingerprint SQL"

// Fingerprint returns PostgreSQL's structural fingerprint of sql: two statements
// that mean the same thing modulo formatting, literal values, and case share a
// fingerprint. A parse failure comes back wrapped in [ErrFingerprint].
func Fingerprint(sql SQL) (string, error) {
	fp, err := pg_query.Fingerprint(string(sql))
	if err != nil {
		return "", ErrFingerprint.With(err)
	}
	return fp, nil
}
