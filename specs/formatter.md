# Spec: meaning-preserving SQL formatter

## Goal

Replace the `formatter` package's destructive AST-deparser with a **meaning-preserving** formatter: it renders PostgreSQL SQL into the canonical `sql-standards` layout while guaranteeing the output parses to the same statement and keeps every comment. It must be designed so the sibling `compare` and `normalize` capabilities stay coherent with formatted output.

## Why

A formatter's one inviolable contract is _never change meaning_. The current engine silently violates it: any expression node beyond `ColumnRef`/`AConst`/`BoolExpr`/`FuncCall` is replaced by `/* complex expression */`, and all comments are dropped (`formatter/expression.go:30`; proven by the demo on `a+b`, `CASE`, `IN`, casts, and a leading comment). Downstream, `graft` edits `.sql` definition files in place, so a formatter that eats expressions or comments is actively dangerous.

## Architecture — Hybrid (AST structure + verbatim token leaves)

Use the AST **only** to decide structure; never to regenerate leaf text.

1. **Parse front-end (shared).** `sql.Parse` (AST) + a new `sql.Scan` wrapper over `pg_query.Scan` (token stream with byte offsets, including `Token_SQL_COMMENT`/`Token_C_COMMENT`). One parse feeds every capability.
2. **Document IR.** A small Wadler/Prettier-style layout algebra — `text`, `line`/`softline`, `group`, `indent`, `concat` — with one width-aware renderer (`max-width`, default 120). Pure Go, independently testable; owns all wrapping and indentation.
3. **Verbatim leaves.** Each AST leaf (expression, literal, qualified name) is rendered by slicing its `[Start,End)` bytes from the source via the token stream — so the formatter _cannot_ misrender or drop an expression it doesn't structurally understand. Unknown/complex subtrees render as their verbatim source span, never a placeholder.
4. **Comment attachment.** A pre-pass indexes comment tokens by offset and attaches each to the nearest enclosing/following node, so the doc-builder can place leading/trailing comments.
5. **Verification gate (the format/compare/normalize unity).** `Format` only returns transformed text when `compare.Compare(in, out)` reports no statement-level difference **and** the comment multiset is preserved (fast path: `pg_query.Fingerprint(in)==Fingerprint(out)` for the structural half). On any violation, return the input statement untouched — never emit corrupted SQL.

## Requirements

- R1 — Round-trip safe: for all parseable input, `Fingerprint(format(x)) == Fingerprint(x)`.
- R2 — Comment-preserving: every `--` and `/* */` comment in the input appears in the output.
- R3 — No placeholders: the string `/* complex` never appears in output that wasn't in the input.
- R4 — Idempotent: `format(format(x)) == format(x)`.
- R5 — Width-aware: lines wrap at the configured `max-width` where the layout allows; never mid-token.
- R6 — Style: lowercase keywords, 2-space indent, leading-comma lists, dollar-quoted bodies — per the `sql-standards` skill.
- R7 — Coherent with siblings: `compare.Compare(x, format(x))` is empty; `normalize` of formatted output equals `normalize` of input.
- R8 — Honest 100% coverage: tests assert the _contract_ (specific round-trips, comment placement, the verification gate firing), never pin placeholder/destructive output.

## Acceptance criteria

- The five demonstrated cases (`a+b`, `CASE`, `IN`, leading comment, `x::int`) round-trip with comments intact and no placeholders.
- A corpus property test asserts R1–R4 across every statement kind pg_query parses, including statements the structural layer doesn't specially handle (they pass through verbatim, still round-trip).
- `make check` green in `gomatic/go-sql` (vet, lint, staticcheck, govulncheck, 100% verified coverage).

## Out of scope (here)

- Reimplementing `compare`/`normalize` (updated later; the formatter must not break them).
- `sqlrest/harrow` — separate repo, a `urfave/cli/v3` CLI over this library, built after the library lands.
