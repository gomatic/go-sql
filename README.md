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
