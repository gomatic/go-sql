//go:build !cgo

package formatter

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"

	sql "github.com/gomatic/go-sql"
)

// deparseTree is the stub deparser for a build without cgo, where libpg_query is
// absent. It reports the same sentinel the sql package does, so a caller matches
// one error regardless of which package it reached.
var deparseTree = func(*pg_query.ParseResult) (string, error) {
	return "", sql.ErrNoCGO.With(nil)
}
