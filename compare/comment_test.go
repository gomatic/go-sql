package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type commentIdentityCase struct {
	name     testName
	sql      sqlStatement
	expected identity
}

func TestCommentIdentity(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []commentIdentityCase{
		{
			name:     "comment_on_column",
			sql:      "COMMENT ON COLUMN my_schema.my_table.my_col IS 'description'",
			expected: "comment.column:my_schema.my_table.my_col",
		},
		{
			name:     "comment_on_schema",
			sql:      "COMMENT ON SCHEMA my_schema IS 'description'",
			expected: "comment.schema:my_schema",
		},
		{
			name:     "comment_on_table",
			sql:      "COMMENT ON TABLE my_schema.my_table IS 'description'",
			expected: "comment.table:my_schema.my_table",
		},
		{
			name:     "comment_on_function",
			sql:      "COMMENT ON FUNCTION my_func() IS 'description'",
			expected: "comment.function:my_func",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, commentIdentity(parseTestSQL(t, must, tt.sql)))
		})
	}
}

func TestCommentIdentity_NilStatement(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Equal(identity(""), commentIdentity(nil))
}

type commentCompareCase struct {
	name        testName
	sourceSQL   sqlStatement
	targetSQL   sqlStatement
	expectDiffs expectBool
}

func TestCommentDiff_SmartTags(t *testing.T) {
	t.Parallel()
	want, must := assert.New(t), require.New(t)

	tests := []commentCompareCase{
		{
			name:        "different_order_same_tags",
			sourceSQL:   "COMMENT ON TABLE t IS '@omit create\n@name foo'",
			targetSQL:   "COMMENT ON TABLE t IS '@name foo\n@omit create'",
			expectDiffs: false,
		},
		{
			name:        "different_smart_tags",
			sourceSQL:   "COMMENT ON TABLE t IS '@omit create'",
			targetSQL:   "COMMENT ON TABLE t IS '@omit delete'",
			expectDiffs: true,
		},
		{
			name:        "identical_smart_tags",
			sourceSQL:   "COMMENT ON TABLE t IS '@omit create,delete'",
			targetSQL:   "COMMENT ON TABLE t IS '@omit create,delete'",
			expectDiffs: false,
		},
		{
			name:        "ignore_title_description",
			sourceSQL:   "COMMENT ON TABLE t IS 'title: foo\ndescription: bar'",
			targetSQL:   "COMMENT ON TABLE t IS 'title: baz\ndescription: qux'",
			expectDiffs: false,
		},
		{
			name:        "no_smart_tags_equal",
			sourceSQL:   "COMMENT ON TABLE t IS 'just a comment'",
			targetSQL:   "COMMENT ON TABLE t IS 'different comment'",
			expectDiffs: false,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			diffs := commentDiff(parseTestSQL(t, must, tt.sourceSQL), parseTestSQL(t, must, tt.targetSQL))
			if tt.expectDiffs {
				want.NotEmpty(diffs)
			} else {
				want.Empty(diffs)
			}
		})
	}
}

func TestReplaceCommentWithSmartTags_NoComment(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	// A normalized statement whose data has no comment field is left alone.
	stmt := statementData{"stmt": map[string]any{"data": map[string]any{}}}
	replaceCommentWithSmartTags(stmt)
	data := extractMap(extractMap(stmt, keyStmt), keyData)
	_, hasComment := data[string(keyComment)]
	want.False(hasComment)
}

type smartTagCase struct {
	name     testName
	comment  commentText
	expected []string
}

func TestExtractSmartTags(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []smartTagCase{
		{
			name:     "multiple_tags",
			comment:  "@omit create\n@name foo\n@unique id",
			expected: []string{"@omit create", "@name foo", "@unique id"},
		},
		{name: "no_smart_tags", comment: "just a regular comment", expected: nil},
		{name: "single_tag", comment: "@omit create,delete", expected: []string{"@omit create,delete"}},
		{name: "text_before_first_at", comment: "some text before @omit create", expected: []string{"@omit create"}},
		{
			name:     "title_description_filtered",
			comment:  "@title: foo\n@description: bar\n@omit create",
			expected: []string{"@omit create"},
		},
		{name: "empty_tag_skipped", comment: "@@omit create", expected: []string{"@omit create"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, extractSmartTags(tt.comment))
		})
	}
}

type tagNameCase struct {
	name     testName
	part     tagPart
	expected tagName
}

func TestExtractTagName(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []tagNameCase{
		{name: "description", part: "description: baz", expected: "description"},
		{name: "name", part: "name foo", expected: "name"},
		{name: "omit", part: "omit create", expected: "omit"},
		{name: "title", part: "title: bar", expected: "title"},
		{name: "unique", part: "unique", expected: "unique"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, extractTagName(tt.part))
		})
	}
}

type tagWhitespaceCase struct {
	name     testName
	tag      tagPart
	expected normalizedTag
}

func TestNormalizeTagWhitespace(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []tagWhitespaceCase{
		{name: "multiline", tag: "name foo\nsome extra", expected: "name foo"},
		{name: "simple", tag: "omit create", expected: "omit create"},
		{name: "spaced", tag: "  spaced  ", expected: "spaced"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, normalizeTagWhitespace(tt.tag))
		})
	}
}

func TestExtractCommentObjectName_Fallbacks(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	want.Equal(qualifiedName(""), extractCommentObjectName(statementData{}), "no object node yields no name")

	// A node whose child shape we don't recognize gives back no name.
	node := statementData{"object": map[string]any{"node": map[string]any{"unknown": map[string]any{}}}}
	want.Equal(qualifiedName(""), extractCommentObjectName(node))
}

func TestMapCommentObjectType(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)
	want.Equal(commentObjectType("table"), mapCommentObjectType(dropObjectTable))
	want.Equal(commentObjectType("schema"), mapCommentObjectType(dropObjectSchema))
}
