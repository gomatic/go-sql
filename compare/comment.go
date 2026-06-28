package compare

import (
	"slices"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

type (
	commentObjectType string   // commentObjectType is the snake_case kind of object a COMMENT targets.
	commentText       = string // commentText is raw comment text.
	normalizedTag     string   // normalizedTag is a whitespace-normalized smart tag body.
	smartTagsText     = string // smartTagsText is the sorted smart tags joined into one string.
	tagName           string   // tagName is the leading name of a smart tag.
	tagPart           string   // tagPart is a fragment produced by splitting a comment on '@'.
)

// Presentation smart tags we leave out of the comparison.
const (
	smartTagDescription = "description"
	smartTagTitle       = "title"
)

// ignoredSmartTags are the PostGraphile tags we skip when comparing: they're
// about presentation, not behavior. Read-only once it's set up.
var ignoredSmartTags = map[string]struct{}{
	smartTagDescription: {},
	smartTagTitle:       {},
}

// commentIdentity identifies a COMMENT by object kind and name — for example,
// comment.table:schema.table.
func commentIdentity(stmt statementData) identity {
	data := extractStatementData(stmt)
	if data == nil {
		return ""
	}
	objType, _ := extractInt(data, "objtype")
	kind := mapCommentObjectType(objectTypeInt(objType))
	name := extractCommentObjectName(data)
	return identity(identityPrefixComment + identityPrefix(kind) + ":" + identityPrefix(name))
}

// commentDiff compares two COMMENT statements by their PostGraphile smart tags
// alone — free-form comment text and tag order don't count.
func commentDiff(source, target statementData) statementDiffs {
	return computeDiffs(normalizeCommentForComparison(source), normalizeCommentForComparison(target))
}

// normalizeCommentForComparison boils a COMMENT's text down to its sorted smart
// tags.
func normalizeCommentForComparison(stmt statementData) statementData {
	normalized := normalizeStatement(stmt)
	replaceCommentWithSmartTags(normalized)
	return normalized
}

// replaceCommentWithSmartTags rewrites the comment field in place to its sorted
// smart tags, and leaves comments that have no smart tags alone.
func replaceCommentWithSmartTags(stmt statementData) {
	data := extractMap(extractMap(stmt, keyStmt), keyData)
	comment := extractString(data, keyComment)
	if comment == "" {
		return
	}
	data[string(keyComment)] = extractSortedSmartTags(commentText(comment))
}

// extractSortedSmartTags returns a comment's smart tags, sorted and joined
// together.
func extractSortedSmartTags(comment commentText) smartTagsText {
	tags := extractSmartTags(comment)
	slices.Sort(tags)
	return strings.Join(tags, "\n")
}

// extractSmartTags returns the @-prefixed smart tags in a comment, minus the
// presentation tags we ignore.
func extractSmartTags(comment commentText) []string {
	parts := strings.Split(comment, "@")
	if len(parts) <= 1 {
		return nil
	}
	tags := make([]string, 0, len(parts)-1)
	for _, part := range parts[1:] {
		if tag, ok := smartTag(tagPart(part)); ok {
			tags = append(tags, tag)
		}
	}
	return tags
}

// smartTag normalizes one '@'-split fragment into a smart tag, and tells you
// whether it's one we keep — non-empty and not ignored.
func smartTag(part tagPart) (string, bool) {
	trimmed := tagPart(strings.TrimSpace(string(part)))
	if trimmed == "" {
		return "", false
	}
	if _, ignored := ignoredSmartTags[string(extractTagName(trimmed))]; ignored {
		return "", false
	}
	return "@" + string(normalizeTagWhitespace(trimmed)), true
}

// extractTagName returns the lower-cased leading name of a tag fragment — the
// part up to the first colon or space.
func extractTagName(part tagPart) tagName {
	s := string(part)
	end := len(s)
	if colon := strings.IndexByte(s, ':'); colon >= 0 {
		end = colon
	}
	if space := strings.IndexByte(s, ' '); space >= 0 && space < end {
		end = space
	}
	return tagName(strings.ToLower(s[:end]))
}

// normalizeTagWhitespace trims a tag down to its first line.
func normalizeTagWhitespace(tag tagPart) normalizedTag {
	first, _, _ := strings.Cut(string(tag), "\n")
	return normalizedTag(strings.TrimSpace(first))
}

// mapCommentObjectType turns a pg_query ObjectType into a snake_case kind.
func mapCommentObjectType(t objectTypeInt) commentObjectType {
	name := strings.TrimPrefix(pg_query.ObjectType(t).String(), "OBJECT_")
	return commentObjectType(toSnakeCase(stringValue(name)))
}

// extractCommentObjectName reads the qualified name of the commented object.
func extractCommentObjectName(data statementData) qualifiedName {
	node := extractMap(extractMap(data, "object"), keyNode)
	if node == nil {
		return ""
	}
	if str := extractMap(node, keyStringNode); str != nil {
		return qualifiedName(extractString(str, keySval))
	}
	if owa := extractMap(node, keyObjWithArgs); owa != nil {
		return extractFunctionDropName(owa)
	}
	if list := extractMap(node, keyList); list != nil {
		return extractListName(list)
	}
	return ""
}
