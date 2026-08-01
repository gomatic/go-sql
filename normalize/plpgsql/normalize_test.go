package plpgsql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNormalizeLexicalSpans covers the spans whose CONTENTS must survive
// normalization untouched: comments, quoted strings and dollar-quoted bodies.
// A boundary mistake here does not reformat code — it reinterprets the inside of
// a literal as code, or swallows code into a comment.
func TestNormalizeLexicalSpans(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input Body
		want  string
	}{
		// Empty / whitespace-only
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   \t\n\r   ",
			want:  "",
		},
		{
			name:  "only comments",
			input: "-- comment 1\n/* comment 2 */\n# comment 3",
			want:  "",
		},

		// Simple functions
		{
			name: "basic begin/end block",
			input: `begin
  new.registered := now();
  return new;
end;`,
			want: "begin new.registered := now(); return new; end",
		},
		{
			name: "function with if statements",
			input: `begin
  if new.host is null and not new.pseudo then
    new.host := 'git.example.com';
  end if;
  return new;
end;`,
			want: "begin if new.host is null and not new.pseudo then new.host := 'git.example.com'; end if; return new; end",
		},
		{
			name: "function with multiple if statements",
			input: `begin
  new.registered := now();
  if new.host is null and not new.pseudo then
    new.host := 'git.example.com';
  end if;
  if new.organization is null then
    new.organization := 'example-org';
  end if;
  if new.pseudo then
    new.host := '';
    new.organization := '';
  end if;
  return new;
end;`,
			want: "begin new.registered := now(); if new.host is null and not new.pseudo then new.host := 'git.example.com'; end if; if new.organization is null then new.organization := 'example-org'; end if; if new.pseudo then new.host := ''; new.organization := ''; end if; return new; end",
		},

		// Comments
		{
			name: "line comment with --",
			input: `begin
  -- This is a comment
  return new;
end;`,
			want: "begin return new; end",
		},
		{
			name: "line comment with #",
			input: `begin
  # This is a comment
  return new;
end;`,
			want: "begin return new; end",
		},
		{
			name: "block comment",
			input: `begin
  /* This is a block comment */
  return new;
end;`,
			want: "begin return new; end",
		},
		{
			name: "nested block comment",
			input: `begin
  /* outer /* inner */ outer */
  return new;
end;`,
			want: "begin return new; end",
		},
		{
			name:  "unterminated block comment consumes to end",
			input: "begin /* unterminated comment",
			want:  "begin",
		},
		{
			name: "multiple comments",
			input: `begin
  -- comment 1
  # comment 2
  /* block comment */
  return new;
end;`,
			want: "begin return new; end",
		},
		{
			name: "comment at end of line",
			input: `begin
  new.registered := now(); -- set timestamp
  return new; -- return modified
end;`,
			want: "begin new.registered := now(); return new; end",
		},

		// Strings
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, string(normalize(tt.input)))
		})
	}
}

// TestNormalizeMethod exercises the exported [Body.Normalize] wrapper.
func TestNormalizeMethod(t *testing.T) {
	t.Parallel()
	input := Body("begin\n  new.x := 1;\nend;")
	assert.Equal(t, Body("begin new.x := 1; end"), input.Normalize())
}

// TestNormalizeIdempotency checks that normalizing twice gives you the same result.
func TestNormalizeIdempotency(t *testing.T) {
	t.Parallel()
	inputs := []Body{
		`begin
  new.registered := now();
  return new;
end;`,
		`begin
  -- comment
  new.registered := now();
  /* block comment */
  return new;
end;`,
		`begin
  new.registered := now();
  if new.host is null and not new.pseudo then
    new.host := 'git.example.com';
  end if;
  return new;
end;`,
	}

	for _, input := range inputs {
		t.Run(string(input), func(t *testing.T) {
			t.Parallel()
			once := normalize(input)
			assert.Equal(t, once, normalize(once))
		})
	}
}

// TestEmitAppendsToTheBuilder names emit's contract. emit is the only place in
// this package that discards an error, so what it does with the text it is given
// is worth pinning: every fragment lands, in order, unaltered.
func TestEmitAppendsToTheBuilder(t *testing.T) {
	t.Parallel()
	var result strings.Builder

	emit(&result, "select")
	emit(&result, " ")
	emit(&result, "1")

	assert.Equal(t, "select 1", result.String())
}
