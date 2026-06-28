package compare

// handler holds the per-kind behavior for a statement: how to derive its
// identity and how to diff two statements of that kind. We keep these as
// first-class functions so kinds that just want the generic diff don't carry any
// boilerplate.
type handler struct {
	identify func(stmt statementData) identity
	diff     func(source, target statementData) statementDiffs
}

// registry maps each handled statement type to its handler. We build it once per
// comparison and never touch it again, so reads are safe to share across
// goroutines.
type registry map[pgQueryType]handler

// newRegistry returns a registry wired up with every supported statement
// handler.
func newRegistry() registry {
	return registry{
		typeAlterTable:     {identify: alterTableIdentity, diff: genericDiff},
		typeComment:        {identify: commentIdentity, diff: commentDiff},
		typeCreateFunction: {identify: createFunctionIdentity, diff: createFunctionDiff},
		typeCreateSchema:   {identify: createSchemaIdentity, diff: genericDiff},
		typeCreateTable:    {identify: createTableIdentity, diff: genericDiff},
		typeCreateTrigger:  {identify: createTriggerIdentity, diff: genericDiff},
		typeIndex:          {identify: createIndexIdentity, diff: genericDiff},
		typeView:           {identify: createViewIdentity, diff: genericDiff},
		typeDrop:           {identify: dropIdentity, diff: genericDiff},
		typeGrant:          {identify: grantIdentity, diff: genericDiff},
	}
}

// get looks up the handler for a statement type, and tells you whether one is
// registered.
func (r registry) get(stmtType pgQueryType) (handler, bool) {
	h, known := r[stmtType]
	return h, known
}

// diff diffs two statements of the given type. If the type isn't registered, you
// get nil back.
func (r registry) diff(stmtType pgQueryType, source, target statementData) statementDiffs {
	if h, known := r.get(stmtType); known {
		return h.diff(source, target)
	}
	return nil
}

// genericDiff is the default diff: a plain normalized structural comparison.
func genericDiff(source, target statementData) statementDiffs {
	return computeDiffs(source, target)
}
