package plpgsql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
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

		// Whitespace
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
