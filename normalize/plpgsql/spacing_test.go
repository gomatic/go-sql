package plpgsql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetLastRuneOnNonEmptyText names the invariant getLastRune documents. Its
// body indexes runes[len-1] unguarded, so "s is never empty" is not a remark —
// it is the precondition standing between this package and a panic on attacker-
// supplied SQL. The claim is only true because addSpaceIfNeeded returns early on
// an empty builder, which is asserted below.
func TestGetLastRuneOnNonEmptyText(t *testing.T) {
	t.Parallel()
	assert.Equal(t, runeType('c'), getLastRune("abc"))
	assert.Equal(t, runeType('x'), getLastRune("x"))
	assert.Equal(t, runeType('é'), getLastRune("café"), "the final RUNE, not the final byte")
}

// TestAddSpaceIfNeededGuardsTheEmptyBuilder is the other half: it pins the guard
// that makes getLastRune's precondition hold. Deleting the result.Len() == 0
// early return turns this into a panic, which is exactly the failure the doc
// comment asserts cannot happen.
func TestAddSpaceIfNeededGuardsTheEmptyBuilder(t *testing.T) {
	t.Parallel()
	var result strings.Builder

	assert.NotPanics(t, func() { addSpaceIfNeeded(&result, hasWhitespace(true), runeType('a')) })
	assert.Zero(t, result.Len(), "nothing may be emitted before the first token")
}

// TestNormalizeSpacing covers the whitespace and operator spacing rules: what
// collapses, what is inserted, and what must be left alone. These decide the
// shape of the output for code that is already correct, so a wrong answer here
// is a diff on every body the normalizer touches.
func TestNormalizeSpacing(t *testing.T) {
	tests := []struct {
		name  string
		input Body
		want  string
	}{
		{
			name: "multiple spaces collapsed",
			input: `begin
  new.registered    :=    now();
  return     new;
end;`,
			want: "begin new.registered := now(); return new; end",
		},
		{
			name:  "tabs replaced with spaces",
			input: "begin\n\tnew.registered\t:=\tnow();\n\treturn\tnew;\nend;",
			want:  "begin new.registered := now(); return new; end",
		},
		{
			name: "multiple newlines collapsed",
			input: `begin


  new.registered := now();


  return new;


end;`,
			want: "begin new.registered := now(); return new; end",
		},
		{
			name:  "mixed whitespace",
			input: "begin\n\t  \t\n  new.registered := now();\n\t  \t\n  return new;\nend;",
			want:  "begin new.registered := now(); return new; end",
		},
		{
			name:  "leading and trailing whitespace removed",
			input: "  \n\t  begin\n  return new;\nend;  \n\t  ",
			want:  "begin return new; end",
		},

		// Operators
		{
			name:  "assignment operator with spaces",
			input: "begin new.x := 5; return new; end",
			want:  "begin new.x := 5; return new; end",
		},
		{
			name:  "assignment operator without spaces",
			input: "begin new.x:=5; return new; end",
			want:  "begin new.x := 5; return new; end",
		},
		{
			name:  "assignment operator mixed spacing",
			input: "begin new.x :=5; return new; end",
			want:  "begin new.x := 5; return new; end",
		},
		{
			name:  "comparison operators",
			input: "begin if x=5 and y<>3 and z>=10 then return true; end if; end",
			want:  "begin if x = 5 and y <> 3 and z >= 10 then return true; end if; end",
		},
		{
			name:  "arrow operator",
			input: "begin new.data:=payload->'key'; return new; end",
			want:  "begin new.data := payload -> 'key'; return new; end",
		},
		{
			name:  "concat operator",
			input: "begin new.text:=a||b||c; return new; end",
			want:  "begin new.text := a || b || c; return new; end",
		},
		{
			name:  "cast operator",
			input: "begin new.val:=x::integer; return new; end",
			want:  "begin new.val := x :: integer; return new; end",
		},
		{
			name:  "multiple operators",
			input: "begin new.x:=a+b-c*d/e%f; return new; end",
			want:  "begin new.x := a + b - c * d / e % f; return new; end",
		},

		// Punctuation
		{
			name:  "function calls",
			input: "begin new.x := now( ); return new; end",
			want:  "begin new.x := now(); return new; end",
		},
		{
			name:  "comma spacing",
			input: "begin perform func(a,b,c); return new; end",
			want:  "begin perform func(a, b, c); return new; end",
		},
		{
			name:  "separator before closing paren keeps no space",
			input: "begin perform func(a,); return new; end",
			want:  "begin perform func(a,); return new; end",
		},
		{
			name:  "semicolon spacing",
			input: "begin new.x := 1;new.y := 2;return new; end",
			want:  "begin new.x := 1; new.y := 2; return new; end",
		},
		{
			name:  "parentheses spacing",
			input: "begin if ( x = 1 ) then return true; end if; end",
			want:  "begin if (x = 1) then return true; end if; end",
		},
		{
			name:  "opening bracket",
			input: "begin new.x := arr[1]; return new; end",
			want:  "begin new.x := arr[1]; return new; end",
		},
		{
			name:  "opening brace",
			input: "begin new.x := {a,b}; return new; end",
			want:  "begin new.x := {a, b}; return new; end",
		},

		// Numbers
		{
			name:  "integer",
			input: "begin new.x := 42; return new; end",
			want:  "begin new.x := 42; return new; end",
		},
		{
			name:  "decimal",
			input: "begin new.x := 3.14; return new; end",
			want:  "begin new.x := 3.14; return new; end",
		},
		{
			name:  "scientific notation",
			input: "begin new.x := 1.5e10; return new; end",
			want:  "begin new.x := 1.5e10; return new; end",
		},
		{
			name:  "scientific notation with positive exponent",
			input: "begin new.x := 2e+5; return new; end",
			want:  "begin new.x := 2e+5; return new; end",
		},
		{
			name:  "scientific notation with negative exponent",
			input: "begin new.x := 1.2e-3; return new; end",
			want:  "begin new.x := 1.2e-3; return new; end",
		},

		// Identifiers
		{
			name:  "identifier starting with number gets separated",
			input: "begin 1invalid := 5; return new; end",
			want:  "begin 1 invalid := 5; return new; end",
		},
		{
			name:  "identifier with underscores",
			input: "begin _private_var := 5; return new; end",
			want:  "begin _private_var := 5; return new; end",
		},
		{
			name:  "identifier with digits",
			input: "begin var123 := 5; return new; end",
			want:  "begin var123 := 5; return new; end",
		},

		// Unicode
		{
			name: "Unicode in strings",
			input: `begin
  new.text := '你好世界';
  return new;
end;`,
			want: "begin new.text := '你好世界'; return new; end",
		},
		{
			name: "Unicode in identifiers",
			input: `begin
  new.名前 := 'value';
  return new;
end;`,
			want: "begin new.名前 := 'value'; return new; end",
		},
		{
			name: "Emoji in strings",
			input: `begin
  new.text := '🎉🎊🎈';
  return new;
end;`,
			want: "begin new.text := '🎉🎊🎈'; return new; end",
		},
		{
			name: "Mixed Unicode and ASCII",
			input: `begin
  new.text := 'Hello 世界 World';
  return new;
end;`,
			want: "begin new.text := 'Hello 世界 World'; return new; end",
		},

		// Real-world examples
		{
			name: "it_registry function from manifest",
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
		{
			name: "ut_registry function from manifest",
			input: `begin
  new.registered := old.registered;
  new.host := old.host;
  new.organization := old.organization;
  new.repository := old.repository;
  new.pseudo := old.pseudo;
  return new;
end;`,
			want: "begin new.registered := old.registered; new.host := old.host; new.organization := old.organization; new.repository := old.repository; new.pseudo := old.pseudo; return new; end",
		},

		// Edge cases
		{
			// Every trailing semicolon (each a meaningless empty statement) is
			// stripped in one pass; stripping only the last would leave ";" for a
			// second pass, breaking idempotence.
			name:  "multiple trailing semicolons collapse fully",
			input: ";;",
			want:  "",
		},
		{
			name: "minus operator vs comment",
			input: `begin
  new.x := a - b; -- This is minus, not comment start
  return new;
end;`,
			want: "begin new.x := a - b; return new; end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, string(normalize(tt.input)))
		})
	}
}
