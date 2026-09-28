package nodeset

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// SplitOnComma will split the input string on commas except for when within square brackets.
// Used for pre-processing input strings for Expand when such input has multiple node patterns
// seperated by comma like 'node[1-2],node[5-6]'
func SplitOnComma(s string) []string {
	var result []string
	var buffer strings.Builder
	inBrackets := 0

	for _, char := range s {
		switch char {
		case '[':
			inBrackets++
			buffer.WriteRune(char)
		case ']':
			inBrackets--
			buffer.WriteRune(char)
		case ',':
			if inBrackets > 0 {
				buffer.WriteRune(char)
			} else {
				result = append(result, buffer.String())
				buffer.Reset()
			}
		default:
			buffer.WriteRune(char)
		}
	}

	result = append(result, buffer.String())
	return result
}

// Expand takes a node set pattern like 'node[1-2]', and a function
// with the signature func(s string). It will parse the pattern
// string and calculate the numerical ranges from the pattern.
// It will then create the Cartesian product for the pattern, for example:
// rack[1-2]node[3-4] ->
// rack1node3, rack1node4, rack2node3, rack2node4.
// Addition pattern syntax supported:
// Union ranges - node[1-2,5-9]
// Step ranges - node[1-4/2]
// The supplied iter function is called per Cartesian product.
func Expand(pattern string, iter func(s string) error) error {
	if pattern == "" {
		return fmt.Errorf("empty pattern")
	}
	if iter == nil {
		return fmt.Errorf("iter function nil")
	}
	ranges, err := splitInput(pattern)
	if err != nil {
		return err
	}

	// ix holds the current index into each dimension's value slice. A single
	// strings.Builder is reused across iterations to avoid per-node slice and
	// string allocations in the Cartesian walk.
	ix := make([]int, len(ranges))
	var b strings.Builder

	for ix[0] < len(ranges[0]) {
		b.Reset()
		for j, k := range ix {
			b.WriteString(ranges[j][k])
		}
		if err := iter(b.String()); err != nil {
			return err
		}
		nextIndex(ix, ranges)
	}
	return nil
}

// nextIndex advances ix to the next Cartesian combination, incrementing the
// rightmost dimension first and carrying over into earlier dimensions. When
// the highest combination is reached, ix[0] is left equal to len(ranges[0])
// to terminate the caller's loop.
func nextIndex(ix []int, ranges [][]string) {
	for j := len(ix) - 1; j >= 0; j-- {
		ix[j]++
		if j == 0 || ix[j] < len(ranges[j]) {
			return
		}
		ix[j] = 0
	}
}

func splitInput(input string) ([][]string, error) {
	var ranges [][]string

	for input != "" {
		if input[0] == '[' {
			end := 0
			for ; end < len(input) && input[end] != ']'; end++ {
				if end != 0 && input[end] == '[' {
					return [][]string{}, fmt.Errorf("input %s, contains a nested left bracket", input)
				}
			}
			if end == len(input) || input[end] != ']' {
				return [][]string{}, fmt.Errorf("input %s, contains a left bracket without a right bracket", input)
			}
			set, err := parseRange(input[:end+1])
			if err != nil {
				return [][]string{}, err
			}
			ranges = append(ranges, set)
			input = input[end+1:]
		} else {
			end := 0
			for ; end < len(input) && input[end] != '['; end++ {
				if input[end] == ']' {
					return [][]string{}, fmt.Errorf("input %s, contains a right bracket without a left bracket", input)
				}
			}

			ranges = append(ranges, []string{input[:end]})
			input = input[end:]
		}
	}
	return ranges, nil
}

// parseRange takes a string in the form of [1], [1-2], or [1-4/2]
// The returned range sets are deduplicated and numeric sorted.
func parseRange(rangeStr string) ([]string, error) {
	// value pairs the numeric value with its zero-padding width so sorting and
	// deduplication can operate on integers rather than re-parsing strings.
	type value struct {
		v       uint64
		padding int
	}
	var values []value

	// Remove brackets from the range string
	if len(rangeStr) > 1 && rangeStr[0] == '[' && rangeStr[len(rangeStr)-1] == ']' {
		rangeStr = rangeStr[1 : len(rangeStr)-1]
	} else {
		return []string{}, fmt.Errorf("range [%s], is missing enclosing brackets", rangeStr)
	}

	// Split the range string by ','
	for _, index := range strings.Split(rangeStr, ",") {
		index, step, err := parseStep(index)
		if err != nil {
			return []string{}, err
		}

		rangeSplit := strings.Split(index, "-")

		if len(rangeSplit) == 1 {
			if step != 0 {
				return []string{}, fmt.Errorf("range [%s], contains a step without a start and stop range", index)
			}
			val, err := strconv.ParseUint(rangeSplit[0], 10, 64)
			if err != nil {
				return []string{}, fmt.Errorf("range [%s], contains a single value that is not an integer", index)
			}
			values = append(values, value{v: val})
		} else if len(rangeSplit) == 2 {
			start, err := strconv.ParseUint(rangeSplit[0], 10, 64)
			if err != nil {
				return []string{}, fmt.Errorf("range [%s], start with a value that is not an integer", index)
			}
			end, err := strconv.ParseUint(rangeSplit[1], 10, 64)
			if err != nil {
				return []string{}, fmt.Errorf("range [%s], ends with a value that is not an integer", index)
			}

			if start > end {
				return []string{}, fmt.Errorf("range [%s], starts with a value that is greater than the end value", index)
			}

			// If range start value has more than two characters and has a leading zero, assume that the output
			// should be padded to the same length as the start value.
			var padding int
			if len(rangeSplit[0]) > 1 && rangeSplit[0][0] == '0' {
				if len(rangeSplit[0]) > len(rangeSplit[1]) {
					return []string{}, fmt.Errorf("range [%s], zero padding on start value greater than end value length", index)
				}
				if rangeSplit[1][0] == '0' && (len(rangeSplit[0]) != len(rangeSplit[1])) {
					return []string{}, fmt.Errorf("range [%s], zero padding on end value must be same length as start value", index)
				}
				padding = len(rangeSplit[0])
			}

			// If step is its zero-value, default to incrementing by 1.
			if step == 0 {
				step = 1
			}

			// Preallocate for this segment's values.
			if cap(values)-len(values) < int((end-start)/step)+1 {
				grown := make([]value, len(values), len(values)+int((end-start)/step)+1)
				copy(grown, values)
				values = grown
			}
			for i := start; ; i += step {
				values = append(values, value{v: i, padding: padding})
				// Stop before i += step would exceed end or overflow uint64.
				// end - i cannot underflow here since i <= end always holds.
				if end-i < step {
					break
				}
			}
		}
	}

	// Sort numerically and deduplicate on the integer value before formatting,
	// avoiding the need to re-parse formatted strings.
	slices.SortStableFunc(values, func(a, b value) int {
		return cmp.Compare(a.v, b.v)
	})
	values = slices.CompactFunc(values, func(a, b value) bool {
		return a.v == b.v && a.padding == b.padding
	})

	// Format the deduplicated values, padding once per value.
	rangeValues := make([]string, len(values))
	var b strings.Builder
	var num [20]byte
	for i, val := range values {
		b.Reset()
		writeUintPadded(&b, num[:], val.v, val.padding)
		rangeValues[i] = b.String()
	}
	return rangeValues, nil
}

// writeUintPadded writes v to b, left-padded with zeros to at least width
// digits. scratch is a caller-provided buffer (>= 20 bytes) reused to avoid
// allocation.
func writeUintPadded(b *strings.Builder, scratch []byte, v uint64, width int) {
	digits := strconv.AppendUint(scratch[:0], v, 10)
	for pad := width - len(digits); pad > 0; pad-- {
		b.WriteByte('0')
	}
	b.Write(digits)
}

func parseStep(rangeStr string) (string, uint64, error) {
	var step uint64
	stepSplit := strings.Split(rangeStr, "/")
	if len(stepSplit) > 2 {
		return "", 0, fmt.Errorf("range [%s], contains more than one step delineator '/'", rangeStr)
	} else if len(stepSplit) == 2 {
		var err error
		step, err = strconv.ParseUint(stepSplit[1], 10, 64)
		if err != nil {
			return "", 0, fmt.Errorf("range [%s], contains a step that is not an integer", rangeStr)
		}
	}
	return stepSplit[0], step, nil
}
