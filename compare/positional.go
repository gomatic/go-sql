package compare

// The source-position metadata stripped before two statements are compared.
// These keys move with FORMATTING rather than with meaning, so leaving them in
// would make every reformatted-but-identical statement compare unequal — which
// is the whole question this package answers.

// keyLocation is the per-node source-position key.
const keyLocation fieldKey = "location"

// positionalField is a source-position key: one that moves with FORMATTING
// rather than with meaning. It is its own type rather than a fieldKey because
// the set below is exactly these five — fieldKey names two dozen navigation
// keys, none of which are positional, and typing the set as fieldKey would
// claim to be a decision about all of them.
type positionalField string

// The source-position keys stripped before comparing.
const (
	positionalColno        positionalField = "colno"
	positionalLineno       positionalField = "lineno"
	positionalLocation                     = positionalField(keyLocation)
	positionalStmtLen      positionalField = "stmt_len"
	positionalStmtLocation positionalField = "stmt_location"
)

// positionalFields are the source-position keys we strip before comparing: they
// move around with formatting, not with meaning.
var positionalFields = map[positionalField]struct{}{
	positionalColno:        {},
	positionalLineno:       {},
	positionalLocation:     {},
	positionalStmtLen:      {},
	positionalStmtLocation: {},
}
