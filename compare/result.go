package compare

import (
	"cmp"
	"slices"
)

// Result is what you get back from comparing a source script against a target
// script. Its JSON shape is stable: three statement lists, each under its own
// lowercase key. A statement that shows up only in the target is Added, one only
// in the source is Removed, and one in both with different content is Changed.
type Result struct {
	Added   []StatementResult `json:"added"`
	Changed []StatementResult `json:"changed"`
	Removed []StatementResult `json:"removed"`
}

// HasChanges tells you whether the comparison found any difference at all.
func (r Result) HasChanges() bool {
	return len(r.Added) > 0 || len(r.Changed) > 0 || len(r.Removed) > 0
}

// StatementResult describes one statement in a [Result]. For added and removed
// statements, Statement carries the whole decoded statement; for changed ones,
// Diffs carries the field-level differences and Statement is empty.
type StatementResult struct {
	Statement map[string]any `json:"statement,omitempty"`
	Identity  string         `json:"identity"`
	Type      string         `json:"type"`
	Diffs     []Diff         `json:"diffs,omitempty"`
}

// Diff is one field-level difference between two statements. Source or Target is
// nil when the field only exists on one side.
type Diff struct {
	Source any    `json:"source"`
	Target any    `json:"target"`
	Field  string `json:"field"`
}

// diffStatements pairs up the indexed source and target statements and builds
// the Result: removed and changed come from the source side, added from the
// target side.
func diffStatements(reg registry, source, target stmtsByID) Result {
	var result Result
	for _, e := range sortedByIndex(source) {
		classifySource(reg, e.id, e.stmt, target, &result)
	}
	for _, e := range sortedByIndex(target) {
		appendIfAdded(e.id, e.stmt, source, &result)
	}
	return result
}

// indexedEntry pairs an identity with its statement for stable iteration.
type indexedEntry struct {
	id   identity
	stmt indexedStatement
}

// sortedByIndex returns a map's entries ordered by each statement's position in
// its script, so the Result lists are deterministic rather than map-ordered.
func sortedByIndex(m stmtsByID) []indexedEntry {
	entries := make([]indexedEntry, 0, len(m))
	for id, s := range m {
		entries = append(entries, indexedEntry{id: id, stmt: s})
	}
	slices.SortFunc(entries, func(a, b indexedEntry) int {
		return cmp.Compare(a.stmt.index, b.stmt.index)
	})
	return entries
}

// classifySource marks a source statement removed when the target doesn't have
// it, or changed when the paired target statement differs.
func classifySource(reg registry, id identity, src indexedStatement, target stmtsByID, result *Result) {
	tgt, paired := target[id]
	if !paired {
		result.Removed = append(result.Removed, fullStatement(id, src.stmt))
		return
	}
	stmtType := statementType(src.stmt)
	if diffs := reg.diff(stmtType, src.stmt, tgt.stmt); len(diffs) > 0 {
		result.Changed = append(result.Changed, changedStatement(id, stmtType, diffs))
	}
}

// appendIfAdded marks a target statement added when the source doesn't have it.
func appendIfAdded(id identity, tgt indexedStatement, source stmtsByID, result *Result) {
	if _, paired := source[id]; paired {
		return
	}
	result.Added = append(result.Added, fullStatement(id, tgt.stmt))
}

// fullStatement builds an added/removed entry that carries the whole statement.
func fullStatement(id identity, stmt statementData) StatementResult {
	return StatementResult{
		Identity:  string(id),
		Type:      string(statementType(stmt)),
		Statement: map[string]any(stmt),
	}
}

// changedStatement builds a changed entry that carries the field-level diffs.
func changedStatement(id identity, stmtType pgQueryType, diffs statementDiffs) StatementResult {
	return StatementResult{
		Identity: string(id),
		Type:     string(stmtType),
		Diffs:    publicDiffs(diffs),
	}
}

// publicDiffs converts the internal field diffs into the exported [Diff] shape.
func publicDiffs(diffs statementDiffs) []Diff {
	out := make([]Diff, len(diffs))
	for i, d := range diffs {
		out[i] = Diff{Field: string(d.Field), Source: d.Source, Target: d.Target}
	}
	return out
}

// statementType pulls the pg_query type tag out of a decoded statement.
func statementType(stmt statementData) pgQueryType {
	stmtObj := extractMap(stmt, keyStmt)
	if stmtObj == nil {
		return ""
	}
	return pgQueryType(extractString(stmtObj, keyType))
}

// indexStatements indexes statements by identity, skipping any whose type has no
// handler or whose identity comes out empty.
func indexStatements(reg registry, stmts []statementData) stmtsByID {
	result := make(stmtsByID, len(stmts))
	for i, stmt := range stmts {
		h, known := reg.get(statementType(stmt))
		if !known {
			continue
		}
		id := h.identify(stmt)
		if id == "" {
			continue
		}
		result[id] = indexedStatement{index: stmtIndex(i), stmt: stmt}
	}
	return result
}
