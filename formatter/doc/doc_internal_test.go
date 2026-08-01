package doc

import (
	"strings"
	"testing"
)

// TestRenderIgnoresAnUndeclaredKind covers the default arms of step and
// fitsStep. Every declared kind is now named explicitly in both switches, so
// the defaults are reachable only by constructing a kind the package does not
// declare — which is exactly what makes them worth keeping: a kind added to the
// enum without an arm in BOTH walks would otherwise fall into whichever
// behaviour the old catch-all happened to have, and the renderer and the
// measurement would silently disagree about it.
//
// Emitting nothing is the right answer for a kind with no defined meaning:
// guessing a newline or a space would put characters in the user's SQL that no
// document asked for.
func TestRenderIgnoresAnUndeclaredKind(t *testing.T) {
	undeclared := Doc{kind: kind(99), text: "should not appear"}

	if got := Render(undeclared, 80); got != "" {
		t.Errorf("an undeclared kind rendered %q, want nothing", got)
	}

	// Inside a group it must also contribute nothing to the fit measurement, or
	// the group's width would be computed from a node the renderer skips.
	wrapped := Group(Concat(Text("a"), undeclared, Text("b")))
	if got := Render(wrapped, 80); got != "ab" {
		t.Errorf("a group holding an undeclared kind rendered %q, want %q", got, "ab")
	}
	if got := Render(wrapped, 1); !strings.Contains(got, "a") || !strings.Contains(got, "b") {
		t.Errorf("a narrow group holding an undeclared kind rendered %q, losing content", got)
	}
}
