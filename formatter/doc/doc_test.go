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
