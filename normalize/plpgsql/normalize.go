// Package plpgsql canonicalizes PL/pgSQL code so you can compare it for meaning
// rather than formatting. It strips line and block comments, keeps single-,
// double-, and dollar-quoted strings verbatim, and normalizes whitespace and
// operator spacing down to one canonical form. There's no error path: every
// input gives you a deterministic result.
package plpgsql

import (
	"strings"
	"unicode"
)

// Domain types for PL/pgSQL normalization.
type (
	hasWhitespace  bool   // hasWhitespace says whitespace or a comment came before the current token.
	isOperator     bool   // isOperator says whether a rune is an operator character.
	normalizedText string // normalizedText is text we've already written to the result builder.
	quotedString   string // quotedString is a fully scanned quoted or dollar-quoted literal.
	runeCount      int    // runeCount is how many runes a scan consumed.
	runeIndex      int    // runeIndex is a position within the rune slice.
	runeType       rune   // runeType is a single classified rune.
)

// spaceDecision is what a single spacing rule decides.
type spaceDecision int

const (
	spaceUndecided spaceDecision = iota // spaceUndecided punts to the next rule.
	spaceYes                            // spaceYes inserts a separating space.
	spaceNo                             // spaceNo holds back a separating space.
)

// Body is normalized PL/pgSQL code.
type Body string

// Normalize returns the canonical form of the PL/pgSQL code.
func (p Body) Normalize() Body { return normalize(p) }

// normalize collapses whitespace, drops comments, keeps quoted literals intact,
// and trims a trailing semicolon.
func normalize(input Body) Body {
	var result strings.Builder
	runes := []rune(input)
	i := runeIndex(0)
	had := hasWhitespace(false)

	for int(i) < len(runes) {
		i, had = processToken(runes, &result, i, had)
	}

	// Strip every trailing semicolon — each a meaningless empty statement — plus
	// any whitespace around them in one right-trim pass, so the canonical form is
	// idempotent (";;" settles on "" in a single pass, not "" only after two).
	canonical := strings.TrimRight(result.String(), "; \t\n\r\f\v")
	return Body(strings.TrimSpace(canonical))
}

// processToken consumes the token at i, appends its normalized text to result, and
// hands back the next position plus whether that token was whitespace or a comment.
func processToken(runes []rune, result *strings.Builder, i runeIndex, isHad hasWhitespace) (runeIndex, hasWhitespace) {
	if next, nextHad, ok := scanStructured(runes, result, i, isHad); ok {
		return next, nextHad
	}

	idx := int(i)
	if unicode.IsSpace(runes[idx]) {
		return runeIndex(idx + 1), hasWhitespace(true)
	}

	addSpaceIfNeeded(result, isHad, runeType(runes[idx]))
	emit(result, emittedText(string(runes[idx])))
	return runeIndex(idx + 1), hasWhitespace(false)
}

// emittedText is a fragment of canonical output appended to the result builder.
type emittedText string

// emit appends s to result. [strings.Builder.WriteString] is documented to
// return a nil error, so the error is swallowed in this one spot rather than
// threaded through every caller.
func emit(result *strings.Builder, s emittedText) {
	_, _ = result.WriteString(string(s))
}

// scanStructured handles the multi-rune token shapes — dollar quotes, quoted
// strings, comments. The bool comes back false when none of them apply, so the
// caller treats the rune as an ordinary character.
func scanStructured(
	runes []rune,
	result *strings.Builder,
	i runeIndex,
	isHad hasWhitespace,
) (runeIndex, hasWhitespace, bool) {
	idx := int(i)
	switch r := runes[idx]; {
	case r == '$':
		return scanDollarToken(runes, result, i, isHad)
	case r == '\'' || r == '"':
		return scanQuoteToken(runes, result, i, isHad, runeType(r))
	case isLineCommentStart(runes, i):
		return skipLineComment(runes, i), hasWhitespace(true), true
	case isBlockCommentStart(runes, i):
		return skipBlockComment(runes, i), hasWhitespace(true), true
	default:
		return 0, false, false
	}
}

// scanDollarToken writes a dollar-quoted literal, or reports not-handled when the
// dollar sign doesn't actually open a valid dollar quote.
func scanDollarToken(
	runes []rune,
	result *strings.Builder,
	i runeIndex,
	isHad hasWhitespace,
) (runeIndex, hasWhitespace, bool) {
	content, length := scanDollarQuote(runes, i)
	if length == 0 {
		return 0, false, false
	}
	return writeLiteral(runes, result, i, isHad, content, length), hasWhitespace(false), true
}

// scanQuoteToken writes a single- or double-quoted literal.
func scanQuoteToken(
	runes []rune,
	result *strings.Builder,
	i runeIndex,
	isHad hasWhitespace,
	quote runeType,
) (runeIndex, hasWhitespace, bool) {
	content, length := scanString(runes, i, quote)
	return writeLiteral(runes, result, i, isHad, content, length), hasWhitespace(false), true
}

// writeLiteral appends a scanned literal, adding a leading space if we need one,
// and returns the position right after the literal.
func writeLiteral(
	runes []rune,
	result *strings.Builder,
	i runeIndex,
	isHad hasWhitespace,
	content quotedString,
	length runeCount,
) runeIndex {
	idx := int(i)
	addSpaceIfNeeded(result, isHad, runeType(runes[idx]))
	emit(result, emittedText(string(content)))
	return runeIndex(idx + int(length))
}

// isLineCommentStart says whether a line comment (-- or #) begins at idx.
func isLineCommentStart(runes []rune, idx runeIndex) bool {
	if runes[int(idx)] == '#' {
		return true
	}
	return int(idx)+1 < len(runes) && runes[int(idx)] == '-' && runes[int(idx)+1] == '-'
}

// skipLineComment returns the position past the line comment at idx, newline and
// all.
func skipLineComment(runes []rune, idx runeIndex) runeIndex {
	i := int(idx)
	for i < len(runes) && runes[i] != '\n' {
		i++
	}
	if i < len(runes) {
		i++
	}
	return runeIndex(i)
}

// isBlockCommentStart says whether a block comment opens at idx.
func isBlockCommentStart(runes []rune, idx runeIndex) bool {
	return int(idx)+1 < len(runes) && runes[int(idx)] == '/' && runes[int(idx)+1] == '*'
}

// commentDepth is the nesting depth inside a block comment.
type commentDepth int

// skipBlockComment returns the position past a possibly nested block comment.
func skipBlockComment(runes []rune, idx runeIndex) runeIndex {
	i := runeIndex(int(idx) + 2)
	depth := commentDepth(1)
	for int(i) < len(runes) && depth > 0 {
		i, depth = stepBlockComment(runes, i, depth)
	}
	return i
}

// stepBlockComment moves one step through a block comment, bumping nesting depth
// up or down when it hits an opening or closing delimiter.
func stepBlockComment(runes []rune, idx runeIndex, depth commentDepth) (runeIndex, commentDepth) {
	i := int(idx)
	if i+1 < len(runes) {
		switch {
		case runes[i] == '/' && runes[i+1] == '*':
			return runeIndex(i + 2), depth + 1
		case runes[i] == '*' && runes[i+1] == '/':
			return runeIndex(i + 2), depth - 1
		}
	}
	return runeIndex(i + 1), depth
}
