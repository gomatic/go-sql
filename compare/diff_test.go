package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeDiffs(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []computeDiffsCase{
		{
			name:        "equal_statements",
			source:      statementData{"name": "test"},
			target:      statementData{"name": "test"},
			expectDiffs: false,
		},
		{
			name:        "different_value",
			source:      statementData{"name": "a"},
			target:      statementData{"name": "b"},
			expectDiffs: true,
		},
		{
			name:        "missing_key_in_target",
			source:      statementData{"name": "test"},
			target:      statementData{},
			expectDiffs: true,
		},
		{
			name:        "extra_key_in_target",
			source:      statementData{},
			target:      statementData{"name": "test"},
			expectDiffs: true,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			diffs := computeDiffs(tt.source, tt.target)
			if tt.expectDiffs {
				want.NotEmpty(diffs)
			} else {
				want.Empty(diffs)
			}
		})
	}
}

func TestSlicesHaveSameElements(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []slicesHaveSameElementsCase{
		{name: "empty_slices", source: []any{}, target: []any{}, expected: true},
		{name: "same_order", source: []any{"a", "b", "c"}, target: []any{"a", "b", "c"}, expected: true},
		{name: "different_order", source: []any{"a", "b", "c"}, target: []any{"c", "b", "a"}, expected: true},
		{name: "different_length", source: []any{"a", "b"}, target: []any{"a", "b", "c"}, expected: false},
		{name: "different_elements", source: []any{"a", "b"}, target: []any{"a", "c"}, expected: false},
		{name: "duplicate_elements", source: []any{"a", "a", "b"}, target: []any{"a", "b", "a"}, expected: true},
		{
			name:     "maps_same_order",
			source:   []any{map[string]any{"name": "a"}, map[string]any{"name": "b"}},
			target:   []any{map[string]any{"name": "a"}, map[string]any{"name": "b"}},
			expected: true,
		},
		{
			name:     "maps_different_order",
			source:   []any{map[string]any{"name": "a"}, map[string]any{"name": "b"}},
			target:   []any{map[string]any{"name": "b"}, map[string]any{"name": "a"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			want.Equal(tt.expected, slicesHaveSameElements(tt.source, tt.target))
		})
	}
}

func TestComputeValueDiffs(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []computeValueDiffsCase{
		{name: "both_nil", source: nil, target: nil, expectDiffs: false},
		{name: "source_nil", source: nil, target: "value", expectDiffs: true},
		{name: "target_nil", source: "value", target: nil, expectDiffs: true},
		{name: "equal_strings", source: "test", target: "test", expectDiffs: false},
		{name: "different_strings", source: "a", target: "b", expectDiffs: true},
		{
			name:        "equal_maps",
			source:      map[string]any{"key": "value"},
			target:      map[string]any{"key": "value"},
			expectDiffs: false,
		},
		{
			name:        "different_maps",
			source:      map[string]any{"key": "a"},
			target:      map[string]any{"key": "b"},
			expectDiffs: true,
		},
		{name: "equal_slices", source: []any{"a", "b"}, target: []any{"a", "b"}, expectDiffs: false},
		{name: "different_slices", source: []any{"a"}, target: []any{"b"}, expectDiffs: true},
		{name: "type_mismatch_map_slice", source: map[string]any{}, target: []any{}, expectDiffs: true},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			diffs := computeValueDiffs("test", tt.source, tt.target, nil)
			if tt.expectDiffs {
				want.NotEmpty(diffs)
			} else {
				want.Empty(diffs)
			}
		})
	}
}

func TestComputeSliceDiffs(t *testing.T) {
	t.Parallel()
	want, _ := assert.New(t), require.New(t)

	tests := []computeSliceDiffsCase{
		{name: "identical", source: []any{"a", "b"}, target: []any{"a", "b"}, expectDiffs: false, expectOrder: false},
		{name: "different_length", source: []any{"a"}, target: []any{"a", "b"}, expectDiffs: true, expectOrder: false},
		{
			name:        "reordered_same_elements",
			source:      []any{"a", "b"},
			target:      []any{"b", "a"},
			expectDiffs: true,
			expectOrder: true,
		},
		{
			name:        "different_elements",
			source:      []any{"a", "b"},
			target:      []any{"c", "d"},
			expectDiffs: true,
			expectOrder: false,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			diffs := computeSliceDiffs("test", tt.source, tt.target, nil)
			if !tt.expectDiffs {
				want.Empty(diffs)
				return
			}
			want.NotEmpty(diffs)
			if tt.expectOrder {
				want.Len(diffs, 1)
				want.Contains(string(diffs[0].Field), "order")
			}
		})
	}
}
