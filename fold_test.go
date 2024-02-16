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
			expected: []string{"g1", "g01"},
		},
		{
			name:     "Range with gap",
			input:    []string{"a0c", "a1c", "a2c", "a4c"},
			expected: []string{"a[0-2,4]c"},
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
			name:     "Digits increasing in length",
			input:    []string{"k9", "k10"},
			expected: []string{"k[9-10]"},
		},
		{
			name:     "Mixed padding, folding not supported",
			input:    []string{"k2", "k03", "k004"},
			expected: []string{"k2", "k03", "k004"},
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

func TestSplitOnDigits(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "No digits",
			input:    "a",
			expected: []string{"a"},
		},
		{
			name:     "Leading digits",
			input:    "0g",
			expected: []string{"0", "g"},
		},
		{
			name:     "Trailing multiple digits",
			input:    "j0001",
			expected: []string{"j", "0001"},
		},
		{
			name:     "Multiple consective digits in middle",
			input:    "j0001h",
			expected: []string{"j", "0001", "h"},
		},
		{
			name:     "Multiple digit ranges",
			input:    "eh1f0h0",
			expected: []string{"eh", "1", "f", "0", "h", "0"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitOnDigits(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, but got %v", tc.expected, result)
			}
		})
	}
}
