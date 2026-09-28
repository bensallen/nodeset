package nodeset

import (
	"reflect"
	"testing"
)

func TestFold(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "No digits, single entry",
			input:    []string{"a"},
			expected: []string{"a"},
		},
		{
			name:     "Empty string input",
			input:    []string{""},
			expected: []string{""},
		},
		{
			name:     "Leading digits",
			input:    []string{"0g", "1g"},
			expected: []string{"[0-1]g"},
		},
		{
			name:     "Trailing digits",
			input:    []string{"g0", "g1"},
			expected: []string{"g[0-1]"},
		},
		{
			name:     "Duplicates",
			input:    []string{"g1", "g1", "g01"},
			expected: []string{"g[1,01]"},
		},
		{
			name:     "Range with gap",
			input:    []string{"a0c", "a1c", "a2c", "a4c"},
			expected: []string{"a[0-2,4]c"},
		},
		{
			name:     "Multiple isolated singles",
			input:    []string{"a0", "a2", "a4", "a6"},
			expected: []string{"a[0,2,4,6]"},
		},
		{
			name:     "Range with padding",
			input:    []string{"j0001", "j0002"},
			expected: []string{"j[0001-0002]"},
		},
		{
			name:     "Multiple ranges",
			input:    []string{"eh1f0", "eh1f1", "eh2f0", "eh2f1"},
			expected: []string{"eh[1-2]f[0-1]"},
		},
		{
			name:     "Multiple ranges, with mismatching second range",
			input:    []string{"eh1f0", "eh1f1", "eh2f0", "eh2f1", "eh2f4"},
			expected: []string{"eh1f[0-1]", "eh2f[0-1,4]"},
		},
		{
			name:     "Digits increasing in length",
			input:    []string{"k9", "k10"},
			expected: []string{"k[9-10]"},
		},
		{
			name:     "Large values beyond int32 range",
			input:    []string{"n4294967295", "n4294967296", "n4294967297"},
			expected: []string{"n[4294967295-4294967297]"},
		},
		{
			name:     "Mixed padding shares a bracket",
			input:    []string{"k2", "k03", "k004"},
			expected: []string{"k[2,03,004]"},
		},
		{
			name:     "Unpadded value joins matching pad-width class",
			input:    []string{"a01", "a02", "a10"},
			expected: []string{"a[01-02,10]"},
		},
		{
			name:     "Padded range spanning into unpadded width",
			input:    []string{"a08", "a09", "a10", "a11"},
			expected: []string{"a[08-11]"},
		},
		{
			name:     "Separate pad-width classes ordered by width",
			input:    []string{"a007", "a008", "a9", "a10"},
			expected: []string{"a[9-10,007-008]"},
		},
		{
			name:     "Passthrough node with fewer dimensions",
			input:    []string{"node1", "node2", "other"},
			expected: []string{"node[1-2]", "other"},
		},
		{
			name:     "Multi-dimensional fold",
			input:    []string{"rack1node1", "rack1node2", "rack2node1"},
			expected: []string{"rack1node[1-2]", "rack2node1"},
		},
		{
			name:     "Five-dimensional Cray names fold innermost",
			input:    []string{"x1000c0s0b0n0", "x1000c0s0b0n1", "x1000c0s0b0n2", "x1000c0s0b0n3"},
			expected: []string{"x1000c0s0b0n[0-3]"},
		},
		{
			name:     "Five-dimensional Cray names fold multiple dimensions",
			input:    []string{"x1000c0s0b0n0", "x1000c0s0b0n1", "x1000c0s1b0n0", "x1000c0s1b0n1"},
			expected: []string{"x1000c0s[0-1]b0n[0-1]"},
		},
		{
			name:     "Five-dimensional Cray names with mismatched outer dimension",
			input:    []string{"x1000c0s0b0n0", "x1000c0s0b0n1", "x1000c1s0b0n0", "x1000c1s0b0n1", "x1001c0s0b0n0"},
			expected: []string{"x1000c[0-1]s0b0n[0-1]", "x1001c0s0b0n0"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Fold(tc.input)
			//slices.Sort[[]string](result)

			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, but got %v", tc.expected, result)
			}
		})
	}
}

func TestNumericRangeEmpty(t *testing.T) {
	ranges, bracket := numericRange(nil, 0)
	if len(ranges) != 0 || bracket {
		t.Errorf("Expected empty, non-bracketed result, got %v %v", ranges, bracket)
	}
}

func TestNodeString(t *testing.T) {
	// A node retaining a raw, zero-padded digit token renders with padding.
	n := node{{isDigit: false, literal: "x"}, {isDigit: true, value: 5, width: 3}}
	if got := n.String(); got != "x005" {
		t.Errorf("Expected x005, got %q", got)
	}
}

func TestSplitTokens(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected node
	}{
		{
			name:     "No digits",
			input:    "a",
			expected: node{{literal: "a"}},
		},
		{
			name:     "Leading digits",
			input:    "0g",
			expected: node{{isDigit: true, value: 0, width: 1}, {literal: "g"}},
		},
		{
			name:     "Trailing multiple digits",
			input:    "j0001",
			expected: node{{literal: "j"}, {isDigit: true, value: 1, width: 4}},
		},
		{
			name:     "Multiple consective digits in middle",
			input:    "j0001h",
			expected: node{{literal: "j"}, {isDigit: true, value: 1, width: 4}, {literal: "h"}},
		},
		{
			name:  "Multiple digit ranges",
			input: "eh1f0h0",
			expected: node{
				{literal: "eh"}, {isDigit: true, value: 1, width: 1},
				{literal: "f"}, {isDigit: true, value: 0, width: 1},
				{literal: "h"}, {isDigit: true, value: 0, width: 1},
			},
		},
		{
			name:     "Digit run overflowing uint64 kept as literal",
			input:    "n18446744073709551616",
			expected: node{{literal: "n"}, {literal: "18446744073709551616"}},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: node{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitTokens(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, but got %v", tc.expected, result)
			}
		})
	}
}
