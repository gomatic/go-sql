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

// emit appends s to result. strings.Builder writes are documented never to fail,
// so we swallow the (always-nil) error in this one spot.
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

// spacingContext is the immutable input we feed to the spacing rules.
type spacingContext struct {
	last     runeType
	penult   runeType
	curr     runeType
	hasSpace hasWhitespace
}

// spacingRules is the ordered set of rules we consult to decide whether a space
// goes between the rune we last wrote and the current one.
var spacingRules = []func(spacingContext) spaceDecision{
	ruleAfterSeparator,
	ruleAfterOpening,
	ruleBeforeClosing,
	ruleAroundDot,
	ruleScientific,
	ruleOperatorBoundary,
	ruleDigitLetter,
}

// addSpaceIfNeeded writes a separating space before curr when the spacing rules
// ask for one.
func addSpaceIfNeeded(result *strings.Builder, isHad hasWhitespace, curr runeType) {
	if result.Len() == 0 {
		return
	}
	written := normalizedText(result.String())
	ctx := spacingContext{
		last:     getLastRune(written),
		penult:   getPenultimateRune(written),
		curr:     curr,
		hasSpace: isHad,
	}
	if spaceWanted(ctx) {
		emit(result, emittedText(" "))
	}
}

// spaceWanted runs each rule in order and returns the first decisive answer; if
// none of them decide, it keeps whatever whitespace was originally there.
func spaceWanted(ctx spacingContext) bool {
	for _, rule := range spacingRules {
		if d := rule(ctx); d != spaceUndecided {
			return d == spaceYes
		}
	}
	return bool(ctx.hasSpace)
}

// ruleAfterSeparator forces a space after a comma or semicolon, unless the next
// rune closes a group or is itself a separator.
func ruleAfterSeparator(ctx spacingContext) spaceDecision {
	if ctx.last != runeType(',') && ctx.last != runeType(';') {
		return spaceUndecided
	}
	if isCloseOrSeparator(ctx.curr) {
		return spaceNo
	}
	return spaceYes
}

// ruleAfterOpening holds back a space right after an opening bracket.
func ruleAfterOpening(ctx spacingContext) spaceDecision {
	if ctx.last == runeType('(') || ctx.last == runeType('[') || ctx.last == runeType('{') {
		return spaceNo
	}
	return spaceUndecided
}

// ruleBeforeClosing holds back a space before a closing bracket or separator.
func ruleBeforeClosing(ctx spacingContext) spaceDecision {
	if isCloseOrSeparator(ctx.curr) {
		return spaceNo
	}
	return spaceUndecided
}

// ruleAroundDot holds back a space on either side of a dot.
func ruleAroundDot(ctx spacingContext) spaceDecision {
	if ctx.last == runeType('.') || ctx.curr == runeType('.') {
		return spaceNo
	}
	return spaceUndecided
}

// ruleScientific holds back spaces inside scientific-notation exponents like e+5
// or e-3.
func ruleScientific(ctx spacingContext) spaceDecision {
	if isExponentSign(ctx.last, ctx.curr) || isSignedExponentDigit(ctx) {
		return spaceNo
	}
	return spaceUndecided
}

// ruleOperatorBoundary forces a space wherever we cross between operator and
// non-operator characters.
func ruleOperatorBoundary(ctx spacingContext) spaceDecision {
	if bool(isOperatorChar(ctx.last)) != bool(isOperatorChar(ctx.curr)) {
		return spaceYes
	}
	return spaceUndecided
}

// ruleDigitLetter forces a space between a digit and a following letter, except
// for the exponent marker of scientific notation.
func ruleDigitLetter(ctx spacingContext) spaceDecision {
	notExponent := ctx.curr != runeType('e') && ctx.curr != runeType('E')
	if unicode.IsDigit(rune(ctx.last)) && unicode.IsLetter(rune(ctx.curr)) && notExponent {
		return spaceYes
	}
	return spaceUndecided
}

// isCloseOrSeparator says whether r closes a group or separates items.
func isCloseOrSeparator(r runeType) bool {
	switch rune(r) {
	case ')', ']', '}', ',', ';':
		return true
	default:
		return false
	}
}

// isExponentSign says whether last is an exponent marker and curr its sign.
func isExponentSign(last, curr runeType) bool {
	isE := last == runeType('e') || last == runeType('E')
	isSign := curr == runeType('+') || curr == runeType('-')
	return isE && isSign
}

// isSignedExponentDigit says whether curr is a digit sitting after an exponent
// sign that itself sits after an exponent marker — the digits of e+5.
func isSignedExponentDigit(ctx spacingContext) bool {
	signed := ctx.last == runeType('+') || ctx.last == runeType('-')
	if !signed || !unicode.IsDigit(rune(ctx.curr)) {
		return false
	}
	return ctx.penult == runeType('e') || ctx.penult == runeType('E')
}

// isOperatorChar says whether r is one of the operator characters we recognize.
func isOperatorChar(r runeType) isOperator {
	switch rune(r) {
	case ':', '=', '<', '>', '!', '+', '-', '*', '/', '%', '|', '&', '^', '~':
		return isOperator(true)
	default:
		return isOperator(false)
	}
}

// getLastRune returns the final rune of s. Callers only hit it with non-empty
// written text — addSpaceIfNeeded guards on result length — so s is never empty.
func getLastRune(s normalizedText) runeType {
	runes := []rune(string(s))
	return runeType(runes[len(runes)-1])
}

// getPenultimateRune returns the second-to-last rune of s, or rune zero when s has
// fewer than two runes.
func getPenultimateRune(s normalizedText) runeType {
	runes := []rune(string(s))
	if len(runes) < 2 {
		return runeType(0)
	}
	return runeType(runes[len(runes)-2])
}

// scanDollarQuote scans a dollar-quoted literal ($$…$$ or $tag$…$tag$) and returns
// the literal plus how many runes it consumed, or empty/zero when start doesn't
// open a valid dollar quote.
func scanDollarQuote(runes []rune, start runeIndex) (quotedString, runeCount) {
	startIdx := int(start)
	tagEnd, ok := dollarTagEnd(runes, start)
	if !ok {
		return quotedString(""), runeCount(0)
	}

	tag := string(runes[startIdx : tagEnd+1])
	end := findClosingTag(runes, runeIndex(tagEnd+1), dollarTag(tag))
	if end < 0 {
		return quotedString(""), runeCount(0)
	}

	return quotedString(string(runes[startIdx:end])), runeCount(end - startIdx)
}

// dollarTagEnd returns the index of the closing $ of a dollar-quote opening tag,
// or false when the tag is malformed or never terminates.
func dollarTagEnd(runes []rune, startIdx runeIndex) (int, bool) {
	tagEnd := int(startIdx) + 1
	for tagEnd < len(runes) && runes[tagEnd] != '$' {
		if !isTagChar(runeType(runes[tagEnd])) {
			return 0, false
		}
		tagEnd++
	}
	if tagEnd >= len(runes) {
		return 0, false
	}
	return tagEnd, true
}

// isTagChar says whether r is allowed inside a dollar-quote tag.
func isTagChar(r runeType) bool {
	return unicode.IsLetter(rune(r)) || unicode.IsDigit(rune(r)) || rune(r) == '_'
}

// dollarTag is the delimiter tag of a dollar-quoted literal, dollar signs included ($tag$).
type dollarTag string

// findClosingTag returns the index just past the closing tag at or after from, or
// -1 if the tag never shows up again.
func findClosingTag(runes []rune, from runeIndex, tag dollarTag) int {
	tagLen := len([]rune(string(tag)))
	for i := int(from); i < len(runes); i++ {
		if i+tagLen <= len(runes) && string(runes[i:i+tagLen]) == string(tag) {
			return i + tagLen
		}
	}
	return -1
}

// scanString scans a single- or double-quoted literal, handling doubled-quote and
// backslash escapes. If the literal never closes, it eats the rest of the runes.
func scanString(runes []rune, start runeIndex, quote runeType) (quotedString, runeCount) {
	startIdx := int(start)

	for i := startIdx + 1; i < len(runes); {
		end, next := stringStep(runes, runeIndex(i), quote)
		if end >= 0 {
			return quotedString(string(runes[startIdx:end])), runeCount(end - startIdx)
		}
		i = next
	}

	return quotedString(string(runes[startIdx:])), runeCount(len(runes) - startIdx)
}

// stringStep looks at the rune at i. It returns a non-negative end index when the
// literal closes at i; otherwise end is -1 and next is where to pick back up.
func stringStep(runes []rune, i runeIndex, quote runeType) (end, next int) {
	if runes[int(i)] == rune(quote) {
		if isDoubledQuote(runes, i, quote) {
			return -1, int(i) + 2
		}
		return int(i) + 1, 0
	}
	if isBackslashEscape(runes, i, quote) {
		return -1, int(i) + 2
	}
	return -1, int(i) + 1
}

// isDoubledQuote says whether a doubled quote (an escaped quote) sits at i.
func isDoubledQuote(runes []rune, i runeIndex, quote runeType) bool {
	return int(i)+1 < len(runes) && runes[int(i)+1] == rune(quote)
}

// isBackslashEscape says whether a backslash escape sits at i inside a
// single-quoted literal.
func isBackslashEscape(runes []rune, i runeIndex, quote runeType) bool {
	return rune(quote) == '\'' && runes[int(i)] == '\\' && int(i)+1 < len(runes)
}
