package plpgsql

import (
	"strings"
	"unicode"
)

// The spacing rules: given the last rune written and the one about to be, decide
// whether a space belongs between them. Each rule answers for one situation and
// abstains otherwise, so the set composes — and every rule that abstains must
// keep abstaining, because a rule that answers outside its situation silently
// reshapes bodies it was never meant to touch.

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
