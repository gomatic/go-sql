//go:build cgo

package formatter

import pg_query "github.com/pganalyze/pg_query_go/v6"

// deparseTree is the real deparser. It is bound here rather than imported from
// the sql package so each package owns its own cgo boundary and no binding has
// to be exported to cross a package line.
var deparseTree = pg_query.Deparse
