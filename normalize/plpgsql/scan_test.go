package plpgsql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDollarTagEndRejectsAMalformedTag names dollarTagEnd's documented failure
// answers: a tag holding a character that cannot appear in one, and a tag that
// never terminates before the input ends.
func TestDollarTagEndRejectsAMalformedTag(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input string
	}{
		{name: "illegal tag character", input: "$ab!$"},
		{name: "tag never terminates", input: "$abc"},
		{name: "bare dollar at end", input: "$"},
	} {
		_, ok := dollarTagEnd([]rune(tc.input), 0)
		assert.False(t, ok, tc.name)
	}

	end, ok := dollarTagEnd([]rune("$tag$body$tag$"), 0)
	require.True(t, ok)
	assert.Equal(t, 4, end, "the index of the tag's closing dollar")
}

// TestFindClosingTagReportsAnAbsentTag names findClosingTag's documented -1: a
// dollar quote whose closing tag never shows up again is unterminated, and
// reporting a positive index there would slice past the literal's real end.
func TestFindClosingTagReportsAnAbsentTag(t *testing.T) {
	t.Parallel()
	runes := []rune("$a$ body with no close")

	assert.Equal(t, -1, findClosingTag(runes, 3, "$a$"))
	assert.Equal(t, 12, findClosingTag([]rune("$a$ body $a$"), 3, "$a$"), "just past the closing tag")
}

// TestScanStringEatsAnUnclosedLiteral names scanString's documented behaviour at
// the end of input. Consuming the remainder is what stops an unterminated quote
// from being rescanned as ordinary tokens, which would let the text inside a
// broken literal be reinterpreted as SQL.
func TestScanStringEatsAnUnclosedLiteral(t *testing.T) {
	t.Parallel()
	runes := []rune("'unterminated")

	literal, count := scanString(runes, 0, runeType('\''))

	assert.Equal(t, quotedString("'unterminated"), literal)
	assert.Equal(t, runeCount(len(runes)), count, "every remaining rune is consumed")
}

// TestNormalizeQuotedSpans covers quoted strings and dollar-quoted bodies —
// the spans whose contents must survive untouched. A boundary mistake here
// reinterprets the inside of a literal as code.
func TestNormalizeQuotedSpans(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input Body
		want  string
	}{
		{
			name: "single-quoted string",
			input: `begin
  new.host := 'example.com';
  return new;
end;`,
			want: "begin new.host := 'example.com'; return new; end",
		},
		{
			name: "double-quoted identifier",
			input: `begin
  new."ColumnName" := 'value';
  return new;
end;`,
			want: `begin new."ColumnName" := 'value'; return new; end`,
		},
		{
			name: "string with escaped quote",
			input: `begin
  new.text := 'It''s a string';
  return new;
end;`,
			want: "begin new.text := 'It''s a string'; return new; end",
		},
		{
			name: "string with backslash escape",
			input: `begin
  new.path := 'C:\\path\\to\\file';
  return new;
end;`,
			want: `begin new.path := 'C:\\path\\to\\file'; return new; end`,
		},
		{
			name: "string containing --",
			input: `begin
  new.text := 'This has -- dashes';
  return new;
end;`,
			want: "begin new.text := 'This has -- dashes'; return new; end",
		},
		{
			name: "string containing /* */",
			input: `begin
  new.text := 'This has /* block */ comment syntax';
  return new;
end;`,
			want: "begin new.text := 'This has /* block */ comment syntax'; return new; end",
		},
		{
			name:  "unterminated string consumes to end",
			input: "begin new.x := 'unterminated; end",
			want:  "begin new.x := 'unterminated; end",
		},
		{
			name: "string with newlines",
			input: `begin
  new.text := 'line1
line2
line3';
  return new;
end;`,
			want: "begin new.text := 'line1\nline2\nline3'; return new; end",
		},

		// Dollar quotes
		{
			name: "simple dollar quote",
			input: `begin
  new.body := $$SELECT * FROM table$$;
  return new;
end;`,
			want: "begin new.body := $$SELECT * FROM table$$; return new; end",
		},
		{
			name: "dollar quote with tag",
			input: `begin
  new.body := $tag$SELECT * FROM table$tag$;
  return new;
end;`,
			want: "begin new.body := $tag$SELECT * FROM table$tag$; return new; end",
		},
		{
			name: "dollar quote containing --",
			input: `begin
  new.body := $$
    SELECT * FROM table
    -- This is in a dollar quote
    WHERE id = 1
  $$;
  return new;
end;`,
			want: "begin new.body := $$\n    SELECT * FROM table\n    -- This is in a dollar quote\n    WHERE id = 1\n  $$; return new; end",
		},
		{
			name: "dollar quote containing /* */",
			input: `begin
  new.body := $$
    SELECT * FROM table
    /* This is in a dollar quote */
    WHERE id = 1
  $$;
  return new;
end;`,
			want: "begin new.body := $$\n    SELECT * FROM table\n    /* This is in a dollar quote */\n    WHERE id = 1\n  $$; return new; end",
		},
		{
			name: "nested dollar quotes with different tags",
			input: `begin
  new.body := $outer$
    This is outer
    $inner$This is inner$inner$
    Back to outer
  $outer$;
  return new;
end;`,
			want: "begin new.body := $outer$\n    This is outer\n    $inner$This is inner$inner$\n    Back to outer\n  $outer$; return new; end",
		},
		{
			name:  "dollar quote with underscores in tag",
			input: "begin new.x := $my_tag$content$my_tag$; return new; end",
			want:  "begin new.x := $my_tag$content$my_tag$; return new; end",
		},
		{
			name:  "numeric dollar quote tag is valid",
			input: "begin new.x := $123$invalid$123$; return new; end",
			want:  "begin new.x := $123$invalid$123$; return new; end",
		},
		{
			name:  "unterminated dollar quote treated as regular text",
			input: "begin new.x := $$unterminated; return new; end",
			want:  "begin new.x := $$unterminated; return new; end",
		},
		{
			name:  "dollar with non-tag char is plain dollar",
			input: "$ x",
			want:  "$ x",
		},
		{
			name:  "dollar with tag chars but no closing dollar at end",
			input: "$abc",
			want:  "$abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, string(normalize(tt.input)))
		})
	}
}
