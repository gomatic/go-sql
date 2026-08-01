package compare

import "testing"

// TestPositionalFieldNamesExactlyTheKeysThatMoveWithFormatting names
// positionalField's claim. Stripping these five is what makes the comparison
// answer "are these the same statement" rather than "were these written the
// same way" — a reformatted-but-identical statement must compare equal. Strip
// one too few and every reindentation is a difference; strip one too many and a
// genuine change in that field goes unreported.
func TestPositionalFieldNamesExactlyTheKeysThatMoveWithFormatting(t *testing.T) {
	t.Parallel()

	want := map[positionalField]struct{}{
		positionalColno: {}, positionalLineno: {}, positionalLocation: {},
		positionalStmtLen: {}, positionalStmtLocation: {},
	}
	if len(positionalFields) != len(want) {
		t.Fatalf("positionalFields has %d entries, want %d", len(positionalFields), len(want))
	}
	for key := range want {
		if _, ok := positionalFields[key]; !ok {
			t.Errorf("%q moves with formatting and must be stripped", key)
		}
	}

	// A navigation key must NOT be stripped: those carry meaning, and removing
	// one would make two different statements compare equal.
	for _, key := range []positionalField{"relname", "sval", "stmt", "type"} {
		if _, ok := positionalFields[key]; ok {
			t.Errorf("%q carries meaning and must survive normalization", key)
		}
	}
}
