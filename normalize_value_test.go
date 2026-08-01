package sql

import (
	"reflect"
	"testing"
)

// TestEmptiableKindCoversWhatDeepEqualCannotDecide names emptiableKind's claim.
// isZeroValue drops zero-valued fields before comparing two ASTs, and for most
// kinds "is it zero" is answered by DeepEqual against the type's zero. These
// kinds are the exceptions: an empty non-nil slice is NOT DeepEqual to a nil
// slice, so without the Len test an empty list would survive normalization in
// one AST and not the other, and two identical statements would compare
// different.
func TestEmptiableKindCoversWhatDeepEqualCannotDecide(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		value any
		why   string
		want  bool
	}{
		{value: nil, want: true, why: "an untyped nil"},
		{value: []any{}, want: true, why: "an empty slice"},
		{value: map[string]any{}, want: true, why: "an empty map"},
		{value: [0]int{}, want: true, why: "an empty array"},
		{value: "", want: true, why: "an empty string"},
		{value: (*int)(nil), want: true, why: "a nil pointer"},

		{value: 0, want: true, why: "a scalar zero is decided by DeepEqual, not by this switch"},
		{value: false, want: true, why: "and so is a false bool"},

		{value: 1, want: false, why: "a non-zero scalar is present"},
		{value: []any{1}, want: false, why: "a non-empty slice"},
		{value: map[string]any{"a": 1}, want: false, why: "a non-empty map"},
		{value: "x", want: false, why: "a non-empty string"},
		{value: new(int), want: false, why: "a non-nil pointer is present even though it points at zero"},
	} {
		if got := isZeroValue(tc.value); got != tc.want {
			t.Errorf("isZeroValue(%#v) = %v, want %v: %s", tc.value, got, tc.want, tc.why)
		}
	}
}

// TestScalarKindWidensEveryIntegerAndFloatWidth names scalarKind's claim. Two
// ASTs that differ only in the width a field was decoded at describe the same
// statement, so every integer width must collapse to int64 and every float to
// float64 — otherwise an int32 3 and an int64 3 compare unequal.
func TestScalarKindWidensEveryIntegerAndFloatWidth(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		value any
		want  any
		why   string
	}{
		{value: int8(3), want: int64(3), why: "int8 widens"},
		{value: int32(3), want: int64(3), why: "int32 widens"},
		{value: int64(3), want: int64(3), why: "int64 is already canonical"},
		{value: uint8(3), want: uint64(3), why: "uint8 widens"},
		{value: uint64(3), want: uint64(3), why: "uint64 is already canonical"},
		{value: float32(1.5), want: 1.5, why: "float32 widens"},
		{value: "x", want: "x", why: "a string is itself"},
		{value: true, want: true, why: "and so is a bool"},
	} {
		if got := scalarValue(reflect.ValueOf(tc.value)); got != tc.want {
			t.Errorf("scalarValue(%#v) = %#v, want %#v: %s", tc.value, got, tc.want, tc.why)
		}
	}
}
