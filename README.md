# go-sql

PostgreSQL SQL parsing, canonical normalization, AST diffing, and formatting for Go — a thin, well-tested layer over [`pg_query_go`](https://github.com/pganalyze/pg_query_go).

## Packages

| Import | Purpose |
|---|---|
| [`github.com/gomatic/go-sql`](.) | Parse SQL to a PostgreSQL AST, canonically order column lists, deparse back to SQL, and render statements as normalized JSON. |
| [`github.com/gomatic/go-sql/formatter`](formatter/) | Format SQL text into a canonical, indented layout. |
| [`github.com/gomatic/go-sql/compare`](compare/) | Diff two SQL statements at the AST level, ignoring cosmetic differences. |
| [`github.com/gomatic/go-sql/normalize/sql`](normalize/sql/) | Normalize SQL text (whitespace, comments, column ordering) for canonical comparison. |
| [`github.com/gomatic/go-sql/normalize/plpgsql`](normalize/plpgsql/) | Normalize PL/pgSQL function bodies, token by token. |

## Usage

```go
import (
	sql "github.com/gomatic/go-sql"
	"github.com/gomatic/go-sql/formatter"
	"github.com/gomatic/go-sql/compare"
	sqlnorm "github.com/gomatic/go-sql/normalize/sql"
)

// Format: canonical, lowercase-keyword layout. Never changes a statement's
// meaning and never corrupts SQL — a statement it can't prove it rendered
// faithfully is emitted verbatim instead.
out, _ := formatter.New().Format("SELECT   A,B FROM t")   // "select a, b from t"

// Normalize: a deterministic canonical form for meaning-based comparison.
a := sqlnorm.SQL("select  a , b from t").Normalize()      // "SELECT a, b FROM t"
b := sqlnorm.SQL("SELECT a, b FROM t;").Normalize()
equal := a == b                                           // true

// Compare: structural diff of two scripts, ignoring cosmetic differences.
diff, _ := compare.Compare("create table t (a int)", "create table t (a int, b text)")
changed := diff.HasChanges()                              // true
```

### Guarantees

These properties are exercised by seeded [Go fuzz](https://go.dev/doc/security/fuzz/) targets, not just examples:

- **[`formatter.Format`](formatter/) preserves meaning.** Output always re-parses to the same statement (identical PostgreSQL fingerprint); formatting is idempotent (`Format(Format(x)) == Format(x)`). Comments are preserved per statement — a comment trailing the final statement, attached to no statement, is the one exception and is dropped.
- **[`Normalize`](normalize/) is deterministic and idempotent.** The same input always yields the same canonical form, and re-normalizing is a no-op (for input the underlying engine canonicalizes stably).
- **[`compare.Compare`](compare/) is reflexive.** A script compared against itself reports no changes.

## Errors

Every error a package can emit is a [`errs.Const`](https://github.com/gomatic/go-error) sentinel — match with `errors.Is`, never by string:

```go
tree, err := sql.Parse("not valid ((")
if errors.Is(err, sql.ErrParse) {
    // ...
}
```

## Build configuration is managed upstream

`Makefile`, `.golangci.yaml`, `.editorconfig`, `.gitignore`, `scripts/`, and `.github/` are distributed and owned by [`nicerobot/tools.repository`](https://github.com/nicerobot/tools.repository) (sourced from `nicerobot/tools.build/build/go/Makefile`). Do not edit them in-tree — they are overwritten on the next push. Per-repo customization goes in a `Makefile.local`.

```sh
make check   # vet, lint, staticcheck, govulncheck, 100% coverage gate
```
