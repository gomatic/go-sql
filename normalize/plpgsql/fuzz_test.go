package plpgsql

import "testing"

// bodySeeds walks the edge matrix for PL/pgSQL normalization: empty,
// whitespace-only, comment-only, every quote shape (single, double, dollar,
// tagged, nested), escapes, unterminated literals and comments, operators,
// scientific notation, and unicode.
var bodySeeds = []string{
	"",
	"   \t\n ",
	"-- only comment",
	"begin new.x := now(); return new; end;",
	"begin /* unterminated comment",
	"begin new.x := 'unterminated; end",
	"begin new.body := $tag$ a $b$ c $tag$; end",
	"begin new.x := $$unterminated; end",
	"begin new.text := 'It''s a string'; end",
	"begin new.path := 'C:\\x\\y'; end",
	"begin new.x:=a+b-c*d/e%f::integer; end",
	"begin new.x := 1.5e-10; new.y := 2e+5; end",
	"begin new.名前 := '你好世界🎉'; end",
	"$abc",
	"$ x",
	"begin perform f(a,b,); end",
}

// FuzzNormalize asserts PL/pgSQL normalization never panics and is idempotent:
// the canonical form of an already-canonical body is itself. The package promises
// a deterministic result for every input with no error path, so idempotence must
// hold universally.
func FuzzNormalize(f *testing.F) {
	for _, s := range bodySeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		once := Body(in).Normalize()
		twice := once.Normalize()
		if once != twice {
			t.Fatalf("Normalize not idempotent: %q -> %q -> %q", in, once, twice)
		}
	})
}
