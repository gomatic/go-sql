package plpgsql

import "unicode"

// Scanning the multi-rune token shapes: dollar quotes and quoted literals.
// These are the spans whose CONTENTS must survive normalization untouched, so
// every boundary decision here is the difference between preserving a literal
// and reinterpreting its text as code.

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
