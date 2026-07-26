//go:build cgo

package sql

import pg_query "github.com/pganalyze/pg_query_go/v6"

// The real engine. pg_query's parsing functions live behind a cgo build
// constraint because they call into libpg_query; its protobuf types do not, so
// everything in this package that only walks the AST compiles either way and
// only these four bindings need splitting.
var (
	parseSQL       = pg_query.Parse
	deparseTree    = pg_query.Deparse
	fingerprintSQL = pg_query.Fingerprint
	scanSQL        = pg_query.Scan
)
