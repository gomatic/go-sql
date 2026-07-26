//go:build !cgo

package sql

import (
	errs "github.com/gomatic/go-error"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// ErrNoCGO means this binary was built without cgo, so libpg_query is absent and
// nothing can be parsed. Match it with [errors.Is].
//
// The package still builds and links without cgo — deliberately. A consumer's
// release build cross-compiles for targets where cgo is unavailable, and before
// this split the whole package simply failed to compile there ("undefined:
// pg_query.Parse"), taking down every CLI that imported it. Failing at the call
// with a matchable sentinel is honest and keeps the build green; failing to
// compile broke consumers that never call these functions on those targets.
const ErrNoCGO errs.Const = "SQL parsing requires a cgo build"

// The stub engine. Every binding mirrors its cgo counterpart's signature so the
// callers are identical under both constraints.
var (
	parseSQL = func(string) (*pg_query.ParseResult, error) {
		return nil, ErrNoCGO.With(nil)
	}
	deparseTree = func(*pg_query.ParseResult) (string, error) {
		return "", ErrNoCGO.With(nil)
	}
	fingerprintSQL = func(string) (string, error) {
		return "", ErrNoCGO.With(nil)
	}
	scanSQL = func(string) (*pg_query.ScanResult, error) {
		return nil, ErrNoCGO.With(nil)
	}
)
