package sql

import (
	errs "github.com/gomatic/go-error"
)

// ErrFingerprint means we couldn't fingerprint the SQL text. Match it with
// [errors.Is], not by string.
const ErrFingerprint errs.Const = "fingerprint SQL"

// Fingerprint returns PostgreSQL's structural fingerprint of sql: two statements
// that mean the same thing modulo formatting, literal values, and case share a
// fingerprint. A parse failure comes back wrapped in [ErrFingerprint].
func Fingerprint(sql SQL) (string, error) {
	fp, err := fingerprintSQL(string(sql))
	if err != nil {
		return "", ErrFingerprint.With(err)
	}
	return fp, nil
}
