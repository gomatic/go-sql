package formatter

import (
	"slices"

	sql "github.com/gomatic/go-sql"
)

// chooseFormatted returns the first candidate that renders original without
// changing its meaning or its comments, falling back to original verbatim when
// none is safe. This is what keeps the formatter from ever corrupting SQL: a
// candidate it can't prove equivalent is simply not used.
func chooseFormatted(original string, candidates ...string) string {
	for _, candidate := range candidates {
		if preservesMeaning(original, candidate) {
			return candidate
		}
	}
	return original
}

// preservesMeaning reports whether candidate is the same statement as original
// up to formatting: identical PostgreSQL fingerprints and the same multiset of
// comments. A candidate that doesn't parse, or whose fingerprint or comments
// differ, is not preserving.
func preservesMeaning(original, candidate string) bool {
	originalFP, err := sql.Fingerprint(sql.SQL(original))
	if err != nil {
		return false
	}
	candidateFP, err := sql.Fingerprint(sql.SQL(candidate))
	if err != nil || originalFP != candidateFP {
		return false
	}
	return commentsEqual(sql.SQL(original), sql.SQL(candidate))
}

// commentsEqual reports whether two SQL texts carry the same multiset of
// comments. Order doesn't matter — only that none were dropped or invented.
func commentsEqual(original, candidate sql.SQL) bool {
	originalComments, err := sql.Comments(original)
	if err != nil {
		return false
	}
	candidateComments, err := sql.Comments(candidate)
	if err != nil {
		return false
	}
	slices.Sort(originalComments)
	slices.Sort(candidateComments)
	return slices.Equal(originalComments, candidateComments)
}
