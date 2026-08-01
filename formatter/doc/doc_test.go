package doc_test

import (
	"testing"

	"github.com/gomatic/go-sql/formatter/doc"
)

func TestText(t *testing.T) {
	got := doc.Render(doc.Text("select"), 80)
	if got != "select" {
		t.Fatalf("got %q, want %q", got, "select")
	}
}

func TestConcat(t *testing.T) {
	got := doc.Render(doc.Concat(doc.Text("a"), doc.Text("b"), doc.Text("c")), 80)
	if got != "abc" {
		t.Fatalf("got %q, want %q", got, "abc")
	}
}

func TestGroupFitsStaysFlat(t *testing.T) {
	d := doc.Group(doc.Concat(doc.Text("a"), doc.Line(), doc.Text("b")))
	got := doc.Render(d, 80)
	if got != "a b" {
		t.Fatalf("got %q, want %q", got, "a b")
	}
}

func TestGroupTooWideBreaks(t *testing.T) {
	d := doc.Group(doc.Concat(doc.Text("aaaa"), doc.Line(), doc.Text("bbbb")))
	got := doc.Render(d, 5)
	if got != "aaaa\nbbbb" {
		t.Fatalf("got %q, want %q", got, "aaaa\nbbbb")
	}
}

func TestIndentAppliesAfterBreak(t *testing.T) {
	d := doc.Group(doc.Concat(doc.Text("select"), doc.Indent(doc.Concat(doc.Line(), doc.Text("x")))))
	got := doc.Render(d, 4)
	if got != "select\n  x" {
		t.Fatalf("got %q, want %q", got, "select\n  x")
	}
}

func TestSoftlineFlatIsEmpty(t *testing.T) {
	d := doc.Group(doc.Concat(doc.Text("("), doc.Softline(), doc.Text("x"), doc.Softline(), doc.Text(")")))
	got := doc.Render(d, 80)
	if got != "(x)" {
		t.Fatalf("got %q, want %q", got, "(x)")
	}
}

func TestHardlineForcesBreakEvenWhenFits(t *testing.T) {
	d := doc.Group(doc.Concat(doc.Text("a"), doc.Hardline(), doc.Text("b")))
	got := doc.Render(d, 80)
	if got != "a\nb" {
		t.Fatalf("got %q, want %q", got, "a\nb")
	}
}

// A nested group is laid out flat when the whole thing fits: deciding the outer
// group's fit walks into the inner group flat.
func TestNestedGroupsFlatWhenWholeFits(t *testing.T) {
	inner := doc.Group(doc.Concat(doc.Text("bb"), doc.Line(), doc.Text("cc")))
	d := doc.Group(doc.Concat(doc.Text("aa"), doc.Line(), inner))
	got := doc.Render(d, 8)
	if got != "aa bb cc" {
		t.Fatalf("got %q, want %q", got, "aa bb cc")
	}
}

// An indented child is walked while checking a group flat; when it fits, the
// indent is invisible.
func TestIndentFlatWhenFits(t *testing.T) {
	d := doc.Group(doc.Concat(doc.Text("a"), doc.Indent(doc.Concat(doc.Line(), doc.Text("b")))))
	got := doc.Render(d, 80)
	if got != "a b" {
		t.Fatalf("got %q, want %q", got, "a b")
	}
}

// Text sitting after a group, on the same line, counts against whether the group
// fits: it forces the break here, and rides the same line when it fits.
func TestTrailingTextCountsAgainstGroupFit(t *testing.T) {
	d := doc.Concat(doc.Group(doc.Concat(doc.Text("aa"), doc.Line(), doc.Text("bb"))), doc.Text("!!"))
	if got := doc.Render(d, 5); got != "aa\nbb!!" {
		t.Fatalf("width 5: got %q, want %q", got, "aa\nbb!!")
	}
	if got := doc.Render(d, 7); got != "aa bb!!" {
		t.Fatalf("width 7: got %q, want %q", got, "aa bb!!")
	}
}

// A hardline in the continuation ends the current line, so an inner group stays
// flat no matter how long the text after the hardline is.
func TestHardlineInContinuationLetsInnerStayFlat(t *testing.T) {
	inner := doc.Group(doc.Concat(doc.Text("xx"), doc.Line(), doc.Text("yy")))
	d := doc.Group(doc.Concat(inner, doc.Hardline(), doc.Text("zzzzzzzz")))
	got := doc.Render(d, 5)
	if got != "xx yy\nzzzzzzzz" {
		t.Fatalf("got %q, want %q", got, "xx yy\nzzzzzzzz")
	}
}

// A broken line in the continuation likewise ends the current line for an inner
// group's fit check.
func TestBrokenLineInContinuationLetsInnerStayFlat(t *testing.T) {
	inner := doc.Group(doc.Concat(doc.Text("a"), doc.Line(), doc.Text("b")))
	d := doc.Group(doc.Concat(inner, doc.Line(), doc.Text("XXXXXXXXXXXXXXXXXXXX")))
	got := doc.Render(d, 6)
	if got != "a b\nXXXXXXXXXXXXXXXXXXXX" {
		t.Fatalf("got %q, want %q", got, "a b\nXXXXXXXXXXXXXXXXXXXX")
	}
}

// Same as above, but with a broken softline in the continuation.
func TestBrokenSoftlineInContinuationLetsInnerStayFlat(t *testing.T) {
	inner := doc.Group(doc.Concat(doc.Text("a"), doc.Line(), doc.Text("b")))
	d := doc.Group(doc.Concat(inner, doc.Softline(), doc.Text("XXXXXXXXXXXXXXXXXXXX")))
	got := doc.Render(d, 6)
	if got != "a b\nXXXXXXXXXXXXXXXXXXXX" {
		t.Fatalf("got %q, want %q", got, "a b\nXXXXXXXXXXXXXXXXXXXX")
	}
}

// TestPushGroupBreaksOnAHardline names the invariant pushGroup documents: a
// group whose child carries a hardline ALWAYS breaks, however much room is left.
// A hardline is the caller's statement that a break is semantic — a line comment
// ends at a newline, so flattening one would swallow the SQL that follows it into
// the comment. Measuring the group and finding it fits must not override that.
func TestPushGroupBreaksOnAHardline(t *testing.T) {
	group := doc.Group(doc.Concat(doc.Text("a"), doc.Hardline(), doc.Text("b")))

	got := doc.Render(group, 200)

	if got != "a\nb" {
		t.Fatalf("a hardline group must break even at width 200: got %q, want %q", got, "a\nb")
	}
}

// TestPushGroupStaysFlatWhenItFits is the other half of pushGroup's claim —
// "otherwise it stays flat when it fits" — without which the test above is
// satisfied by a renderer that breaks everything unconditionally.
func TestPushGroupStaysFlatWhenItFits(t *testing.T) {
	group := doc.Group(doc.Concat(doc.Text("a"), doc.Line(), doc.Text("b")))

	if got := doc.Render(group, 200); got != "a b" {
		t.Fatalf("a fitting group must stay flat: got %q, want %q", got, "a b")
	}
	if got := doc.Render(group, 2); got != "a\nb" {
		t.Fatalf("a group that does not fit must break: got %q, want %q", got, "a\nb")
	}
}

// TestEveryDocKindIsHandledByBothWalks pins that the renderer and the fits
// measurement agree about every node kind. They are two switches over the same
// enum, and a kind handled by one but not the other is the sharpest kind of
// formatter bug: the measurement says a group fits, the renderer emits
// something wider, and the output silently exceeds the width it was asked for.
//
// The check is behavioural rather than structural — each kind is rendered at a
// width that forces the measurement to matter — so it fails if either switch
// stops accounting for a kind, not merely if the source stops mentioning it.
func TestEveryDocKindIsHandledByBothWalks(t *testing.T) {
	for _, tc := range []struct {
		name  string
		wide  string
		tight string
		doc   doc.Doc
	}{
		{name: "text", doc: doc.Text("abc"), wide: "abc", tight: "abc"},
		{name: "concat", doc: doc.Concat(doc.Text("a"), doc.Text("b")), wide: "ab", tight: "ab"},
		{name: "line", doc: doc.Group(doc.Concat(doc.Text("a"), doc.Line(), doc.Text("b"))), wide: "a b", tight: "a\nb"},
		{name: "softline", doc: doc.Group(doc.Concat(doc.Text("a"), doc.Softline(), doc.Text("b"))), wide: "ab", tight: "a\nb"},
		{name: "hardline", doc: doc.Group(doc.Concat(doc.Text("a"), doc.Hardline(), doc.Text("b"))), wide: "a\nb", tight: "a\nb"},
		{name: "indent", doc: doc.Group(doc.Indent(doc.Concat(doc.Text("a"), doc.Line(), doc.Text("b")))), wide: "a b", tight: "a\n  b"},
		{name: "group", doc: doc.Group(doc.Text("abc")), wide: "abc", tight: "abc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := doc.Render(tc.doc, 200); got != tc.wide {
				t.Errorf("wide: got %q, want %q", got, tc.wide)
			}
			if got := doc.Render(tc.doc, 1); got != tc.tight {
				t.Errorf("tight: got %q, want %q", got, tc.tight)
			}
		})
	}
}
