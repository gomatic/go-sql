package formatter

import "testing"

func TestPreservesMeaningAcceptsReformat(t *testing.T) {
	if !preservesMeaning("SELECT A,B FROM T", "select a\n, b\nfrom t") {
		t.Fatal("a pure reformat should preserve meaning")
	}
}

func TestPreservesMeaningRejectsChangedMeaning(t *testing.T) {
	if preservesMeaning("select a from t", "select b from t") {
		t.Fatal("a different column changes meaning")
	}
}

func TestPreservesMeaningRejectsDroppedComment(t *testing.T) {
	if preservesMeaning("-- note\nselect a from t", "select a from t") {
		t.Fatal("dropping a comment must be rejected")
	}
}

func TestPreservesMeaningRejectsAddedComment(t *testing.T) {
	if preservesMeaning("select a from t", "select a /* added */\nfrom t") {
		t.Fatal("adding a comment must be rejected")
	}
}

func TestPreservesMeaningRejectsUnparsableCandidate(t *testing.T) {
	if preservesMeaning("select a from t", "select a from") {
		t.Fatal("an unparsable candidate must be rejected")
	}
}

func TestChooseFormattedPicksFirstSafeCandidate(t *testing.T) {
	got := chooseFormatted("select a from t", "select bad from", "select a\nfrom t")
	if got != "select a\nfrom t" {
		t.Fatalf("got %q, want the second (safe) candidate", got)
	}
}

func TestChooseFormattedFallsBackToOriginalWhenNoneSafe(t *testing.T) {
	got := chooseFormatted("-- keep\nselect a from t", "select a from t")
	if got != "-- keep\nselect a from t" {
		t.Fatalf("got %q, want the verbatim original", got)
	}
}

func TestPreservesMeaningRejectsUnparsableOriginal(t *testing.T) {
	if preservesMeaning("not sql ((", "select 1") {
		t.Fatal("an unparsable original can't be matched")
	}
}

func TestCommentsEqualRejectsUnscannableOriginal(t *testing.T) {
	if commentsEqual("'unterminated", "select 1") {
		t.Fatal("an unscannable original can't be compared")
	}
}

func TestCommentsEqualRejectsUnscannableCandidate(t *testing.T) {
	if commentsEqual("select 1", "'unterminated") {
		t.Fatal("an unscannable candidate can't be compared")
	}
}
